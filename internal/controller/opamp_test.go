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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// findEnvVar returns the last env var with the given name from a slice,
// or nil if not found. Kubernetes uses the last occurrence when names are duplicated.
func findEnvVar(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	var found *corev1.EnvVar
	for i := range envVars {
		if envVars[i].Name == name {
			found = &envVars[i]
		}
	}
	return found
}

// newTestBindplaneWithOpAMP returns a test Bindplane CR with OpAMP enabled.
func newTestBindplaneWithOpAMP(name, namespace string) *bindplanev1alpha1.Bindplane {
	bp := newTestBindplane(name, namespace)
	bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{
		Enabled: true,
	}
	return bp
}

// --- Unit tests for opampDeployment ---

var _ = Describe("opampDeployment", func() {
	var (
		r         *BindplaneReconciler
		bindplane *bindplanev1alpha1.Bindplane
	)

	BeforeEach(func() {
		r = newReconciler()
		bindplane = newTestBindplaneWithOpAMP("my-bp", "default")
	})

	It("uses opamp component name and labels", func() {
		dep := r.opampDeployment(bindplane)
		Expect(dep.Name).To(Equal("my-bp-opamp"))
		Expect(dep.Namespace).To(Equal("default"))
		Expect(dep.Labels[labelKeyComponent]).To(Equal(opampComponent))
	})

	It("uses the correct image", func() {
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(getOpAMPImage(bindplane)))
	})

	It("sets BINDPLANE_MODE=node", func() {
		dep := r.opampDeployment(bindplane)
		containers := dep.Spec.Template.Spec.Containers
		Expect(containers).NotTo(BeEmpty())
		modeVar := findEnvVar(containers[0].Env, bindplaneModeEnvVar)
		Expect(modeVar).NotTo(BeNil())
		Expect(modeVar.Value).To(Equal(opampModeValue))
	})

	It("uses the user-specified replicas", func() {
		replicas := int32(5)
		bindplane.Spec.OpAMP.Replicas = &replicas
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Replicas).NotTo(BeNil())
		Expect(*dep.Spec.Replicas).To(Equal(int32(5)))
	})

	It("sets replicas to nil when autoscaling is enabled", func() {
		bindplane.Spec.OpAMP.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Replicas).To(BeNil())
	})

	It("uses the user-specified resources", func() {
		bindplane.Spec.OpAMP.Resources = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("500m"))
	})

	It("uses default resources when not specified", func() {
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("2"))
		Expect(dep.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("2Gi"))
	})

	It("uses the user-specified strategy", func() {
		recreate := appsv1.RecreateDeploymentStrategyType
		bindplane.Spec.OpAMP.Strategy = &appsv1.DeploymentStrategy{Type: recreate}
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Strategy.Type).To(Equal(recreate))
	})

	It("uses the user-specified minReadySeconds", func() {
		mrs := int32(120)
		bindplane.Spec.OpAMP.MinReadySeconds = &mrs
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.MinReadySeconds).To(Equal(int32(120)))
	})

	It("selector labels use opamp component", func() {
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Selector.MatchLabels[labelKeyComponent]).To(Equal(opampComponent))
	})

	It("service account name is <bindplane>-opamp", func() {
		dep := r.opampDeployment(bindplane)
		Expect(dep.Spec.Template.Spec.ServiceAccountName).To(Equal("my-bp-opamp"))
	})
})

// --- Unit tests for getOpAMPOverrideEnvVars ---

var _ = Describe("getOpAMPOverrideEnvVars", func() {
	var bindplane *bindplanev1alpha1.Bindplane

	BeforeEach(func() {
		bindplane = newTestBindplaneWithOpAMP("my-bp", "default")
	})

	It("returns empty slice when no overrides configured", func() {
		envVars := getOpAMPOverrideEnvVars(bindplane)
		Expect(envVars).To(BeEmpty())
	})

	It("sets max simultaneous connections when configured", func() {
		maxConn := int64(2000)
		bindplane.Spec.OpAMP.MaxSimultaneousConnections = &maxConn
		envVars := getOpAMPOverrideEnvVars(bindplane)
		connVar := findEnvVar(envVars, bindplaneAgentsMaxSimultaneousConnectionsEnvVar)
		Expect(connVar).NotTo(BeNil())
		Expect(connVar.Value).To(Equal("2000"))
	})

	It("does not set max simultaneous connections when not configured", func() {
		envVars := getOpAMPOverrideEnvVars(bindplane)
		connVar := findEnvVar(envVars, bindplaneAgentsMaxSimultaneousConnectionsEnvVar)
		Expect(connVar).To(BeNil())
	})

	It("sets shutdown grace period target when configured", func() {
		bindplane.Spec.OpAMP.ShutdownGracePeriodTarget = "0.6"
		envVars := getOpAMPOverrideEnvVars(bindplane)
		graceVar := findEnvVar(envVars, bindplaneAdvancedServerOpAMPShutdownGracePeriodTargetEnvVar)
		Expect(graceVar).NotTo(BeNil())
		Expect(graceVar.Value).To(Equal("0.6"))
	})

	It("does not set shutdown grace period target when not configured", func() {
		envVars := getOpAMPOverrideEnvVars(bindplane)
		graceVar := findEnvVar(envVars, bindplaneAdvancedServerOpAMPShutdownGracePeriodTargetEnvVar)
		Expect(graceVar).To(BeNil())
	})
})

