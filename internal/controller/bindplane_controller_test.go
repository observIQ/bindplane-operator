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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// conditionTypeReconciled is the condition type used by the controller to report reconcile status.
const conditionTypeReconciled = "Reconciled"

// newTestBindplane returns a minimal valid Bindplane CR for integration tests.
func newTestBindplane(name, namespace string) *bindplanev1alpha1.Bindplane {
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.98.0",
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				License: "test-license",
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{
						Host: "postgres.postgres.svc.cluster.local",
					},
				},
			},
		},
	}
}

// newReconciler returns a BindplaneReconciler wired to the envtest k8sClient.
func newReconciler() *BindplaneReconciler {
	return &BindplaneReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

// createTestNamespace creates a namespace with a generated name and returns the name.
func createTestNamespace(ctx context.Context, prefix string) string {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix + "-",
		},
	}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns.Name
}

// reconcileRequest builds a reconcile.Request for the given name and namespace.
func reconcileRequest(name, namespace string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}}
}

// markJobComplete patches the migrate Job's status so it appears succeeded.
// Kubernetes 1.35+ requires startTime, completionTime, and SuccessCriteriaMet
// to be set alongside the Complete condition.
func markJobComplete(ctx context.Context, jobName, namespace string) {
	job := &batchv1.Job{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: jobName, Namespace: namespace}, job)).To(Succeed())
	now := metav1.Now()
	job.Status.StartTime = &now
	job.Status.CompletionTime = &now
	job.Status.Conditions = append(job.Status.Conditions,
		batchv1.JobCondition{
			Type:               batchv1.JobSuccessCriteriaMet,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		},
		batchv1.JobCondition{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		},
	)
	Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())
}

// markJobFailed patches the migrate Job's status so it appears failed.
// Kubernetes 1.35+ requires FailureTarget=true before Failed=true.
func markJobFailed(ctx context.Context, jobName, namespace string) {
	job := &batchv1.Job{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: jobName, Namespace: namespace}, job)).To(Succeed())
	now := metav1.Now()
	job.Status.StartTime = &now
	job.Status.Conditions = append(job.Status.Conditions,
		batchv1.JobCondition{
			Type:               batchv1.JobFailureTarget,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		},
		batchv1.JobCondition{
			Type:               batchv1.JobFailed,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		},
	)
	Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())
}

// reconcileUntilMigration performs the reconciles needed to create the migrate Job
// (finalizer reconcile + full reconcile) and returns the Job name.
func reconcileUntilMigration(ctx context.Context, r *BindplaneReconciler, bpName, namespace string) string {
	// Reconcile 1: add finalizer
	_, err := r.Reconcile(ctx, reconcileRequest(bpName, namespace))
	Expect(err).NotTo(HaveOccurred())
	// Reconcile 2: full path, creates resources up to migrate job
	result, err := r.Reconcile(ctx, reconcileRequest(bpName, namespace))
	Expect(err).NotTo(HaveOccurred())
	Expect(result.RequeueAfter).NotTo(BeZero(), "expected RequeueAfter while awaiting migration")
	return bpName + "-migrate"
}

// reconcilePastMigration performs the reconciles to get past the migrate Job gate.
func reconcilePastMigration(ctx context.Context, r *BindplaneReconciler, bpName, namespace string) {
	jobName := reconcileUntilMigration(ctx, r, bpName, namespace)
	markJobComplete(ctx, jobName, namespace)
	_, err := r.Reconcile(ctx, reconcileRequest(bpName, namespace))
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("Reconcile - finalizer lifecycle", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-finalizer")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("adds finalizer on first reconcile", func() {
		name := "bp-finalizer"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		result, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeZero())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		Expect(updated.Finalizers).To(ContainElement("k8s.bindplane.com/finalizer"))
	})

	It("removes finalizer on deletion", func() {
		name := "bp-finalizer-del"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		// First reconcile: adds finalizer
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		// Delete the CR — sets DeletionTimestamp (finalizer prevents immediate deletion)
		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		Expect(k8sClient.Delete(testCtx, updated)).To(Succeed())

		// Second reconcile: removes finalizer, allowing GC to delete the object
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		final := &bindplanev1alpha1.Bindplane{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, final)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})

var _ = Describe("Reconcile - pause annotation", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-pause")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("pauses reconciliation and sets Reconciled=False/Paused", func() {
		name := "bp-pause"
		bp := newTestBindplane(name, testNamespace)
		bp.Annotations = map[string]string{"k8s.bindplane.com/pause-reconciliation": "true"}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		// Reconcile 1: finalizer added (pause check happens after finalizer)
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		// Reconcile 2: hits pause path
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())

		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionFalse))
		Expect(reconciledCond.Reason).To(Equal("Paused"))

		// Verify no workloads were created
		depList := &appsv1.DeploymentList{}
		Expect(k8sClient.List(testCtx, depList, client.InNamespace(testNamespace))).To(Succeed())
		Expect(depList.Items).To(BeEmpty())

		ssList := &appsv1.StatefulSetList{}
		Expect(k8sClient.List(testCtx, ssList, client.InNamespace(testNamespace))).To(Succeed())
		Expect(ssList.Items).To(BeEmpty())
	})

	It("resumes when annotation removed", func() {
		name := "bp-resume"
		bp := newTestBindplane(name, testNamespace)
		bp.Annotations = map[string]string{"k8s.bindplane.com/pause-reconciliation": "true"}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		// Add finalizer
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		// Hit pause
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		// Remove pause annotation
		paused := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, paused)).To(Succeed())
		paused.Annotations = map[string]string{}
		Expect(k8sClient.Update(testCtx, paused)).To(Succeed())

		// Reconcile: should proceed past pause and start creating resources
		// (will block at migrate Job, but TA/TSDB resources are created)
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		// Transform agent ServiceAccount must now exist (proving reconciliation resumed)
		sa := &corev1.ServiceAccount{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-transform-agent", Namespace: testNamespace}, sa)
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Reconcile - validation failure", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-invalid")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("sets Reconciled=False/Invalid for invalid status key (non-UUID)", func() {
		// CRD enforces license and postgres presence, so we test a controller-only
		// validation: status.keys entries must be valid UUIDs.
		name := "bp-invalid-uuid"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.Config.Status = &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			Keys:    []string{"not-a-valid-uuid"},
		}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		// Reconcile 1: add finalizer
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		// Reconcile 2: validation fails
		result, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeZero())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())

		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionFalse))
		Expect(reconciledCond.Reason).To(Equal("Invalid"))
		Expect(reconciledCond.Message).NotTo(BeEmpty())

		// No workloads created
		depList := &appsv1.DeploymentList{}
		Expect(k8sClient.List(testCtx, depList, client.InNamespace(testNamespace))).To(Succeed())
		Expect(depList.Items).To(BeEmpty())
	})

	It("sets Reconciled=False/Invalid for empty postgres host", func() {
		// Postgres field must be present (CRD enforces this), but host can be empty —
		// the controller rejects empty host via ValidatePostgresConfig.
		name := "bp-invalid-pghost"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.Config.Store.Postgres = &bindplanev1alpha1.PostgresConfig{Host: ""}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		result, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeZero())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())

		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionFalse))
		Expect(reconciledCond.Reason).To(Equal("Invalid"))
	})
})

var _ = Describe("Reconcile - Transform Agent", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-ta")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates ServiceAccount, Deployment, Service, PDB", func() {
		name := "bp-ta"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		// Reconcile 1: finalizer
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		// Reconcile 2: full (blocks at migration)
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		saName := name + "-transform-agent"
		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: saName, Namespace: testNamespace}, sa)).To(Succeed())

		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: saName, Namespace: testNamespace}, dep)).To(Succeed())
		Expect(dep.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/observiq/bindplane-transform-agent:1.98.0-bindplane"))

		svc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: saName, Namespace: testNamespace}, svc)).To(Succeed())
		var foundPort bool
		for _, p := range svc.Spec.Ports {
			if p.Port == 4568 {
				foundPort = true
				break
			}
		}
		Expect(foundPort).To(BeTrue(), "expected service port 4568")

		pdb := &policyv1.PodDisruptionBudget{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: saName, Namespace: testNamespace}, pdb)).To(Succeed())
		Expect(pdb.Spec.MinAvailable).NotTo(BeNil())
	})

	It("uses custom replicas when set", func() {
		name := "bp-ta-replicas"
		bp := newTestBindplane(name, testNamespace)
		customReplicas := int32(3)
		bp.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{Replicas: &customReplicas}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-transform-agent", Namespace: testNamespace}, dep)).To(Succeed())
		Expect(dep.Spec.Replicas).NotTo(BeNil())
		Expect(*dep.Spec.Replicas).To(Equal(int32(3)))
	})

	It("skips PDB when disablePodDisruptionBudget is true", func() {
		name := "bp-ta-nopdb"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{
			DisablePodDisruptionBudget: true,
		}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		pdb := &policyv1.PodDisruptionBudget{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-transform-agent", Namespace: testNamespace}, pdb)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("mounts TLS cert-manager secret and env vars when Transform Agent TLS is enabled", func() {
		bp := newTestBindplane("bp-ta-tls", testNamespace)
		replicas := int32(2)
		bp.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{
			Replicas: &replicas,
			TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
				CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
			},
		}
		dep := newReconciler().transformAgentDeployment(bp)
		Expect(dep.Spec.Template.Spec.Volumes).To(ContainElement(
			HaveField("Name", Equal(internalTLSTransformAgentVolumeName)),
		))
		Expect(dep.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(
			And(
				HaveField("Name", Equal(internalTLSTransformAgentVolumeName)),
				HaveField("MountPath", Equal(internalTLSTransformAgentMountPath)),
			),
		))
		envVars := dep.Spec.Template.Spec.Containers[0].Env
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSCertEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSKeyEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/tls.key"))
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSCAEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/ca.crt"))
	})
})

var _ = Describe("Reconcile - TSDB", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-tsdb")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates ServiceAccount, StatefulSet, Service, basic-auth Secret", func() {
		name := "bp-tsdb"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		tsdbName := name + "-tsdb"
		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: tsdbName, Namespace: testNamespace}, sa)).To(Succeed())

		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: tsdbName, Namespace: testNamespace}, ss)).To(Succeed())
		Expect(ss.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/observiq/bindplane-prometheus:1.98.0"))

		svc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: tsdbName, Namespace: testNamespace}, svc)).To(Succeed())

		secret := &corev1.Secret{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-tsdb-basic-auth", Namespace: testNamespace}, secret)).To(Succeed())
		Expect(secret.Data).To(HaveKey("username"))
		Expect(secret.Data).To(HaveKey("password"))
		Expect(secret.Data).To(HaveKey("web-config"))
	})

	It("StatefulSet has correct volume claims", func() {
		name := "bp-tsdb-pvc"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-tsdb", Namespace: testNamespace}, ss)).To(Succeed())

		// VolumeClaimTemplate name is <bindplane-name>-tsdb-data (from getResourceName)
		expectedVCTName := name + "-tsdb-data"
		var foundPVC bool
		for _, vct := range ss.Spec.VolumeClaimTemplates {
			if vct.Name == expectedVCTName {
				foundPVC = true
				break
			}
		}
		Expect(foundPVC).To(BeTrue(), "expected VolumeClaimTemplate named %s", expectedVCTName)
	})
})

