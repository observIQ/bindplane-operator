/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

func intOrStrPtr(v intstr.IntOrString) *intstr.IntOrString {
	return &v
}

var _ = Describe("newPodDisruptionBudget", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = newTestBindplane("my-bp", "default")
	})

	It("defaults to minAvailable: 1 when spec is nil", func() {
		pdb := newPodDisruptionBudget(bindplane, nodeComponent, nil)
		Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))
		Expect(pdb.Spec.MaxUnavailable).To(BeNil())
	})

	It("defaults to minAvailable: 1 when spec sets neither field", func() {
		pdb := newPodDisruptionBudget(bindplane, nodeComponent, &bindplanev1alpha1.PodDisruptionBudgetSpec{})
		Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))
		Expect(pdb.Spec.MaxUnavailable).To(BeNil())
	})

	It("uses maxUnavailable when set", func() {
		spec := &bindplanev1alpha1.PodDisruptionBudgetSpec{MaxUnavailable: intOrStrPtr(intstr.FromString("25%"))}
		pdb := newPodDisruptionBudget(bindplane, opampComponent, spec)
		Expect(pdb.Spec.MinAvailable).To(BeNil())
		Expect(pdb.Spec.MaxUnavailable).To(Equal(intOrStrPtr(intstr.FromString("25%"))))
	})

	It("uses minAvailable when set", func() {
		spec := &bindplanev1alpha1.PodDisruptionBudgetSpec{MinAvailable: intOrStrPtr(intstr.FromInt32(2))}
		pdb := newPodDisruptionBudget(bindplane, natsComponent, spec)
		Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(2))))
		Expect(pdb.Spec.MaxUnavailable).To(BeNil())
	})

	It("does not alias the spec values", func() {
		spec := &bindplanev1alpha1.PodDisruptionBudgetSpec{MaxUnavailable: intOrStrPtr(intstr.FromInt32(1))}
		pdb := newPodDisruptionBudget(bindplane, natsComponent, spec)
		Expect(pdb.Spec.MaxUnavailable).NotTo(BeIdenticalTo(spec.MaxUnavailable))
	})
})

var _ = Describe("Reconcile - PodDisruptionBudget", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-pdb")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	getPDB := func(name string) *policyv1.PodDisruptionBudget {
		pdb := &policyv1.PodDisruptionBudget{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, pdb)).To(Succeed())
		return pdb
	}

	updateSpec := func(r *BindplaneReconciler, name string, mutate func(bp *bindplanev1alpha1.Bindplane)) {
		bp := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, bp)).To(Succeed())
		mutate(bp)
		Expect(k8sClient.Update(testCtx, bp)).To(Succeed())
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
	}

	It("creates PDBs with minAvailable: 1 when podDisruptionBudget is unset", func() {
		name := "bp-pdb-default"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		for _, component := range []string{nodeComponent, opampComponent, natsComponent, transformAgentComponent} {
			pdb := getPDB(name + "-" + component)
			Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))), component)
			Expect(pdb.Spec.MaxUnavailable).To(BeNil(), component)
		}
	})

	It("applies maxUnavailable and minAvailable per component", func() {
		name := "bp-pdb-custom"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		bp.Spec.OpAMP.PodDisruptionBudget = &bindplanev1alpha1.PodDisruptionBudgetSpec{
			MaxUnavailable: intOrStrPtr(intstr.FromInt32(2)),
		}
		bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{
			PodDisruptionBudget: &bindplanev1alpha1.PodDisruptionBudgetSpec{
				MinAvailable: intOrStrPtr(intstr.FromString("50%")),
			},
		}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		opampPDB := getPDB(name + "-opamp")
		Expect(opampPDB.Spec.MinAvailable).To(BeNil())
		Expect(opampPDB.Spec.MaxUnavailable).To(Equal(intOrStrPtr(intstr.FromInt32(2))))

		natsPDB := getPDB(name + "-nats")
		Expect(natsPDB.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromString("50%"))))
		Expect(natsPDB.Spec.MaxUnavailable).To(BeNil())

		nodePDB := getPDB(name + "-node")
		Expect(nodePDB.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))
	})

	It("updates an existing PDB when switching between minAvailable and maxUnavailable", func() {
		name := "bp-pdb-switch"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)
		Expect(getPDB(name + "-node").Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))

		updateSpec(r, name, func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.Bindplane.PodDisruptionBudget = &bindplanev1alpha1.PodDisruptionBudgetSpec{
				MaxUnavailable: intOrStrPtr(intstr.FromInt32(1)),
			}
		})
		pdb := getPDB(name + "-node")
		Expect(pdb.Spec.MinAvailable).To(BeNil())
		Expect(pdb.Spec.MaxUnavailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))

		updateSpec(r, name, func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.Bindplane.PodDisruptionBudget = &bindplanev1alpha1.PodDisruptionBudgetSpec{
				MinAvailable: intOrStrPtr(intstr.FromInt32(2)),
			}
		})
		pdb = getPDB(name + "-node")
		Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(2))))
		Expect(pdb.Spec.MaxUnavailable).To(BeNil())

		updateSpec(r, name, func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.Bindplane.PodDisruptionBudget = nil
		})
		pdb = getPDB(name + "-node")
		Expect(pdb.Spec.MinAvailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))
		Expect(pdb.Spec.MaxUnavailable).To(BeNil())
	})

	It("deletes the PDB when disablePodDisruptionBudget is true even with podDisruptionBudget set", func() {
		name := "bp-pdb-disabled"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{
			PodDisruptionBudget: &bindplanev1alpha1.PodDisruptionBudgetSpec{
				MaxUnavailable: intOrStrPtr(intstr.FromInt32(1)),
			},
		}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)
		Expect(getPDB(name + "-nats").Spec.MaxUnavailable).To(Equal(intOrStrPtr(intstr.FromInt32(1))))

		updateSpec(r, name, func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.Nats.DisablePodDisruptionBudget = true
		})
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-nats", Namespace: testNamespace}, &policyv1.PodDisruptionBudget{})
		Expect(errors.IsNotFound(err)).To(BeTrue(), "PDB should be deleted when disabled")
	})

	It("rejects a spec that sets both minAvailable and maxUnavailable", func() {
		bp := newTestBindplane("bp-pdb-both", testNamespace)
		bp.Spec.Bindplane.PodDisruptionBudget = &bindplanev1alpha1.PodDisruptionBudgetSpec{
			MinAvailable:   intOrStrPtr(intstr.FromInt32(1)),
			MaxUnavailable: intOrStrPtr(intstr.FromInt32(1)),
		}
		err := k8sClient.Create(testCtx, bp)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("minAvailable and maxUnavailable are mutually exclusive"))
	})
})