// --- Env ordering tests ---

var _ = Describe("opampDeployment env ordering", func() {
	var (
		r         *BindplaneReconciler
		bindplane *bindplanev1alpha1.Bindplane
	)

	BeforeEach(func() {
		r = newReconciler()
		bindplane = newTestBindplaneWithOpAMP("my-bp", "default")
	})

	It("prepends spec.opamp.extraEnv before operator-managed env vars", func() {
		bindplane.Spec.OpAMP.ExtraEnv = []corev1.EnvVar{
			{Name: "HTTP_PROXY", Value: "http://proxy.example.com:3128"},
			{Name: "NO_PROXY", Value: "localhost"},
		}

		dep := r.opampDeployment(bindplane)
		envVars := dep.Spec.Template.Spec.Containers[0].Env

		Expect(envVars[0].Name).To(Equal("HTTP_PROXY"))
		Expect(envVars[0].Value).To(Equal("http://proxy.example.com:3128"))
		Expect(envVars[1].Name).To(Equal("NO_PROXY"))
	})

	It("operator-managed env vars win when spec.opamp.extraEnv duplicates a name", func() {
		// User attempts to override BINDPLANE_MODE — the operator-managed value
		// must appear last so Kubernetes uses it.
		bindplane.Spec.OpAMP.ExtraEnv = []corev1.EnvVar{
			{Name: bindplaneModeEnvVar, Value: "user-override"},
		}

		dep := r.opampDeployment(bindplane)
		envVars := dep.Spec.Template.Spec.Containers[0].Env

		modeVar := findEnvVar(envVars, bindplaneModeEnvVar)
		Expect(modeVar).NotTo(BeNil())
		Expect(modeVar.Value).To(Equal(opampModeValue), "operator-managed BINDPLANE_MODE must win over user override")
	})

	It("OpAMP max simultaneous connections override appears after the shared value", func() {
		// Set the shared value via config.agents
		bindplane.Spec.Config.Agents = &bindplanev1alpha1.AgentsConfig{
			MaxSimultaneousConnections: 100,
		}
		// Set the OpAMP-specific override
		maxConn := int64(2000)
		bindplane.Spec.OpAMP.MaxSimultaneousConnections = &maxConn

		dep := r.opampDeployment(bindplane)
		envVars := dep.Spec.Template.Spec.Containers[0].Env

		// Find all occurrences of BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS
		var indices []int
		for i, ev := range envVars {
			if ev.Name == bindplaneAgentsMaxSimultaneousConnectionsEnvVar {
				indices = append(indices, i)
			}
		}
		// Both should appear (shared + override), and the override must be last
		Expect(len(indices)).To(BeNumerically(">=", 2), "expected at least two occurrences (shared and override)")
		lastIdx := indices[len(indices)-1]
		Expect(envVars[lastIdx].Value).To(Equal("2000"), "last occurrence must be the OpAMP override value")
	})

})

// --- Unit tests for opampService ---

var _ = Describe("opampService", func() {
	var (
		r         *BindplaneReconciler
		bindplane *bindplanev1alpha1.Bindplane
	)

	BeforeEach(func() {
		r = newReconciler()
		bindplane = newTestBindplaneWithOpAMP("my-bp", "default")
	})

	It("uses opamp component name", func() {
		svc := r.opampService(bindplane)
		Expect(svc.Name).To(Equal("my-bp-opamp"))
		Expect(svc.Namespace).To(Equal("default"))
	})

	It("selector labels use opamp component", func() {
		svc := r.opampService(bindplane)
		Expect(svc.Spec.Selector[labelKeyComponent]).To(Equal(opampComponent))
	})

	It("exposes the http port", func() {
		svc := r.opampService(bindplane)
		Expect(svc.Spec.Ports).NotTo(BeEmpty())
		Expect(svc.Spec.Ports[0].Name).To(Equal(opampHTTPPortName))
		Expect(svc.Spec.Ports[0].Port).To(Equal(opampHTTPPort))
	})
})

// --- Integration tests for reconcileOpAMP ---