var _ = Describe("Reconcile - Jobs Migrate", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-migrate")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates migrate Job and returns RequeueAfter when Job is pending", func() {
		name := "bp-migrate"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		result, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).NotTo(BeZero())

		job := &batchv1.Job{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-migrate", Namespace: testNamespace}, job)).To(Succeed())

		// Jobs and Node Deployments should not exist yet
		dep := &appsv1.Deployment{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-jobs", Namespace: testNamespace}, dep)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("proceeds past migration when Job is manually marked Complete", func() {
		name := "bp-migrate-done"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		markJobComplete(testCtx, name+"-migrate", testNamespace)

		result, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(BeZero())

		// Jobs Deployment should now exist
		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-jobs", Namespace: testNamespace}, dep)).To(Succeed())

		// Node Deployment should now exist
		nodeDep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, nodeDep)).To(Succeed())

		// NATS StatefulSet should now exist
		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-nats", Namespace: testNamespace}, ss)).To(Succeed())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionTrue))
	})

	It("sets MigrationFailed condition when Job fails", func() {
		name := "bp-migrate-fail"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())
		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		markJobFailed(testCtx, name+"-migrate", testNamespace)

		_, err = r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionFalse))
		Expect(reconciledCond.Reason).To(Equal("MigrationFailed"))

		dep := &appsv1.Deployment{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-jobs", Namespace: testNamespace}, dep)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})

var _ = Describe("Reconcile - NATS", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-nats")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates ServiceAccount, StatefulSet, client Service, cluster Service, PDB", func() {
		name := "bp-nats"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		natsName := name + "-nats"
		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: natsName, Namespace: testNamespace}, sa)).To(Succeed())

		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: natsName, Namespace: testNamespace}, ss)).To(Succeed())
		Expect(ss.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/observiq/bindplane-ee:1.98.0"))

		clientSvc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: natsName + "-client", Namespace: testNamespace}, clientSvc)).To(Succeed())

		clusterSvc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: natsName + "-cluster", Namespace: testNamespace}, clusterSvc)).To(Succeed())

		pdb := &policyv1.PodDisruptionBudget{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: natsName, Namespace: testNamespace}, pdb)).To(Succeed())
	})

	It("uses custom replicas", func() {
		name := "bp-nats-replicas"
		bp := newTestBindplane(name, testNamespace)
		customReplicas := int32(5)
		bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{Replicas: &customReplicas}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		ss := &appsv1.StatefulSet{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-nats", Namespace: testNamespace}, ss)).To(Succeed())
		Expect(ss.Spec.Replicas).NotTo(BeNil())
		Expect(*ss.Spec.Replicas).To(Equal(int32(5)))
	})
})

var _ = Describe("Reconcile - Node", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-node")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates ServiceAccount, Deployment, Service, PDB", func() {
		name := "bp-node"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		nodeName := name + "-node"
		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: nodeName, Namespace: testNamespace}, sa)).To(Succeed())

		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: nodeName, Namespace: testNamespace}, dep)).To(Succeed())
		Expect(dep.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/observiq/bindplane-ee:1.98.0"))

		svc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: nodeName, Namespace: testNamespace}, svc)).To(Succeed())

		pdb := &policyv1.PodDisruptionBudget{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: nodeName, Namespace: testNamespace}, pdb)).To(Succeed())
	})

	It("creates HPA when autoscaling is enabled", func() {
		name := "bp-node-hpa"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, hpa)).To(Succeed())
		Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(name + "-node"))
		Expect(hpa.Spec.MinReplicas).NotTo(BeNil())
		Expect(*hpa.Spec.MinReplicas).To(Equal(int32(2)))
		Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))
	})

	It("deletes HPA when autoscaling is disabled", func() {
		name := "bp-node-hpa-del"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		// Verify HPA exists
		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, hpa)).To(Succeed())

		// Disable autoscaling
		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		updated.Spec.Bindplane.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: false}
		Expect(k8sClient.Update(testCtx, updated)).To(Succeed())

		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		hpaAfter := &autoscalingv2.HorizontalPodAutoscaler{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, hpaAfter)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})

var _ = Describe("Reconcile - status", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-status")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("sets phase to ApplyingChanges when replicas are not ready", func() {
		name := "bp-status-phase"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		// In envtest, no pods run, so ready replicas are 0 => ApplyingChanges
		Expect(updated.Status.Phase).To(Equal("ApplyingChanges"))
		Expect(updated.Status.NodeReadyReplicas).To(BeNumerically("==", 0))
	})

	It("sets Reconciled=True on successful reconcile", func() {
		name := "bp-status-reconciled"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())

		var reconciledCond *metav1.Condition
		for i := range updated.Status.Conditions {
			if updated.Status.Conditions[i].Type == conditionTypeReconciled {
				reconciledCond = &updated.Status.Conditions[i]
				break
			}
		}
		Expect(reconciledCond).NotTo(BeNil())
		Expect(reconciledCond.Status).To(Equal(metav1.ConditionTrue))
		Expect(reconciledCond.Reason).To(Equal("Reconciled"))
		Expect(reconciledCond.ObservedGeneration).To(Equal(updated.Generation))
	})
})

var _ = Describe("Reconcile - idempotency", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-idempotent")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("second reconcile does not error and does not duplicate resources", func() {
		name := "bp-idempotent"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		// Count resources before second reconcile
		depsBefore := &appsv1.DeploymentList{}
		Expect(k8sClient.List(testCtx, depsBefore, client.InNamespace(testNamespace))).To(Succeed())
		countBefore := len(depsBefore.Items)

		// Second full reconcile
		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		depsAfter := &appsv1.DeploymentList{}
		Expect(k8sClient.List(testCtx, depsAfter, client.InNamespace(testNamespace))).To(Succeed())
		Expect(depsAfter.Items).To(HaveLen(countBefore))
	})
})

var _ = Describe("Reconcile - Bindplane Jobs", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-jobs")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("creates ServiceAccount and Deployment", func() {
		name := "bp-jobs"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		jobsName := name + "-jobs"
		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: jobsName, Namespace: testNamespace}, sa)).To(Succeed())

		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: jobsName, Namespace: testNamespace}, dep)).To(Succeed())
		Expect(dep.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/observiq/bindplane-ee:1.98.0"))
	})
})

var _ = Describe("newService", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bindplane",
				Namespace: "default",
			},
		}
	})

	Context("when creating a service with a single port", func() {
		It("should create a ClusterIP service with the correct port configuration", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service).NotTo(BeNil())
			Expect(service.Name).To(Equal("test-bindplane-test-component"))
			Expect(service.Namespace).To(Equal("default"))
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(8080)))
			Expect(service.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
		})

		It("should have correct labels", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service.Labels).To(HaveKeyWithValue(labelKeyName, labelValueName))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyInstance, "test-bindplane"))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyManagedBy, labelValueManagedBy))
			Expect(service.Labels).To(HaveKeyWithValue(labelKeyPartOf, labelValuePartOf))
		})

		It("should have correct selector labels", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyName, labelValueName))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyInstance, "test-bindplane"))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Spec.Selector).NotTo(HaveKey(labelKeyManagedBy))
			Expect(service.Spec.Selector).NotTo(HaveKey(labelKeyPartOf))
		})
	})

	Context("when creating a service with multiple ports", func() {
		It("should create a service with all specified ports", func() {
			service := newService(bindplane, "test-component",
				WithPort("http", 8080),
				WithPort("metrics", 9090),
				WithPort("grpc", 50051),
			)

			Expect(service).NotTo(BeNil())
			Expect(service.Spec.Ports).To(HaveLen(3))

			// Verify first port
			Expect(service.Spec.Ports[0].Name).To(Equal("http"))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(8080)))

			// Verify second port
			Expect(service.Spec.Ports[1].Name).To(Equal("metrics"))
			Expect(service.Spec.Ports[1].Port).To(Equal(int32(9090)))
			Expect(service.Spec.Ports[1].TargetPort).To(Equal(intstr.FromInt(9090)))

			// Verify third port
			Expect(service.Spec.Ports[2].Name).To(Equal("grpc"))
			Expect(service.Spec.Ports[2].Port).To(Equal(int32(50051)))
			Expect(service.Spec.Ports[2].TargetPort).To(Equal(intstr.FromInt(50051)))

			// All ports should use TCP protocol
			for _, port := range service.Spec.Ports {
				Expect(port.Protocol).To(Equal(corev1.ProtocolTCP))
			}
		})

		It("should maintain port order", func() {
			service := newService(bindplane, "test-component",
				WithPort("first", 1000),
				WithPort("second", 2000),
				WithPort("third", 3000),
			)

			Expect(service.Spec.Ports).To(HaveLen(3))
			Expect(service.Spec.Ports[0].Name).To(Equal("first"))
			Expect(service.Spec.Ports[1].Name).To(Equal("second"))
			Expect(service.Spec.Ports[2].Name).To(Equal("third"))
		})
	})

	Context("when creating a service with no ports", func() {
		It("should create a service with empty ports slice", func() {
			service := newService(bindplane, "test-component")

			Expect(service).NotTo(BeNil())
			Expect(service.Spec.Ports).To(BeEmpty())
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		})

		It("should still have correct labels and selectors", func() {
			service := newService(bindplane, "test-component")

			Expect(service.Labels).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
			Expect(service.Spec.Selector).To(HaveKeyWithValue(labelKeyComponent, "test-component"))
		})
	})

	Context("when using different component names", func() {
		It("should generate correct service names", func() {
			service1 := newService(bindplane, "component-a", WithPort("http", 8080))
			service2 := newService(bindplane, "component-b", WithPort("http", 8080))

			Expect(service1.Name).To(Equal("test-bindplane-component-a"))
			Expect(service2.Name).To(Equal("test-bindplane-component-b"))
		})

		It("should use component name in labels and selectors", func() {
			service := newService(bindplane, "my-component", WithPort("http", 8080))

			Expect(service.Labels[labelKeyComponent]).To(Equal("my-component"))
			Expect(service.Spec.Selector[labelKeyComponent]).To(Equal("my-component"))
		})
	})

	Context("when using WithPort option", func() {
		It("should set both Port and TargetPort to the same value", func() {
			service := newService(bindplane, "test-component", WithPort("http", 8080))

			port := service.Spec.Ports[0]
			Expect(port.Port).To(Equal(int32(8080)))
			Expect(port.TargetPort).To(Equal(intstr.FromInt(8080)))
		})

		It("should handle different port values", func() {
			service := newService(bindplane, "test-component",
				WithPort("http", 80),
				WithPort("https", 443),
				WithPort("custom", 12345),
			)

			Expect(service.Spec.Ports[0].Port).To(Equal(int32(80)))
			Expect(service.Spec.Ports[1].Port).To(Equal(int32(443)))
			Expect(service.Spec.Ports[2].Port).To(Equal(int32(12345)))
		})
	})
})

var _ = Describe("newContainerSecurityContext", func() {
	Context("when creating a security context with default options", func() {
		It("should create a security context with all default security settings", func() {
			sc := newContainerSecurityContext()

			Expect(sc).NotTo(BeNil())
			Expect(sc.AllowPrivilegeEscalation).NotTo(BeNil())
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())
			Expect(sc.Capabilities).NotTo(BeNil())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(sc.ReadOnlyRootFilesystem).NotTo(BeNil())
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())
			Expect(sc.RunAsNonRoot).NotTo(BeNil())
			Expect(*sc.RunAsNonRoot).To(BeTrue())
			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534))) // Default nobody user
		})

		It("should have correct default RunAsUser", func() {
			sc := newContainerSecurityContext()

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534)))
		})
	})

	Context("when using WithRunAsUser option", func() {
		It("should override the default RunAsUser", func() {
			sc := newContainerSecurityContext(WithRunAsUser(1000))

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(1000)))
		})

		It("should maintain all other security settings when using WithRunAsUser", func() {
			sc := newContainerSecurityContext(WithRunAsUser(2000))

			// Verify RunAsUser is overridden
			Expect(*sc.RunAsUser).To(Equal(int64(2000)))

			// Verify all other settings remain the same
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())
			Expect(*sc.RunAsNonRoot).To(BeTrue())
		})

		It("should handle different RunAsUser values", func() {
			testCases := []struct {
				userID int64
			}{
				{0},     // root
				{1000},  // regular user
				{65534}, // nobody (default)
				{9999},  // custom user
			}

			for _, tc := range testCases {
				sc := newContainerSecurityContext(WithRunAsUser(tc.userID))
				Expect(sc.RunAsUser).NotTo(BeNil())
				Expect(*sc.RunAsUser).To(Equal(tc.userID))
			}
		})
	})

	Context("when verifying security best practices", func() {
		It("should always disable privilege escalation", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.AllowPrivilegeEscalation).To(BeFalse())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.AllowPrivilegeEscalation).To(BeFalse())
		})

		It("should always drop all capabilities", func() {
			sc := newContainerSecurityContext()
			Expect(sc.Capabilities).NotTo(BeNil())
			Expect(sc.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
			Expect(sc.Capabilities.Add).To(BeEmpty())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(sc2.Capabilities).NotTo(BeNil())
			Expect(sc2.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")))
		})

		It("should always use read-only root filesystem", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.ReadOnlyRootFilesystem).To(BeTrue())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.ReadOnlyRootFilesystem).To(BeTrue())
		})

		It("should always run as non-root", func() {
			sc := newContainerSecurityContext()
			Expect(*sc.RunAsNonRoot).To(BeTrue())

			sc2 := newContainerSecurityContext(WithRunAsUser(1000))
			Expect(*sc2.RunAsNonRoot).To(BeTrue())
		})
	})

	Context("when comparing default vs custom RunAsUser", func() {
		It("should produce different RunAsUser values", func() {
			defaultSC := newContainerSecurityContext()
			customSC := newContainerSecurityContext(WithRunAsUser(1000))

			Expect(*defaultSC.RunAsUser).To(Equal(int64(65534)))
			Expect(*customSC.RunAsUser).To(Equal(int64(1000)))
			Expect(*defaultSC.RunAsUser).NotTo(Equal(*customSC.RunAsUser))
		})

		It("should produce identical security settings except RunAsUser", func() {
			defaultSC := newContainerSecurityContext()
			customSC := newContainerSecurityContext(WithRunAsUser(1000))

			// RunAsUser should be different
			Expect(*defaultSC.RunAsUser).NotTo(Equal(*customSC.RunAsUser))

			// All other settings should be identical
			Expect(*defaultSC.AllowPrivilegeEscalation).To(Equal(*customSC.AllowPrivilegeEscalation))
			Expect(defaultSC.Capabilities.Drop).To(Equal(customSC.Capabilities.Drop))
			Expect(*defaultSC.ReadOnlyRootFilesystem).To(Equal(*customSC.ReadOnlyRootFilesystem))
			Expect(*defaultSC.RunAsNonRoot).To(Equal(*customSC.RunAsNonRoot))
		})
	})

	Context("when using WithRunAsUser with the default value", func() {
		It("should work correctly when explicitly setting the default value", func() {
			sc := newContainerSecurityContext(WithRunAsUser(65534))

			Expect(sc.RunAsUser).NotTo(BeNil())
			Expect(*sc.RunAsUser).To(Equal(int64(65534)))
		})
	})
})

var _ = Describe("newPodSecurityContext", func() {
	It("should create a pod security context with runtime default seccomp", func() {
		sc := newPodSecurityContext()

		Expect(sc).NotTo(BeNil())
		Expect(sc.FSGroup).To(Equal(new(int64(65534))))
		Expect(sc.RunAsGroup).To(Equal(new(int64(65534))))
		Expect(sc.RunAsUser).To(Equal(new(int64(65534))))
		Expect(sc.SeccompProfile).NotTo(BeNil())
		Expect(sc.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault))
	})
})

var _ = Describe("mergePodTemplateSpec", func() {
	var operatorManaged corev1.PodTemplateSpec

	BeforeEach(func() {
		operatorManaged = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app.kubernetes.io/name":      "bindplane",
					"app.kubernetes.io/instance":  "test-instance",
					"app.kubernetes.io/component": "test",
				},
				Annotations: map[string]string{
					"operator-managed": "true",
				},
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: "operator-managed-sa",
				Containers: []corev1.Container{
					{
						Name:  "operator-container",
						Image: "operator-image:latest",
					},
				},
				TerminationGracePeriodSeconds: new(int64(60)),
				SecurityContext:               newPodSecurityContext(),
				Volumes: []corev1.Volume{
					{
						Name: "operator-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}
	})

	Context("when user-provided template is nil", func() {
		It("should return operator-managed template unchanged", func() {
			result := mergePodTemplateSpec(operatorManaged, nil)
			Expect(result).To(Equal(operatorManaged))
		})
	})

	Context("when merging metadata", func() {
		It("should merge labels from user-provided template", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "bindplane"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "test-instance"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "test"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
		})

		It("should protect operator-managed selector labels from user overrides", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/name":      "user-override-name",
							"app.kubernetes.io/instance":  "user-override-instance",
							"app.kubernetes.io/component": "user-override-component",
							"user-label":                  "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			// Protected labels should retain operator-managed values
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "bindplane"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", "test-instance"))
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "test"))
			// User labels should still be merged
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
		})

		It("should merge annotations from user-provided template", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"user-annotation": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("operator-managed", "true"))
			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("user-annotation", "user-value"))
		})

		It("should handle nil labels and annotations in operator-managed template", func() {
			operatorManaged.Labels = nil
			operatorManaged.Annotations = nil

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "user-value",
						},
						Annotations: map[string]string{
							"user-annotation": "user-value",
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "user-value"))
			Expect(result.ObjectMeta.Annotations).To(HaveKeyWithValue("user-annotation", "user-value"))
		})
	})

	Context("when merging affinity", func() {
		It("should use user-provided affinity", func() {
			userAffinity := &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": "test",
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Affinity: userAffinity,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Affinity).To(Equal(userAffinity))
		})

		It("should preserve nil affinity when user doesn't provide it", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Affinity).To(BeNil())
		})
	})

	Context("when merging tolerations", func() {
		It("should use user-provided tolerations", func() {
			userTolerations := []corev1.Toleration{
				{
					Key:      "key1",
					Operator: corev1.TolerationOpEqual,
					Value:    "value1",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Tolerations: userTolerations,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Tolerations).To(Equal(userTolerations))
		})
	})

	Context("when merging nodeSelector", func() {
		It("should use user-provided nodeSelector", func() {
			userNodeSelector := map[string]string{
				"disktype": "ssd",
				"zone":     "us-west-1",
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: userNodeSelector,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.NodeSelector).To(Equal(userNodeSelector))
		})
	})

	Context("when merging priorityClassName", func() {
		It("should use user-provided priorityClassName", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						PriorityClassName: "high-priority",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.PriorityClassName).To(Equal("high-priority"))
		})

		It("should not override when priorityClassName is empty", func() {
			operatorManaged.Spec.PriorityClassName = "operator-priority"

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						PriorityClassName: "",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.PriorityClassName).To(Equal("operator-priority"))
		})
	})

	Context("when merging runtimeClassName", func() {
		It("should use user-provided runtimeClassName", func() {
			runtimeClass := "gvisor"
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RuntimeClassName: &runtimeClass,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.RuntimeClassName).To(Equal(&runtimeClass))
		})
	})

	Context("when merging hostNetwork, hostPID, hostIPC", func() {
		It("should use user-provided hostNetwork", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: true,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.HostNetwork).To(BeTrue())
		})

		It("should not override when hostNetwork is false", func() {
			operatorManaged.Spec.HostNetwork = true

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						HostNetwork: false,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.HostNetwork).To(BeTrue())
		})
	})

	Context("when merging volumes", func() {
		It("should merge user volumes with operator volumes", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "user-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "user-config",
										},
									},
								},
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Volumes).To(HaveLen(2))
			volumeNames := make(map[string]bool)
			for _, vol := range result.Spec.Volumes {
				volumeNames[vol.Name] = true
			}
			Expect(volumeNames).To(HaveKey("operator-volume"))
			Expect(volumeNames).To(HaveKey("user-volume"))
		})

		It("should allow user to override operator volume with same name", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "operator-volume",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "user-override",
										},
									},
								},
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Volumes).To(HaveLen(1))
			Expect(result.Spec.Volumes[0].Name).To(Equal("operator-volume"))
			Expect(result.Spec.Volumes[0].ConfigMap).ToNot(BeNil())
			Expect(result.Spec.Volumes[0].ConfigMap.Name).To(Equal("user-override"))
		})
	})

	Context("when merging initContainers", func() {
		It("should use user-provided initContainers", func() {
			userInitContainers := []corev1.Container{
				{
					Name:    "user-init",
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo init"},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						InitContainers: userInitContainers,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.InitContainers).To(Equal(userInitContainers))
		})
	})

	Context("when merging securityContext", func() {
		It("should merge user securityContext fields with operator securityContext", func() {
			userFSGroup := new(int64(1000))
			userSeccomp := &corev1.SeccompProfile{
				Type:             corev1.SeccompProfileTypeLocalhost,
				LocalhostProfile: new("profiles/custom.json"),
			}
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:        userFSGroup,
							RunAsUser:      new(int64(1000)),
							SeccompProfile: userSeccomp,
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(userFSGroup))
			Expect(result.Spec.SecurityContext.RunAsUser).To(Equal(new(int64(1000))))
			// Operator-managed fields should be preserved if not overridden
			Expect(result.Spec.SecurityContext.RunAsGroup).To(Equal(new(int64(65534))))
			Expect(result.Spec.SecurityContext.SeccompProfile).To(Equal(userSeccomp))
		})

		It("should handle nil securityContext in operator-managed template", func() {
			operatorManaged.Spec.SecurityContext = nil

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup: new(int64(1000)),
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.SecurityContext).ToNot(BeNil())
			Expect(result.Spec.SecurityContext.FSGroup).To(Equal(new(int64(1000))))
		})
	})

	Context("when preserving operator-managed fields", func() {
		It("should preserve ServiceAccountName", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "user-sa",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.ServiceAccountName).To(Equal("operator-managed-sa"))
		})

		It("should preserve Containers", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "user-container",
								Image: "user-image:latest",
							},
						},
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.Containers).To(HaveLen(1))
			Expect(result.Spec.Containers[0].Name).To(Equal("operator-container"))
			Expect(result.Spec.Containers[0].Image).To(Equal("operator-image:latest"))
		})

		It("should preserve TerminationGracePeriodSeconds", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						TerminationGracePeriodSeconds: new(int64(30)),
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(new(int64(60))))
		})
	})

	Context("when merging DNS settings", func() {
		It("should use user-provided DNSPolicy", func() {
			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSPolicy: corev1.DNSClusterFirst,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.DNSPolicy).To(Equal(corev1.DNSClusterFirst))
		})

		It("should use user-provided DNSConfig", func() {
			userDNSConfig := &corev1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8"},
				Searches:    []string{"example.com"},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSConfig: userDNSConfig,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.DNSConfig).To(Equal(userDNSConfig))
		})
	})

	Context("when merging imagePullSecrets", func() {
		It("should use user-provided imagePullSecrets", func() {
			userImagePullSecrets := []corev1.LocalObjectReference{
				{Name: "user-secret"},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ImagePullSecrets: userImagePullSecrets,
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			Expect(result.Spec.ImagePullSecrets).To(Equal(userImagePullSecrets))
		})
	})

	Context("complex merge scenarios", func() {
		It("should correctly merge multiple fields at once", func() {
			userAffinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "disktype",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"ssd"},
									},
								},
							},
						},
					},
				},
			}

			userProvided := &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"user-label": "value",
						},
					},
					Spec: corev1.PodSpec{
						Affinity: userAffinity,
						Tolerations: []corev1.Toleration{
							{
								Key:      "key1",
								Operator: corev1.TolerationOpEqual,
								Value:    "value1",
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
						NodeSelector: map[string]string{
							"zone": "us-west-1",
						},
						PriorityClassName: "high-priority",
					},
				},
			}

			result := mergePodTemplateSpec(operatorManaged, userProvided)

			// Verify all user fields are merged
			Expect(result.ObjectMeta.Labels).To(HaveKeyWithValue("user-label", "value"))
			Expect(result.Spec.Affinity).To(Equal(userAffinity))
			Expect(result.Spec.Tolerations).To(HaveLen(1))
			Expect(result.Spec.NodeSelector).To(HaveKeyWithValue("zone", "us-west-1"))
			Expect(result.Spec.PriorityClassName).To(Equal("high-priority"))

			// Verify operator-managed fields are preserved
			Expect(result.Spec.ServiceAccountName).To(Equal("operator-managed-sa"))
			Expect(result.Spec.Containers).To(HaveLen(1))
			Expect(result.Spec.Containers[0].Name).To(Equal("operator-container"))
			Expect(result.Spec.TerminationGracePeriodSeconds).To(Equal(new(int64(60))))
		})
	})
})