var _ = Describe("Reconcile - OpAMP", func() {
	var (
		testNamespace string
		testCtx       context.Context
	)

	BeforeEach(func() {
		testCtx = context.Background()
		testNamespace = createTestNamespace(testCtx, "test-opamp")
	})

	AfterEach(func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
		_ = k8sClient.Delete(testCtx, ns)
	})

	It("does not create OpAMP resources when opamp is not configured", func() {
		name := "bp-no-opamp"
		bp := newTestBindplane(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		dep := &appsv1.Deployment{}
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, dep)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "OpAMP deployment should not exist when not configured")
	})

	It("does not create OpAMP resources when enabled=false", func() {
		name := "bp-opamp-disabled"
		bp := newTestBindplane(name, testNamespace)
		bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Enabled: false}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		dep := &appsv1.Deployment{}
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, dep)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "OpAMP deployment should not exist when disabled")
	})

	It("creates ServiceAccount, Deployment, Service, PDB when enabled=true", func() {
		name := "bp-opamp-enabled"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		opampName := name + "-opamp"

		sa := &corev1.ServiceAccount{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, sa)).To(Succeed())

		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, dep)).To(Succeed())
		Expect(dep.Spec.Template.Spec.Containers).NotTo(BeEmpty())
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(fmt.Sprintf("ghcr.io/observiq/bindplane-ee:%s", bp.Spec.Version)))

		svc := &corev1.Service{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, svc)).To(Succeed())

		pdb := &policyv1.PodDisruptionBudget{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, pdb)).To(Succeed())
	})

	It("also creates the Node deployment alongside the OpAMP deployment", func() {
		name := "bp-opamp-both"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		nodeDep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, nodeDep)).To(Succeed())

		opampDep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, opampDep)).To(Succeed())
	})

	It("creates HPA when autoscaling is enabled", func() {
		name := "bp-opamp-hpa"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		bp.Spec.OpAMP.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, hpa)).To(Succeed())
		Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(name + "-opamp"))
		Expect(hpa.Spec.MinReplicas).NotTo(BeNil())
		Expect(*hpa.Spec.MinReplicas).To(Equal(int32(2)))
		Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))
	})

	It("does not create HPA when autoscaling is not enabled", func() {
		name := "bp-opamp-no-hpa"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, hpa)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("deletes HPA when autoscaling is disabled after being enabled", func() {
		name := "bp-opamp-hpa-del"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		bp.Spec.OpAMP.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: true}
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		hpa := &autoscalingv2.HorizontalPodAutoscaler{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, hpa)).To(Succeed())

		// Disable autoscaling
		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		updated.Spec.OpAMP.Autoscaling = &bindplanev1alpha1.NodeAutoscalingSpec{Enabled: false}
		Expect(k8sClient.Update(testCtx, updated)).To(Succeed())

		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		hpaAfter := &autoscalingv2.HorizontalPodAutoscaler{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, hpaAfter)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("deletes OpAMP resources when enabled is toggled from true to false", func() {
		name := "bp-opamp-toggle"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		opampName := name + "-opamp"
		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, dep)).To(Succeed())

		// Disable OpAMP
		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		updated.Spec.OpAMP.Enabled = false
		Expect(k8sClient.Update(testCtx, updated)).To(Succeed())

		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		depAfter := &appsv1.Deployment{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, depAfter)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "OpAMP deployment should be deleted when disabled")

		svcAfter := &corev1.Service{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, svcAfter)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "OpAMP service should be deleted when disabled")
	})

	It("deletes OpAMP resources when the opamp block is removed", func() {
		name := "bp-opamp-remove"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		opampName := name + "-opamp"
		dep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, dep)).To(Succeed())

		// Remove the OpAMP block entirely
		updated := &bindplanev1alpha1.Bindplane{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name, Namespace: testNamespace}, updated)).To(Succeed())
		updated.Spec.OpAMP = nil
		Expect(k8sClient.Update(testCtx, updated)).To(Succeed())

		_, err := r.Reconcile(testCtx, reconcileRequest(name, testNamespace))
		Expect(err).NotTo(HaveOccurred())

		depAfter := &appsv1.Deployment{}
		err = k8sClient.Get(testCtx, types.NamespacedName{Name: opampName, Namespace: testNamespace}, depAfter)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "OpAMP deployment should be deleted when block removed")
	})

	It("OpAMP deployment uses the same license env var as the Node deployment", func() {
		name := "bp-opamp-shared-config"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		bp.Spec.Config.License = "test-license-key"
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		nodeDep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-node", Namespace: testNamespace}, nodeDep)).To(Succeed())

		opampDep := &appsv1.Deployment{}
		Expect(k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, opampDep)).To(Succeed())

		nodeEnv := nodeDep.Spec.Template.Spec.Containers[0].Env
		opampEnv := opampDep.Spec.Template.Spec.Containers[0].Env

		nodeLicense := findEnvVar(nodeEnv, bindplaneLicenseEnvVar)
		opampLicense := findEnvVar(opampEnv, bindplaneLicenseEnvVar)

		Expect(nodeLicense).NotTo(BeNil())
		Expect(opampLicense).NotTo(BeNil())
		Expect(opampLicense.Value).To(Equal(nodeLicense.Value), "OpAMP and Node deployments must share the same license env var")
	})

	It("does not create PDB when disablePodDisruptionBudget=true", func() {
		name := "bp-opamp-no-pdb"
		bp := newTestBindplaneWithOpAMP(name, testNamespace)
		bp.Spec.OpAMP.DisablePodDisruptionBudget = true
		Expect(k8sClient.Create(testCtx, bp)).To(Succeed())

		r := newReconciler()
		reconcilePastMigration(testCtx, r, name, testNamespace)

		pdb := &policyv1.PodDisruptionBudget{}
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: name + "-opamp", Namespace: testNamespace}, pdb)
		Expect(errors.IsNotFound(err)).To(BeTrue(), "PDB should not be created when disabled")
	})
})