// envVarByName returns the value of the env var with the given name from the slice, or empty string if not found.
func envVarByName(envVars []corev1.EnvVar, name string) string {
	for _, ev := range envVars {
		if ev.Name == name {
			if ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
				return "(secret)"
			}
			return ev.Value
		}
	}
	return ""
}

func expectedGoMemLimit(quantity string) string {
	q := resource.MustParse(quantity)
	return strconv.FormatInt(applyMemoryHeadroom(q.Value()), 10)
}

var _ = Describe("getGoRuntimeEnvVars", func() {
	It("prefers limits over requests and applies the conversion policy", func() {
		envVars := getGoRuntimeEnvVars(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		})

		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("2"))
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("1Gi")))
	})

	It("falls back to requests and rounds CPU up to at least one core", func() {
		envVars := getGoRuntimeEnvVars(corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		})

		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("1"))
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("500Mi")))
	})

	It("rounds a 500m CPU limit up to one core", func() {
		envVars := getGoRuntimeEnvVars(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		})

		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("1"))
	})

	It("omits env vars when no CPU or memory resources are set", func() {
		envVars := getGoRuntimeEnvVars(corev1.ResourceRequirements{})
		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(BeEmpty())
	})
})

var _ = Describe("mergePodTemplateSpec Go runtime env vars", func() {
	It("applies runtime env vars when no user pod template is provided", func() {
		operatorManaged := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "server",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		}

		result := mergePodTemplateSpec(operatorManaged, nil)
		envVars := result.Spec.Containers[0].Env
		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("1"))
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("1Gi")))
	})

	It("uses user-overridden resources and protects env vars from user override", func() {
		operatorManaged := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "server",
						Env:  []corev1.EnvVar{{Name: "BASE_ENV", Value: "operator"}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		}
		userProvided := &bindplanev1alpha1.PodTemplateSpec{
			PodTemplateSpec: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "server",
							Env:  []corev1.EnvVar{{Name: goMaxProcsEnvVar, Value: "99"}},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1500m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						},
					},
				},
			},
		}

		result := mergePodTemplateSpec(operatorManaged, userProvided)
		container := result.Spec.Containers[0]

		expectedLimitMemory := resource.MustParse("2Gi")
		Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1500)))
		Expect(container.Resources.Limits.Memory().Value()).To(Equal(expectedLimitMemory.Value()))
		Expect(envVarByName(container.Env, "BASE_ENV")).To(Equal("operator"))
		Expect(envVarByName(container.Env, goMaxProcsEnvVar)).To(Equal("2"))
		Expect(envVarByName(container.Env, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("2Gi")))
	})
})

var _ = Describe("workload Go runtime env vars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		replicas := int32(3)
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
				Bindplane: bindplanev1alpha1.BindplaneComponentSpec{
					Replicas: &replicas,
				},
			},
		}
	}

	It("adds runtime env vars to the node deployment from default resources", func() {
		bindplane := baseBindplane()
		deployment := (&BindplaneReconciler{}).nodeDeployment(bindplane)
		envVars := deployment.Spec.Template.Spec.Containers[0].Env

		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("2"))
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("2048Mi")))
	})

	It("adds runtime env vars to the TSDB statefulset from default resources", func() {
		bindplane := baseBindplane()
		statefulSet := (&BindplaneReconciler{}).tsdbStatefulSet(bindplane)
		envVars := statefulSet.Spec.Template.Spec.Containers[0].Env

		Expect(envVarByName(envVars, goMaxProcsEnvVar)).To(Equal("1"))
		Expect(envVarByName(envVars, goMemLimitEnvVar)).To(Equal(expectedGoMemLimit("2048Mi")))
	})
})

var _ = Describe("component-level resources field", func() {
	natsReplicas := int32(2)
	taReplicas := int32(2)

	newBindplane := func() *bindplanev1alpha1.Bindplane {
		replicas := int32(3)
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"},
					},
				},
				Bindplane: bindplanev1alpha1.BindplaneComponentSpec{Replicas: &replicas},
				Nats:      &bindplanev1alpha1.NatsComponentSpec{Replicas: &natsReplicas},
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					Replicas: &taReplicas,
				},
			},
		}
	}

	customResources := func() *corev1.ResourceRequirements {
		return &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("50Mi"),
			},
		}
	}

	assertResources := func(got corev1.ResourceRequirements, want *corev1.ResourceRequirements) {
		Expect(got.Requests.Cpu().MilliValue()).To(Equal(want.Requests.Cpu().MilliValue()))
		Expect(got.Requests.Memory().Value()).To(Equal(want.Requests.Memory().Value()))
		Expect(got.Limits.Cpu().MilliValue()).To(Equal(want.Limits.Cpu().MilliValue()))
		Expect(got.Limits.Memory().Value()).To(Equal(want.Limits.Memory().Value()))
	}

	Describe("spec.bindplane.resources", func() {
		It("applies top-level resources to the node container", func() {
			bp := newBindplane()
			bp.Spec.Bindplane.Resources = customResources()
			container := (&BindplaneReconciler{}).nodeDeployment(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when resources is not set", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).nodeDeployment(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("2048Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})

		It("podTemplate.resources takes precedence over top-level resources for specified fields", func() {
			podTemplateResources := &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("777m"),
					corev1.ResourceMemory: resource.MustParse("999Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("777m"),
					corev1.ResourceMemory: resource.MustParse("999Mi"),
				},
			}
			bp := newBindplane()
			bp.Spec.Bindplane.Resources = customResources()
			bp.Spec.Bindplane.PodTemplate = &bindplanev1alpha1.PodTemplateSpec{
				PodTemplateSpec: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: nodeContainerName, Resources: *podTemplateResources},
						},
					},
				},
			}
			container := (&BindplaneReconciler{}).nodeDeployment(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, podTemplateResources)
		})
	})

	Describe("spec.nats.resources", func() {
		It("applies top-level resources to the NATS container", func() {
			bp := newBindplane()
			bp.Spec.Nats.Resources = customResources()
			container := (&BindplaneReconciler{}).natsStatefulSet(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when resources is not set", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).natsStatefulSet(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("500Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(250)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})
	})

	Describe("spec.tsdb.resources", func() {
		It("applies top-level resources to the TSDB container", func() {
			bp := newBindplane()
			bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{Resources: customResources()}
			container := (&BindplaneReconciler{}).tsdbStatefulSet(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when TSDB spec is nil", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).tsdbStatefulSet(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("2048Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1000)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})

		It("podTemplate.resources takes precedence over top-level resources for specified fields", func() {
			podTemplateResources := &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("777m"),
					corev1.ResourceMemory: resource.MustParse("999Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("777m"),
					corev1.ResourceMemory: resource.MustParse("999Mi"),
				},
			}
			bp := newBindplane()
			bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{
				Resources: customResources(),
				PodTemplate: &bindplanev1alpha1.PodTemplateSpec{
					PodTemplateSpec: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: tsdbContainerName, Resources: *podTemplateResources},
							},
						},
					},
				},
			}
			container := (&BindplaneReconciler{}).tsdbStatefulSet(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, podTemplateResources)
		})
	})

	Describe("spec.transformAgent.resources", func() {
		It("applies top-level resources to the transform agent container", func() {
			bp := newBindplane()
			bp.Spec.TransformAgent.Resources = customResources()
			container := (&BindplaneReconciler{}).transformAgentDeployment(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when resources is not set", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).transformAgentDeployment(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("512Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(250)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})
	})

	Describe("spec.bindplaneJobs.resources", func() {
		It("applies top-level resources to the jobs container", func() {
			bp := newBindplane()
			bp.Spec.BindplaneJobs = &bindplanev1alpha1.BindplaneJobsComponentSpec{Resources: customResources()}
			container := (&BindplaneReconciler{}).bindplaneJobsDeployment(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when bindplaneJobs spec is nil", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).bindplaneJobsDeployment(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("1024Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1000)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})
	})

	Describe("spec.bindplaneJobsMigrate.resources", func() {
		It("applies top-level resources to the jobs migrate container", func() {
			bp := newBindplane()
			bp.Spec.BindplaneJobsMigrate = &bindplanev1alpha1.BindplaneJobsMigrateComponentSpec{Resources: customResources()}
			container := (&BindplaneReconciler{}).bindplaneJobsMigrateJob(bp).Spec.Template.Spec.Containers[0]
			assertResources(container.Resources, customResources())
		})

		It("uses defaults when bindplaneJobsMigrate spec is nil", func() {
			bp := newBindplane()
			container := (&BindplaneReconciler{}).bindplaneJobsMigrateJob(bp).Spec.Template.Spec.Containers[0]
			expectedMem := resource.MustParse("2048Mi")
			Expect(container.Resources.Requests.Cpu().MilliValue()).To(Equal(int64(100)))
			Expect(container.Resources.Requests.Memory().Value()).To(Equal(expectedMem.Value()))
		})
	})
})

var _ = Describe("workload pod security context defaults", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		nodeReplicas := int32(3)
		natsReplicas := int32(2)
		transformAgentReplicas := int32(2)
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
				Bindplane: bindplanev1alpha1.BindplaneComponentSpec{
					Replicas: &nodeReplicas,
				},
				Nats: &bindplanev1alpha1.NatsComponentSpec{
					Replicas: &natsReplicas,
				},
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					Replicas: &transformAgentReplicas,
				},
			},
		}
	}

	assertRuntimeDefaultSeccomp := func(securityContext *corev1.PodSecurityContext) {
		Expect(securityContext).NotTo(BeNil())
		Expect(securityContext.SeccompProfile).NotTo(BeNil())
		Expect(securityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault))
	}

	It("sets runtime default seccomp on node deployments", func() {
		bindplane := baseBindplane()
		deployment := (&BindplaneReconciler{}).nodeDeployment(bindplane)

		assertRuntimeDefaultSeccomp(deployment.Spec.Template.Spec.SecurityContext)
	})

	It("sets runtime default seccomp on TSDB statefulsets", func() {
		bindplane := baseBindplane()
		statefulSet := (&BindplaneReconciler{}).tsdbStatefulSet(bindplane)

		assertRuntimeDefaultSeccomp(statefulSet.Spec.Template.Spec.SecurityContext)
	})

	It("sets runtime default seccomp on migrate jobs", func() {
		bindplane := baseBindplane()
		job := (&BindplaneReconciler{}).bindplaneJobsMigrateJob(bindplane)

		assertRuntimeDefaultSeccomp(job.Spec.Template.Spec.SecurityContext)
	})
})

var _ = Describe("nodeTerminationGracePeriodSeconds", func() {
	makeBindplane := func(opampPeriod string) *bindplanev1alpha1.Bindplane {
		bp := &bindplanev1alpha1.Bindplane{}
		if opampPeriod != "" {
			bp.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
				Server: &bindplanev1alpha1.AdvancedServerConfig{
					OpAMPShutdownGracePeriod: opampPeriod,
				},
			}
		}
		return bp
	}

	It("returns default when OpAMPShutdownGracePeriod is not set", func() {
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane(""))).To(Equal(int64(60)))
	})

	It("returns 125% of 100s rounded up", func() {
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("100s"))).To(Equal(int64(125)))
	})

	It("returns 125% of 60s rounded up", func() {
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("60s"))).To(Equal(int64(75)))
	})

	It("returns 125% of 1m (60s) rounded up", func() {
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("1m"))).To(Equal(int64(75)))
	})

	It("rounds a fractional result up to the next whole second", func() {
		// 4s * 1.25 = 5s exactly — no rounding needed
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("4s"))).To(Equal(int64(5)))
		// 1s * 1.25 = 1.25s → ceil = 2s
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("1s"))).To(Equal(int64(2)))
	})

	It("falls back to default on an unparseable value", func() {
		Expect(nodeTerminationGracePeriodSeconds(makeBindplane("invalid"))).To(Equal(int64(60)))
	})

	It("falls back to default when Advanced is nil", func() {
		Expect(nodeTerminationGracePeriodSeconds(&bindplanev1alpha1.Bindplane{})).To(Equal(int64(60)))
	})
})

var _ = Describe("getBindplaneConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
			},
		}
	}

	It("sets default metrics env vars when Metrics config is omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_TYPE")).To(Equal("prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_INTERVAL")).To(Equal("60s"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_ENDPOINT")).To(Equal("/metrics"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
	})

	It("sets explicit metrics type prometheus, interval 60s, endpoint /metrics", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Metrics = &bindplanev1alpha1.MetricsConfig{
			Type:     "prometheus",
			Interval: "60s",
			Prometheus: &bindplanev1alpha1.MetricsPrometheusConfig{
				Endpoint: "/metrics",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_TYPE")).To(Equal("prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_INTERVAL")).To(Equal("60s"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_ENDPOINT")).To(Equal("/metrics"))
	})

	It("sets tracing type otlp with endpoint, insecure, and sampling rate", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Tracing = &bindplanev1alpha1.TracingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.TracingOTLPConfig{
				Endpoint: "http://otel:4317",
				Insecure: true,
			},
			SamplingRate: "0.5",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(Equal("otlp"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_ENDPOINT")).To(Equal("http://otel:4317"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_INSECURE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_SAMPLING_RATE")).To(Equal("0.5"))
	})

	It("sets metrics Prometheus auth username and password secret ref", func() {
		bindplane := baseBindplane()
		secretName := "metrics-auth"
		bindplane.Spec.Config.Metrics = &bindplanev1alpha1.MetricsConfig{
			Type:     "prometheus",
			Interval: "60s",
			Prometheus: &bindplanev1alpha1.MetricsPrometheusConfig{
				Endpoint: "/metrics",
				Username: "metrics-user",
				PasswordSecretRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
					Key:                  "password",
				},
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)

		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_USERNAME")).To(Equal("metrics-user"))
		Expect(envVarByName(envVars, "BINDPLANE_METRICS_PROMETHEUS_PASSWORD")).To(Equal("(secret)"))
	})

	It("does not set tracing env vars when Tracing is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_OTLP_ENDPOINT")).To(BeEmpty())
	})

	It("does not set tracing env vars when Tracing Type is empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Tracing = &bindplanev1alpha1.TracingConfig{}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TRACING_TYPE")).To(BeEmpty())
	})

	It("sets maxConcurrency and auditTrail retention with defaults when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_MAX_CONCURRENCY")).To(Equal("10"))
		Expect(envVarByName(envVars, "BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS")).To(Equal("10"))
		Expect(envVarByName(envVars, "BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS")).To(Equal("365"))
	})

	It("sets explicit maxConcurrency and auditTrail.retentionDays", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.MaxConcurrency = 20
		bindplane.Spec.Config.AuditTrail = &bindplanev1alpha1.AuditTrailConfig{RetentionDays: 180}
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{MaxSimultaneousConnections: 20}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_MAX_CONCURRENCY")).To(Equal("20"))
		Expect(envVarByName(envVars, "BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS")).To(Equal("20"))
		Expect(envVarByName(envVars, "BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS")).To(Equal("180"))
	})

	It("defaults BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS to 10 when agents is nil", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = nil
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS")).To(Equal("10"))
	})

	It("sets network webURL and corsAllowedOrigins only when configured", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_WEB_URL")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_CORS_ALLOWED_ORIGINS")).To(BeEmpty())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			WebURL:             "https://bindplane.example.com",
			CorsAllowedOrigins: "https://app.example.com",
		}
		envVars = getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_WEB_URL")).To(Equal("https://bindplane.example.com"))
		Expect(envVarByName(envVars, "BINDPLANE_CORS_ALLOWED_ORIGINS")).To(Equal("https://app.example.com"))
	})

	It("does not set network TLS env vars when Network or TLS is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_MIN_VERSION")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_SKIP_VERIFY")).To(BeEmpty())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{}
		envVars = getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
	})

	It("sets network TLS minVersion and skipVerify only when no secret (no path env vars)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				MinVersion: "1.2",
				SkipVerify: true,
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_MIN_VERSION")).To(Equal("1.2"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_SKIP_VERIFY")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(BeEmpty())
	})

	It("sets network TLS cert and key paths when secretName, certKey, keyKey are set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				SecretName: "tls-secret",
				CertKey:    "tls.crt",
				KeyKey:     "tls.key",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(Equal("/etc/bindplane/network-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(Equal("/etc/bindplane/network-tls/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(BeEmpty())
	})

	It("sets network TLS cert, key, and ca paths when caKey is set (mutual TLS)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{
			TLS: &bindplanev1alpha1.NetworkTLSConfig{
				SecretName: "tls-secret",
				CertKey:    "tls.crt",
				KeyKey:     "tls.key",
				CAKey:      "ca.crt",
			},
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CERT")).To(Equal("/etc/bindplane/network-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_KEY")).To(Equal("/etc/bindplane/network-tls/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_TLS_CA")).To(Equal("/etc/bindplane/network-tls/ca.crt"))
	})
})

var _ = Describe("getNetworkTLSVolumeAndMount", func() {
	It("returns nil when Network or TLS is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())

		bindplane.Spec.Config.Network = &bindplanev1alpha1.NetworkConfig{}
		vols, mounts = getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns nil when secretName or certKey or keyKey is missing", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					Network: &bindplanev1alpha1.NetworkConfig{
						TLS: &bindplanev1alpha1.NetworkTLSConfig{SecretName: "tls-secret"},
					},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when server TLS is configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					Network: &bindplanev1alpha1.NetworkConfig{
						TLS: &bindplanev1alpha1.NetworkTLSConfig{
							SecretName: "tls-secret",
							CertKey:    "tls.crt",
							KeyKey:     "tls.key",
						},
					},
				},
			},
		}
		vols, mounts := getNetworkTLSVolumeAndMount(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal("network-tls"))
		Expect(vols[0].Secret).ToNot(BeNil())
		Expect(vols[0].Secret.SecretName).To(Equal("tls-secret"))
		Expect(mounts).To(HaveLen(1))
		Expect(mounts[0].Name).To(Equal("network-tls"))
		Expect(mounts[0].MountPath).To(Equal("/etc/bindplane/network-tls"))
	})
})

var _ = Describe("getBindplaneConfigEnvVars Postgres TLS", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"},
					},
				},
			},
		}
	}

	It("defaults sslMode to disable when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_MODE")).To(Equal(postgresSSLModeDisable))
	})

	It("does not set postgres TLS path env vars when TLS is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(BeEmpty())
	})

	It("sets postgres TLS root cert only when TLS has secretName and caKey (server-side TLS)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.Postgres.TLS = &bindplanev1alpha1.PostgresTLSConfig{
			SecretName: "pg-tls",
			CAKey:      "ca.crt",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(Equal("/etc/bindplane/postgres-tls/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(BeEmpty())
	})

	It("sets postgres TLS root cert, cert, and key when mutual TLS (caKey, certKey, keyKey)", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.Postgres.TLS = &bindplanev1alpha1.PostgresTLSConfig{
			SecretName: "pg-tls",
			CAKey:      "ca.crt",
			CertKey:    "tls.crt",
			KeyKey:     "tls.key",
		}
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_ROOT_CERT")).To(Equal("/etc/bindplane/postgres-tls/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_CERT")).To(Equal("/etc/bindplane/postgres-tls/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_SSL_KEY")).To(Equal("/etc/bindplane/postgres-tls/tls.key"))
	})

	It("does not set maxIdleConnections or maxIdleTime when omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_TIME")).To(BeEmpty())
	})

	It("sets maxIdleConnections and maxIdleTime when provided", func() {
		bindplane := baseBindplane()
		maxIdle := 5
		bindplane.Spec.Config.Store.Postgres.MaxIdleConnections = &maxIdle
		bindplane.Spec.Config.Store.Postgres.MaxIdleTime = "20s"
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS")).To(Equal("5"))
		Expect(envVarByName(envVars, "BINDPLANE_POSTGRES_MAX_IDLE_TIME")).To(Equal("20s"))
	})
})

var _ = Describe("getStoreConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"},
					},
				},
			},
		}
	}

	It("does not set store tuning env vars when all fields are omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_MAX_EVENTS")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_STORE_EVENT_MERGE_WINDOW")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS")).To(BeEmpty())
	})

	It("sets BINDPLANE_STORE_MAX_EVENTS when maxEvents > 0", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.MaxEvents = 200
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_MAX_EVENTS")).To(Equal("200"))
	})

	It("does not set BINDPLANE_STORE_MAX_EVENTS when maxEvents == 0", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.MaxEvents = 0
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_MAX_EVENTS")).To(BeEmpty())
	})

	It("sets BINDPLANE_STORE_EVENT_MERGE_WINDOW when non-empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Store.EventMergeWindow = "200ms"
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_EVENT_MERGE_WINDOW")).To(Equal("200ms"))
	})

	It("does not set BINDPLANE_STORE_EVENT_MERGE_WINDOW when empty", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_EVENT_MERGE_WINDOW")).To(BeEmpty())
	})

	It("sets BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS when non-nil with value > 0", func() {
		bindplane := baseBindplane()
		days := 90
		bindplane.Spec.Config.Store.SummaryRollupRetentionDays = &days
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS")).To(Equal("90"))
	})

	It("sets BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS=0 when non-nil with value == 0 (indefinite retention)", func() {
		bindplane := baseBindplane()
		days := 0
		bindplane.Spec.Config.Store.SummaryRollupRetentionDays = &days
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS")).To(Equal("0"))
	})

	It("does not set BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS when nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneConfigEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS")).To(BeEmpty())
	})
})

var _ = Describe("getBindplaneCommonEnvVars profiling and pprof", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bp",
				Namespace: "default",
			},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
						},
					},
				},
			},
		}
	}

	It("does not set profiling or pprof env vars when disabled or omitted", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneProfilingEnabledEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplanePprofEnabledEnvVar)).To(BeEmpty())
	})

	It("sets profiling env vars with default serviceName per component", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Profiling = &bindplanev1alpha1.ProfilingConfig{
			Enabled:   true,
			ProjectID: "my-gcp-project",
		}
		for _, tc := range []struct {
			component string
			wantName  string
		}{
			{nodeComponent, "bindplane-node"},
			{bindplaneJobsComponent, "bindplane-jobs"},
			{bindplaneJobsMigrateComponent, "bindplane-migrate"},
			{natsComponent, "bindplane-nats"},
		} {
			envVars := getBindplaneCommonEnvVars(bindplane, tc.component)
			Expect(envVarByName(envVars, bindplaneProfilingEnabledEnvVar)).To(Equal("true"))
			Expect(envVarByName(envVars, bindplaneProfilingProjectIDEnvVar)).To(Equal("my-gcp-project"))
			Expect(envVarByName(envVars, bindplaneProfilingServiceNameEnvVar)).To(Equal(tc.wantName))
		}
	})

	It("sets profiling noCPU, noAlloc, noHeap, noGoroutine, mutex when enabled", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Profiling = &bindplanev1alpha1.ProfilingConfig{
			Enabled:     true,
			ProjectID:   "proj",
			NoCPU:       true,
			NoAlloc:     true,
			NoHeap:      true,
			NoGoroutine: true,
			Mutex:       true,
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneProfilingNoCPUEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoAllocEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoHeapEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingNoGoroutineEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneProfilingMutexEnvVar)).To(Equal("true"))
	})

	It("sets pprof env vars with default endpoint when enabled and endpoint unset", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Pprof = &bindplanev1alpha1.PprofConfig{Enabled: true}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplanePprofEnabledEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplanePprofEndpointEnvVar)).To(Equal(defaultPprofEndpoint))
	})

	It("sets pprof env vars with explicit endpoint when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Pprof = &bindplanev1alpha1.PprofConfig{
			Enabled:  true,
			Endpoint: "0.0.0.0:6061",
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplanePprofEndpointEnvVar)).To(Equal("0.0.0.0:6061"))
	})

	It("does not set status env vars when Status is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneStatusEnabledEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneStatusKeysEnvVar)).To(BeEmpty())
	})

	It("sets BINDPLANE_STATUS_ENABLED=true and BINDPLANE_STATUS_KEYS from inline keys", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Status = &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			Keys:    []string{"550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneStatusEnabledEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneStatusKeysEnvVar)).To(Equal("550e8400-e29b-41d4-a716-446655440000,6ba7b810-9dad-11d1-80b4-00c04fd430c8"))
	})

	It("sets BINDPLANE_STATUS_ENABLED=false and no keys env var when disabled", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Status = &bindplanev1alpha1.StatusConfig{Enabled: false}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneStatusEnabledEnvVar)).To(Equal("false"))
		Expect(envVarByName(envVars, bindplaneStatusKeysEnvVar)).To(BeEmpty())
	})

	It("sets BINDPLANE_STATUS_KEYS from keysSecretRef using ValueFrom", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Status = &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			KeysSecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
				Key:                  "keys",
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneStatusEnabledEnvVar)).To(Equal("true"))
		var keysVar *corev1.EnvVar
		for i := range envVars {
			if envVars[i].Name == bindplaneStatusKeysEnvVar {
				keysVar = &envVars[i]
				break
			}
		}
		Expect(keysVar).NotTo(BeNil())
		Expect(keysVar.ValueFrom).NotTo(BeNil())
		Expect(keysVar.ValueFrom.SecretKeyRef.Name).To(Equal("my-secret"))
		Expect(keysVar.ValueFrom.SecretKeyRef.Key).To(Equal("keys"))
	})
})

var _ = Describe("defaultRequiredHosts", func() {
	It("returns floor(total/2)+1 with default replicas (node=3, nats=2)", func() {
		natsReplicas := int32(2)
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
				Nats: &bindplanev1alpha1.NatsComponentSpec{Replicas: &natsReplicas},
			},
		}
		// total = 3 + 2 + 1 = 6, floor(6/2)+1 = 4
		Expect(defaultRequiredHosts(bindplane)).To(Equal(int32(4)))
	})

	It("returns floor(total/2)+1 with custom replicas (node=5, nats=3)", func() {
		nodeReplicas := int32(5)
		natsReplicas := int32(3)
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Bindplane: bindplanev1alpha1.BindplaneComponentSpec{Replicas: &nodeReplicas},
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
				Nats: &bindplanev1alpha1.NatsComponentSpec{Replicas: &natsReplicas},
			},
		}
		// total = 5 + 3 + 1 = 9, floor(9/2)+1 = 5
		Expect(defaultRequiredHosts(bindplane)).To(Equal(int32(5)))
	})
})

var _ = Describe("migrate Job helpers", func() {
	makeJob := func(image string, conditions ...batchv1.JobCondition) *batchv1.Job {
		return &batchv1.Job{
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: image}},
					},
				},
			},
			Status: batchv1.JobStatus{Conditions: conditions},
		}
	}

	Describe("isJobSucceeded", func() {
		It("returns true when JobComplete condition is True", func() {
			job := makeJob("img", batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			})
			Expect(isJobSucceeded(job)).To(BeTrue())
		})

		It("returns false when no conditions", func() {
			Expect(isJobSucceeded(makeJob("img"))).To(BeFalse())
		})

		It("returns false when JobComplete condition is False", func() {
			job := makeJob("img", batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionFalse,
			})
			Expect(isJobSucceeded(job)).To(BeFalse())
		})
	})

	Describe("isJobFailed", func() {
		It("returns true when JobFailed condition is True", func() {
			job := makeJob("img", batchv1.JobCondition{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionTrue,
			})
			Expect(isJobFailed(job)).To(BeTrue())
		})

		It("returns false when no conditions", func() {
			Expect(isJobFailed(makeJob("img"))).To(BeFalse())
		})
	})

	Describe("extractJobContainerImage", func() {
		It("returns the first container image", func() {
			Expect(extractJobContainerImage(makeJob("my-image:1.2.3"))).To(Equal("my-image:1.2.3"))
		})

		It("returns empty string when no containers", func() {
			job := &batchv1.Job{}
			Expect(extractJobContainerImage(job)).To(Equal(""))
		})
	})

	Describe("bindplaneJobsMigrateJob", func() {
		var bindplane *bindplanev1alpha1.Bindplane
		var r *BindplaneReconciler

		BeforeEach(func() {
			bindplane = &bindplanev1alpha1.Bindplane{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: bindplanev1alpha1.BindplaneSpec{
					Config: bindplanev1alpha1.BindplaneConfigSpec{
						License: "license",
						Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					},
				},
			}
			r = &BindplaneReconciler{}
		})

		It("produces a Job with correct name and namespace", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			Expect(job.Name).To(Equal("test-migrate"))
			Expect(job.Namespace).To(Equal("default"))
		})

		It("sets the migrate command", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			containers := job.Spec.Template.Spec.Containers
			Expect(containers).To(HaveLen(1))
			Expect(containers[0].Command).To(Equal([]string{"/bindplane", "migrate", "-y"}))
		})

		It("sets RestartPolicy to OnFailure", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			Expect(job.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyOnFailure))
		})

		It("sets BackoffLimit to 3", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			Expect(job.Spec.BackoffLimit).NotTo(BeNil())
			Expect(*job.Spec.BackoffLimit).To(Equal(int32(3)))
		})

		It("sets TTLSecondsAfterFinished to 86400 (24 hours)", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			Expect(job.Spec.TTLSecondsAfterFinished).NotTo(BeNil())
			Expect(*job.Spec.TTLSecondsAfterFinished).To(Equal(int32(86400)))
		})

		It("has no ports or probes", func() {
			job := r.bindplaneJobsMigrateJob(bindplane)
			c := job.Spec.Template.Spec.Containers[0]
			Expect(c.Ports).To(BeEmpty())
			Expect(c.LivenessProbe).To(BeNil())
			Expect(c.ReadinessProbe).To(BeNil())
			Expect(c.StartupProbe).To(BeNil())
		})

		It("does not mount the NATS TLS volume even when NATS cert-manager TLS is configured", func() {
			bindplane.Spec.Config.Nats = &bindplanev1alpha1.NatsConfig{
				TLS: &bindplanev1alpha1.NatsTLSConfig{
					CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer"},
				},
			}
			job := r.bindplaneJobsMigrateJob(bindplane)
			for _, v := range job.Spec.Template.Spec.Volumes {
				Expect(v.Name).NotTo(Equal(internalTLSNatsVolumeName))
			}
			for _, m := range job.Spec.Template.Spec.Containers[0].VolumeMounts {
				Expect(m.Name).NotTo(Equal(internalTLSNatsVolumeName))
			}
		})
	})
})

var _ = Describe("getEventBusHealthEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		nodeReplicas := int32(3)
		natsReplicas := int32(2)
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Bindplane: bindplanev1alpha1.BindplaneComponentSpec{Replicas: &nodeReplicas},
				Nats:      &bindplanev1alpha1.NatsComponentSpec{Replicas: &natsReplicas},
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	It("does not set event bus health env vars when EventBus is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneEventBusHealthRequiredHostsEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneEventBusHealthIntervalEnvVar)).To(BeEmpty())
	})

	It("uses default requiredHosts (node=3, nats=2) = 4 when not overridden", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.EventBus = &bindplanev1alpha1.EventBusConfig{
			Health: &bindplanev1alpha1.EventBusHealthConfig{},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneEventBusHealthRequiredHostsEnvVar)).To(Equal("4"))
	})

	It("uses override requiredHosts when set", func() {
		override := int32(3)
		bindplane := baseBindplane()
		bindplane.Spec.Config.EventBus = &bindplanev1alpha1.EventBusConfig{
			Health: &bindplanev1alpha1.EventBusHealthConfig{RequiredHosts: &override},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneEventBusHealthRequiredHostsEnvVar)).To(Equal("3"))
	})

	It("sets interval env var when interval is provided", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.EventBus = &bindplanev1alpha1.EventBusConfig{
			Health: &bindplanev1alpha1.EventBusHealthConfig{Interval: "15s"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneEventBusHealthIntervalEnvVar)).To(Equal("15s"))
	})

	It("omits interval env var when interval is not set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.EventBus = &bindplanev1alpha1.EventBusConfig{
			Health: &bindplanev1alpha1.EventBusHealthConfig{},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneEventBusHealthIntervalEnvVar)).To(BeEmpty())
	})
})

var _ = Describe("getAnalyticsEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	It("does not set analytics env vars when Analytics is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAnalyticsDisabledEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAnalyticsSegmentWriteKeyEnvVar)).To(BeEmpty())
	})

	It("does not set BINDPLANE_ANALYTICS_DISABLED when disabled is false", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Analytics = &bindplanev1alpha1.AnalyticsConfig{Disabled: false}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAnalyticsDisabledEnvVar)).To(BeEmpty())
	})

	It("sets BINDPLANE_ANALYTICS_DISABLED=true when disabled is true", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Analytics = &bindplanev1alpha1.AnalyticsConfig{Disabled: true}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAnalyticsDisabledEnvVar)).To(Equal("true"))
	})

	It("does not set BINDPLANE_ANALYTICS_SEGMENT_WRITE_KEY when not provided", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Analytics = &bindplanev1alpha1.AnalyticsConfig{}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAnalyticsSegmentWriteKeyEnvVar)).To(BeEmpty())
	})

	It("sets BINDPLANE_ANALYTICS_SEGMENT_WRITE_KEY when provided", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Analytics = &bindplanev1alpha1.AnalyticsConfig{SegmentWriteKey: "my-write-key"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAnalyticsSegmentWriteKeyEnvVar)).To(Equal("my-write-key"))
	})
})

var _ = Describe("getLoggingConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	It("does not set logging env vars when Logging is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingLevelEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingTypeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingOTLPEndpointEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingOTLPInsecureEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingOTLPIntervalEnvVar)).To(BeEmpty())
	})

	It("sets default level=info and type=stdout when Logging is empty struct", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingLevelEnvVar)).To(Equal("info"))
		Expect(envVarByName(envVars, bindplaneLoggingTypeEnvVar)).To(Equal("stdout"))
		Expect(envVarByName(envVars, bindplaneLoggingOTLPEndpointEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingOTLPInsecureEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneLoggingOTLPIntervalEnvVar)).To(BeEmpty())
	})

	It("sets BINDPLANE_LOGGING_LEVEL=debug when level is debug", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{Level: "debug"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingLevelEnvVar)).To(Equal("debug"))
	})

	It("sets BINDPLANE_LOGGING_TYPE=otlp when type is otlp", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{Type: "otlp"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingTypeEnvVar)).To(Equal("otlp"))
	})

	It("sets BINDPLANE_LOGGING_TYPE=stdout,otlp when type is stdout,otlp", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{Type: "stdout,otlp"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingTypeEnvVar)).To(Equal("stdout,otlp"))
	})

	It("sets all OTLP vars when endpoint, insecure, and interval are configured", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.LoggingOTLPConfig{
				Endpoint: "localhost:4317",
				Insecure: true,
				Interval: "30s",
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingOTLPEndpointEnvVar)).To(Equal("localhost:4317"))
		Expect(envVarByName(envVars, bindplaneLoggingOTLPInsecureEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneLoggingOTLPIntervalEnvVar)).To(Equal("30s"))
	})

	It("does not set BINDPLANE_LOGGING_OTLP_INTERVAL when interval is not set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.LoggingOTLPConfig{Endpoint: "localhost:4317"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingOTLPIntervalEnvVar)).To(BeEmpty())
	})

	It("does not set BINDPLANE_LOGGING_OTLP_INSECURE when insecure is false", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Logging = &bindplanev1alpha1.LoggingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.LoggingOTLPConfig{Endpoint: "localhost:4317", Insecure: false},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneLoggingOTLPInsecureEnvVar)).To(BeEmpty())
	})
})

var _ = Describe("getPostgresTLSVolumeAndMount", func() {
	It("returns nil when Postgres or TLS is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns nil when secretName or caKey is missing", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
							TLS:  &bindplanev1alpha1.PostgresTLSConfig{SecretName: "pg-tls"},
						},
					},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when server-side TLS (caKey) is configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Store: bindplanev1alpha1.StoreConfig{
						Postgres: &bindplanev1alpha1.PostgresConfig{
							Host: "pg",
							TLS: &bindplanev1alpha1.PostgresTLSConfig{
								SecretName: "pg-tls",
								CAKey:      "ca.crt",
							},
						},
					},
				},
			},
		}
		vols, mounts := getPostgresTLSVolumeAndMount(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal("postgres-tls"))
		Expect(vols[0].Secret).ToNot(BeNil())
		Expect(vols[0].Secret.SecretName).To(Equal("pg-tls"))
		Expect(mounts).To(HaveLen(1))
		Expect(mounts[0].MountPath).To(Equal("/etc/bindplane/postgres-tls"))
	})
})

// envVarSecretKeyRef returns the SecretKeySelector for the env var with the given name, or nil.
func envVarSecretKeyRef(envVars []corev1.EnvVar, name string) *corev1.SecretKeySelector {
	for _, ev := range envVars {
		if ev.Name == name && ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
			return ev.ValueFrom.SecretKeyRef
		}
	}
	return nil
}

var _ = Describe("getTSDBEnvVars", func() {
	It("returns enable_remote, host, port, and username/password from generated secret when internal TLS is disabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVars).To(HaveLen(6))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_REMOTE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_HOST")).To(Equal("my-bp-tsdb.default.svc"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("9090"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_TYPE")).To(Equal("basic"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")).To(Equal("(secret)"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")).To(Equal("(secret)"))
		refUser := envVarSecretKeyRef(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")
		refPass := envVarSecretKeyRef(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")
		Expect(refUser).ToNot(BeNil())
		Expect(refPass).ToNot(BeNil())
		Expect(refUser.Name).To(Equal("my-bp-tsdb-basic-auth"))
		Expect(refUser.Key).To(Equal(tsdbBasicAuthSecretKeyUser))
		Expect(refPass.Name).To(Equal("my-bp-tsdb-basic-auth"))
		Expect(refPass.Key).To(Equal(tsdbBasicAuthSecretKeyPass))
	})
	It("uses remote Prometheus env vars and skips operator-generated basic auth when remote.enable is true", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable:          true,
							Host:            "vm.example.internal",
							QueryPathPrefix: "/select/0/prometheus",
							RemoteWrite: &bindplanev1alpha1.TSDBRemoteWriteConfig{
								Host: "vm-write.example.internal",
								Port: 8480,
							},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_REMOTE")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_HOST")).To(Equal("vm.example.internal"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("9090"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_QUERY_PATH_PREFIX")).To(Equal("/select/0/prometheus"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_HOST")).To(Equal("vm-write.example.internal"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_PORT")).To(Equal("8480"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_ENDPOINT")).To(Equal("/api/v1/write"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_TYPE")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_USERNAME")).To(BeEmpty())
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_AUTH_PASSWORD")).To(BeEmpty())
	})
	It("uses remote write endpoint override when provided", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable: true,
							Host:   "vm.example.internal",
							Port:   8080,
							RemoteWrite: &bindplanev1alpha1.TSDBRemoteWriteConfig{
								Host:     "vm-write.example.internal",
								Port:     18480,
								Endpoint: "/api/v1/push",
							},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_PORT")).To(Equal("8080"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_REMOTE_WRITE_ENDPOINT")).To(Equal("/api/v1/push"))
	})
	It("adds Prometheus TLS env vars when cert-manager TLS is enabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer", Kind: "ClusterIssuer"},
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVars).To(HaveLen(10))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_ENABLE_TLS")).To(Equal("true"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_CERT")).To(Equal(internalTLSTSDBClientMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_KEY")).To(Equal(internalTLSTSDBClientMountPath + "/tls.key"))
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_CA")).To(Equal(internalTLSTSDBClientMountPath + "/ca.crt"))
	})
	It("adds BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY when prometheus TLS skipVerify is true", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca-issuer"},
							SkipVerify:  true,
						},
					},
				},
			},
		}
		envVars := getTSDBEnvVars(bindplane)
		Expect(envVarByName(envVars, "BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY")).To(Equal("true"))
		Expect(envVars).To(HaveLen(11))
	})
})

var _ = Describe("reconcileTSDB", func() {
	It("short-circuits when remote Prometheus mode is enabled", func() {
		reconciler := &BindplaneReconciler{}
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "my-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						Remote: &bindplanev1alpha1.TSDBRemoteConfig{
							Enable: true,
							Host:   "vm.example.internal",
						},
					},
				},
			},
		}
		Expect(reconciler.reconcileTSDB(context.Background(), bindplane, logf.Log.WithName("test"))).To(Succeed())
	})
})

var _ = Describe("validateTSDBTLSConfig (controller_test)", func() {
	It("returns nil when config.Prometheus is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{Spec: bindplanev1alpha1.BindplaneSpec{Config: bindplanev1alpha1.BindplaneConfigSpec{License: "x", Store: bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}}}}}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
	It("returns error when both secretName and certManager are set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							SecretName:  "x",
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "issuer"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).NotTo(Succeed())
		Expect(validateTSDBTLSConfig(bindplane).Error()).To(ContainSubstring("mutually exclusive"))
	})
	It("returns nil when CertManager has a valid name", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "my-issuer", Kind: "Issuer"},
						},
					},
				},
			},
		}
		Expect(validateTSDBTLSConfig(bindplane)).To(Succeed())
	})
})

var _ = Describe("getInternalTLSVolumesAndMounts", func() {
	It("returns nil when cert-manager TLS is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getInternalTLSVolumesAndMounts(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})
	It("returns one volume and one mount when cert-manager TLS is enabled", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "x",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
					TSDB: &bindplanev1alpha1.TSDBConfig{
						TLS: &bindplanev1alpha1.TSDBTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ca"},
						},
					},
				},
			},
		}
		vols, mounts := getInternalTLSVolumesAndMounts(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(mounts).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(internalTLSTSDBClientVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("bp-tsdb-remote-write-client"))
		Expect(mounts[0].Name).To(Equal(internalTLSTSDBClientVolumeName))
		Expect(mounts[0].MountPath).To(Equal(internalTLSTSDBClientMountPath))
	})
})

var _ = Describe("getNatsTLSEnvVars", func() {
	It("returns nil when NATS TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		envVars := getNatsTLSEnvVars(bindplane)
		Expect(envVars).To(BeNil())
	})
	It("returns NATS TLS env vars when spec.config.nats.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer"},
						},
					},
				},
			},
		}
		envVars := getNatsTLSEnvVars(bindplane)
		Expect(envVars).NotTo(BeNil())
		Expect(envVarByName(envVars, bindplaneNatsEnableTLSEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneNatsTLSCertEnvVar)).To(Equal(internalTLSNatsMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, bindplaneNatsTLSKeyEnvVar)).To(Equal(internalTLSNatsMountPath + "/tls.key"))
		Expect(envVarByName(envVars, bindplaneNatsTLSCAEnvVar)).To(Equal(internalTLSNatsMountPath + "/ca.crt"))
		Expect(envVarByName(envVars, "BINDPLANE_NATS_TLS_SKIP_VERIFY")).To(BeEmpty())
	})
})

var _ = Describe("getNatsTLSVolumesAndMounts", func() {
	It("returns nil when NATS TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getNatsTLSVolumesAndMounts(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})
	It("returns one volume and one mount when spec.config.nats.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Nats: &bindplanev1alpha1.NatsConfig{
						TLS: &bindplanev1alpha1.NatsTLSConfig{
							CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "nats-issuer"},
						},
					},
				},
			},
		}
		vols, mounts := getNatsTLSVolumesAndMounts(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(mounts).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(internalTLSNatsVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("bp-nats-tls"))
		Expect(mounts[0].Name).To(Equal(internalTLSNatsVolumeName))
		Expect(mounts[0].MountPath).To(Equal(internalTLSNatsMountPath))
	})
})

var _ = Describe("getTransformAgentTLSEnvVars", func() {
	It("returns nil when Transform Agent TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		envVars := getTransformAgentTLSEnvVars(bindplane)
		Expect(envVars).To(BeNil())
	})

	It("returns Transform Agent TLS env vars when spec.transformAgent.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
					},
				},
			},
		}
		envVars := getTransformAgentTLSEnvVars(bindplane)
		Expect(envVars).NotTo(BeNil())
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSCertEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSKeyEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/tls.key"))
		Expect(envVarByName(envVars, bindplaneTransformAgentTLSCAEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/ca.crt"))
	})
})

var _ = Describe("getTransformAgentTLSVolumesAndMounts", func() {
	It("returns nil when Transform Agent TLS cert-manager is not configured", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getTransformAgentTLSVolumesAndMounts(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when spec.transformAgent.tls.certManager is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
					TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
					},
				},
			},
		}
		vols, mounts := getTransformAgentTLSVolumesAndMounts(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(mounts).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(internalTLSTransformAgentVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("bp-transform-agent-tls"))
		Expect(mounts[0].Name).To(Equal(internalTLSTransformAgentVolumeName))
		Expect(mounts[0].MountPath).To(Equal(internalTLSTransformAgentMountPath))
	})
})

var _ = Describe("bindplaneJobsMigrateJob", func() {
	It("mounts Transform Agent TLS when enabled", func() {
		bindplane := newTestBindplane("bp-migrate-ta-tls", "default")
		bindplane.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{
			TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
				CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
			},
		}

		job := newReconciler().bindplaneJobsMigrateJob(bindplane)

		Expect(job.Spec.Template.Spec.Volumes).To(ContainElement(
			And(
				HaveField("Name", Equal(internalTLSTransformAgentVolumeName)),
				HaveField("VolumeSource.Secret.SecretName", Equal("bp-migrate-ta-tls-transform-agent-tls")),
			),
		))
		Expect(job.Spec.Template.Spec.Containers[0].VolumeMounts).To(ContainElement(
			And(
				HaveField("Name", Equal(internalTLSTransformAgentVolumeName)),
				HaveField("MountPath", Equal(internalTLSTransformAgentMountPath)),
			),
		))
		Expect(envVarByName(job.Spec.Template.Spec.Containers[0].Env, bindplaneTransformAgentTLSCertEnvVar)).To(Equal(internalTLSTransformAgentMountPath + "/tls.crt"))
	})
})

var _ = Describe("generateTSDBBasicAuthSecretData", func() {
	It("returns username, password, and web-config with bcrypt hash", func() {
		data, err := generateTSDBBasicAuthSecretData()
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyUser))
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyPass))
		Expect(data).To(HaveKey(tsdbBasicAuthSecretKeyWeb))
		Expect(string(data[tsdbBasicAuthSecretKeyUser])).To(Equal(tsdbBasicAuthUsername))
		Expect(data[tsdbBasicAuthSecretKeyPass]).To(HaveLen(32))
		webConfig := string(data[tsdbBasicAuthSecretKeyWeb])
		Expect(webConfig).To(ContainSubstring("basic_auth_users:"))
		Expect(webConfig).To(ContainSubstring(tsdbBasicAuthUsername + ":"))
		Expect(webConfig).To(MatchRegexp(`\$2[aby]\$\d{2}\$`)) // bcrypt hash prefix
	})
})

var _ = Describe("getAdvancedConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	It("does not set advanced env vars when Advanced is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchFlushIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsWorkerCountEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsEnableSortingEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsMetricChannelSizeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchChannelSizeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedServerMaxRequestBytesEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedServerOpAMPShutdownGracePeriodEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedCacheTypeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisAddressEnvVar)).To(BeEmpty())
	})

	It("sets StoreStats env vars when all fields are non-zero/non-empty/true", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Store: &bindplanev1alpha1.AdvancedStoreConfig{
				Stats: &bindplanev1alpha1.AdvancedStoreStatsConfig{
					BatchFlushInterval: "2s",
					WorkerCount:        4,
					EnableSorting:      true,
					MetricChannelSize:  100,
					BatchChannelSize:   50,
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchFlushIntervalEnvVar)).To(Equal("2s"))
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsWorkerCountEnvVar)).To(Equal("4"))
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsEnableSortingEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsMetricChannelSizeEnvVar)).To(Equal("100"))
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchChannelSizeEnvVar)).To(Equal("50"))
	})

	It("does not set StoreStats int fields when zero or bool when false", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Store: &bindplanev1alpha1.AdvancedStoreConfig{
				Stats: &bindplanev1alpha1.AdvancedStoreStatsConfig{},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchFlushIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsWorkerCountEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsEnableSortingEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsMetricChannelSizeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedStoreStatsBatchChannelSizeEnvVar)).To(BeEmpty())
	})

	It("sets Server env vars when maxRequestBytes and opampShutdownGracePeriod are set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Server: &bindplanev1alpha1.AdvancedServerConfig{
				MaxRequestBytes:          20971520,
				OpAMPShutdownGracePeriod: "60s",
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedServerMaxRequestBytesEnvVar)).To(Equal("20971520"))
		Expect(envVarByName(envVars, bindplaneAdvancedServerOpAMPShutdownGracePeriodEnvVar)).To(Equal("60s"))
	})

	It("does not set Server env vars when maxRequestBytes is zero and opampShutdownGracePeriod is empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Server: &bindplanev1alpha1.AdvancedServerConfig{},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedServerMaxRequestBytesEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedServerOpAMPShutdownGracePeriodEnvVar)).To(BeEmpty())
	})

	It("sets Cache type env var when type is non-empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{Type: "redis"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheTypeEnvVar)).To(Equal("redis"))
	})

	It("sets Redis address, plain password, readTimeout, writeTimeout, and enableTLS", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Type: "redis",
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
					Address:      "redis.default.svc:6379",
					Password:     "secret",
					DB:           2,
					ReadTimeout:  "3s",
					WriteTimeout: "3s",
					EnableTLS:    true,
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisAddressEnvVar)).To(Equal("redis.default.svc:6379"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisPasswordEnvVar)).To(Equal("secret"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisDBEnvVar)).To(Equal("2"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisReadTimeoutEnvVar)).To(Equal("3s"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisWriteTimeoutEnvVar)).To(Equal("3s"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisEnableTLSEnvVar)).To(Equal("true"))
	})

	It("sources Redis password from SecretRef when PasswordSecretRef is set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Type: "redis",
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
					Address: "redis.default.svc:6379",
					PasswordSecretRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "redis-secret"},
						Key:                  "password",
					},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		ref := envVarSecretKeyRef(envVars, bindplaneAdvancedCacheRedisPasswordEnvVar)
		Expect(ref).NotTo(BeNil())
		Expect(ref.Name).To(Equal("redis-secret"))
		Expect(ref.Key).To(Equal("password"))
	})

	It("does not set Redis DB env var when DB is zero", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{Address: "redis:6379", DB: 0},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisDBEnvVar)).To(BeEmpty())
	})

	It("does not set BINDPLANE_ADVANCED_CACHE_REDIS_ENABLE_TLS when enableTLS is false", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{Address: "redis:6379", EnableTLS: false},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisEnableTLSEnvVar)).To(BeEmpty())
	})

	It("sets Redis TLS cert/key/ca paths, skipVerify, and minTLSVersion when TLS secret is configured", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
					Address:   "redis:6379",
					EnableTLS: true,
					TLS: &bindplanev1alpha1.AdvancedCacheRedisTLSConfig{
						SecretName:    "redis-tls",
						CertKey:       "tls.crt",
						KeyKey:        "tls.key",
						CAKey:         "ca.crt",
						SkipVerify:    true,
						MinTLSVersion: "1.3",
					},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSCertEnvVar)).To(Equal(advancedCacheRedisTLSMountPath + "/tls.crt"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSKeyEnvVar)).To(Equal(advancedCacheRedisTLSMountPath + "/tls.key"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSCAEnvVar)).To(Equal(advancedCacheRedisTLSMountPath + "/ca.crt"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSSkipVerifyEnvVar)).To(Equal("true"))
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSMinVersionEnvVar)).To(Equal("1.3"))
	})

	It("does not set Redis TLS path env vars when TLS secretName is empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Advanced = &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
					Address:   "redis:6379",
					EnableTLS: true,
					TLS:       &bindplanev1alpha1.AdvancedCacheRedisTLSConfig{CertKey: "tls.crt"},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSCertEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSKeyEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAdvancedCacheRedisTLSCAEnvVar)).To(BeEmpty())
	})
})

var _ = Describe("getAdvancedCacheRedisTLSVolumeAndMount", func() {
	It("returns nil when Advanced is nil", func() {
		bindplane := &bindplanev1alpha1.Bindplane{ObjectMeta: metav1.ObjectMeta{Name: "bp", Namespace: "default"}}
		vols, mounts := getAdvancedCacheRedisTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns nil when Redis TLS secretName is empty", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Advanced: &bindplanev1alpha1.AdvancedConfig{
						Cache: &bindplanev1alpha1.AdvancedCacheConfig{
							Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
								Address: "redis:6379",
								TLS:     &bindplanev1alpha1.AdvancedCacheRedisTLSConfig{},
							},
						},
					},
				},
			},
		}
		vols, mounts := getAdvancedCacheRedisTLSVolumeAndMount(bindplane)
		Expect(vols).To(BeNil())
		Expect(mounts).To(BeNil())
	})

	It("returns one volume and one mount when Redis TLS secretName is set", func() {
		bindplane := &bindplanev1alpha1.Bindplane{
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					Advanced: &bindplanev1alpha1.AdvancedConfig{
						Cache: &bindplanev1alpha1.AdvancedCacheConfig{
							Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{
								Address: "redis:6379",
								TLS:     &bindplanev1alpha1.AdvancedCacheRedisTLSConfig{SecretName: "redis-tls"},
							},
						},
					},
				},
			},
		}
		vols, mounts := getAdvancedCacheRedisTLSVolumeAndMount(bindplane)
		Expect(vols).To(HaveLen(1))
		Expect(vols[0].Name).To(Equal(advancedCacheRedisTLSVolumeName))
		Expect(vols[0].Secret.SecretName).To(Equal("redis-tls"))
		Expect(mounts).To(HaveLen(1))
		Expect(mounts[0].Name).To(Equal(advancedCacheRedisTLSVolumeName))
		Expect(mounts[0].MountPath).To(Equal(advancedCacheRedisTLSMountPath))
		Expect(mounts[0].ReadOnly).To(BeTrue())
	})
})

var _ = Describe("getAgentsConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	intPtr := func(i int) *int { return &i }

	It("does not set agents env vars when Agents is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthTypeEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthSecretKeyHeadersEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthIssuerEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthAudiencesEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthRequiredClaimsEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthRequiredScopesEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthCacheTTLEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatTTLEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatExpiryIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsRebalanceIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsRebalancePercentageEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentsRebalanceJitterEnvVar)).To(BeEmpty())
	})

	It("sets auth.type when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{Type: "oauth,secretKey"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthTypeEnvVar)).To(Equal("oauth,secretKey"))
	})

	It("sets auth.secretKey.headers as comma-joined when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				SecretKey: &bindplanev1alpha1.AgentsAuthSecretKeyConfig{
					Headers: []string{"X-Bindplane-Authorization", "Authorization"},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthSecretKeyHeadersEnvVar)).To(Equal("X-Bindplane-Authorization,Authorization"))
	})

	It("does not set auth.secretKey.headers when slice is empty", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				SecretKey: &bindplanev1alpha1.AgentsAuthSecretKeyConfig{},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthSecretKeyHeadersEnvVar)).To(BeEmpty())
	})

	It("sets auth.oauth.issuer when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				OAuth: &bindplanev1alpha1.AgentsAuthOAuthConfig{Issuer: "https://auth.example.com"},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthIssuerEnvVar)).To(Equal("https://auth.example.com"))
	})

	It("sets auth.oauth.audiences as comma-joined when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				OAuth: &bindplanev1alpha1.AgentsAuthOAuthConfig{
					Audiences: []string{"aud1", "aud2"},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthAudiencesEnvVar)).To(Equal("aud1,aud2"))
	})

	It("sets auth.oauth.requiredClaims as comma-joined when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				OAuth: &bindplanev1alpha1.AgentsAuthOAuthConfig{
					RequiredClaims: []string{"claim1", "claim2"},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthRequiredClaimsEnvVar)).To(Equal("claim1,claim2"))
	})

	It("sets auth.oauth.requiredScopes as comma-joined when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				OAuth: &bindplanev1alpha1.AgentsAuthOAuthConfig{
					RequiredScopes: []string{"scope1", "scope2"},
				},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthRequiredScopesEnvVar)).To(Equal("scope1,scope2"))
	})

	It("sets auth.oauth.cacheTTL when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			Auth: &bindplanev1alpha1.AgentsAuthConfig{
				OAuth: &bindplanev1alpha1.AgentsAuthOAuthConfig{CacheTTL: "2h"},
			},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsAuthOAuthCacheTTLEnvVar)).To(Equal("2h"))
	})

	It("sets heartbeatInterval when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{HeartbeatInterval: "45s"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatIntervalEnvVar)).To(Equal("45s"))
	})

	It("sets heartbeatTTL when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{HeartbeatTTL: "2m"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatTTLEnvVar)).To(Equal("2m"))
	})

	It("sets heartbeatExpiryInterval when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{HeartbeatExpiryInterval: "1m"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsHeartbeatExpiryIntervalEnvVar)).To(Equal("1m"))
	})

	It("sets rebalanceInterval when set", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{RebalanceInterval: "30m"}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsRebalanceIntervalEnvVar)).To(Equal("30m"))
	})

	It("sets rebalancePercentage when non-nil", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{RebalancePercentage: intPtr(50)}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsRebalancePercentageEnvVar)).To(Equal("50"))
	})

	It("does not set rebalancePercentage when nil", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsRebalancePercentageEnvVar)).To(BeEmpty())
	})

	It("sets rebalanceJitter when non-nil", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{RebalanceJitter: intPtr(10)}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsRebalanceJitterEnvVar)).To(Equal("10"))
	})

	It("does not set rebalanceJitter when nil", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentsRebalanceJitterEnvVar)).To(BeEmpty())
	})
})

var _ = Describe("getAgentVersionsConfigEnvVars", func() {
	baseBindplane := func() *bindplanev1alpha1.Bindplane {
		return &bindplanev1alpha1.Bindplane{
			ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
			Spec: bindplanev1alpha1.BindplaneSpec{
				Config: bindplanev1alpha1.BindplaneConfigSpec{
					License: "license",
					Store:   bindplanev1alpha1.StoreConfig{Postgres: &bindplanev1alpha1.PostgresConfig{Host: "pg"}},
				},
			},
		}
	}

	It("does not set agentVersions env vars when agentVersions is nil", func() {
		bindplane := baseBindplane()
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentVersionsSyncIntervalEnvVar)).To(BeEmpty())
		Expect(envVarByName(envVars, bindplaneAgentVersionsClientsEnvVar)).To(BeEmpty())
	})

	It("sets agentVersions syncInterval and clients when configured", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.AgentVersions = &bindplanev1alpha1.AgentVersionsConfig{
			SyncInterval: "2h",
			Clients:      []string{"bdot", "github"},
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentVersionsSyncIntervalEnvVar)).To(Equal("2h"))
		Expect(envVarByName(envVars, bindplaneAgentVersionsClientsEnvVar)).To(Equal("bdot,github"))
	})

	It("sets agentVersions syncInterval only when clients is omitted", func() {
		bindplane := baseBindplane()
		bindplane.Spec.Config.AgentVersions = &bindplanev1alpha1.AgentVersionsConfig{
			SyncInterval: "3h",
		}
		envVars := getBindplaneCommonEnvVars(bindplane, nodeComponent)
		Expect(envVarByName(envVars, bindplaneAgentVersionsSyncIntervalEnvVar)).To(Equal("3h"))
		Expect(envVarByName(envVars, bindplaneAgentVersionsClientsEnvVar)).To(BeEmpty())
	})
})
