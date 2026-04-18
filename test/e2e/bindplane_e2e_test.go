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

package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expectedEEImage(version string) string {
	return "ghcr.io/observiq/bindplane-ee:" + version
}

func expectedTransformAgentImage(version string) string {
	return "ghcr.io/observiq/bindplane-transform-agent:" + version + "-bindplane"
}

func expectedTSDBImage(version string) string {
	return "ghcr.io/observiq/bindplane-prometheus:" + version
}

func envVarValue(envVars []corev1.EnvVar, name string) string {
	for _, envVar := range envVars {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}

func waitForMinimalBindplaneBaseline() {
	waitForBindplaneFinalizer(bindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)
	waitForJobComplete(bindplaneResourceName("migrate"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForStatefulSetReady(bindplaneResourceName("tsdb"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForStatefulSetReady(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForDeploymentAvailable(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForDeploymentAvailable(bindplaneResourceName("jobs"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForDeploymentAvailable(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyLongTimeout)
	waitForBindplaneCondition(
		bindplaneName,
		bindplaneNamespace,
		"Reconciled",
		metav1.ConditionTrue,
		"Reconciled",
		defaultEventuallyLongTimeout,
	)
	waitForBindplanePhase(bindplaneName, bindplaneNamespace, "Ready", defaultEventuallyLongTimeout)
}

var _ = Describe("Bindplane workloads", Ordered, Label(ginkgoLabelRequiresLicense), func() {
	BeforeAll(func() {
		requireBindplaneLicense()
		ensurePostgresReady()
		recreateBindplaneLicenseSecret(bindplaneNamespace)
		cleanupBindplane(bindplaneName, bindplaneNamespace, 30*time.Second)

		By("applying the minimal Bindplane custom resource")
		_, err := applyFixture("bindplane-minimal-secret-license.yaml", bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the minimal Bindplane baseline to reconcile")
		waitForMinimalBindplaneBaseline()
	})

	AfterAll(func() {
		cleanupBindplane(bindplaneName, bindplaneNamespace, 2*time.Minute)
		deleteBindplaneLicenseSecret(bindplaneNamespace)
		cleanupPostgres()
	})

	It("reconciles a minimal Bindplane custom resource into managed workloads", func() {
		waitForBindplaneFinalizer(bindplaneName, bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForJobComplete(bindplaneResourceName("migrate"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForStatefulSetReady(bindplaneResourceName("tsdb"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForStatefulSetReady(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("jobs"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyLongTimeout)

		waitForServiceExists(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceExists(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceExists(bindplaneResourceName("tsdb"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceExists(bindplaneNATSClientServiceName(), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceExists(bindplaneNATSClusterServiceName(), bindplaneNamespace, defaultEventuallyShortTimeout)

		waitForServiceAccountExists(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceAccountExists(bindplaneResourceName("jobs"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceAccountExists(bindplaneResourceName("migrate"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceAccountExists(
			bindplaneResourceName("transform-agent"),
			bindplaneNamespace,
			defaultEventuallyShortTimeout,
		)
		waitForServiceAccountExists(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForServiceAccountExists(bindplaneResourceName("tsdb"), bindplaneNamespace, defaultEventuallyShortTimeout)

		waitForPodDisruptionBudgetExists(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForPodDisruptionBudgetExists(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForPodDisruptionBudgetExists(
			bindplaneResourceName("transform-agent"),
			bindplaneNamespace,
			defaultEventuallyShortTimeout,
		)

		waitForBindplaneCondition(
			bindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionTrue,
			"Reconciled",
			defaultEventuallyLongTimeout,
		)
		waitForBindplanePhase(bindplaneName, bindplaneNamespace, "Ready", defaultEventuallyLongTimeout)

		bindplane, err := getBindplane(bindplaneName, bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(bindplane.Spec.Version).NotTo(BeEmpty())
		Expect(bindplane.Status.MigratedImage).To(Equal(expectedEEImage(bindplane.Spec.Version)))
		Expect(bindplane.Status.NodeReadyReplicas).To(BeNumerically(">", 0))
		Expect(bindplane.Status.NatsReadyReplicas).To(BeNumerically(">", 0))
		Expect(bindplane.Status.TransformAgentReadyReplicas).To(BeNumerically(">", 0))

		nodeDeployment, err := getDeployment(bindplaneResourceName("node"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		jobsDeployment, err := getDeployment(bindplaneResourceName("jobs"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		transformAgentDeployment, err := getDeployment(bindplaneResourceName("transform-agent"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		natsStatefulSet, err := getStatefulSet(bindplaneResourceName("nats"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		tsdbStatefulSet, err := getStatefulSet(bindplaneResourceName("tsdb"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		migrateJob, err := getJob(bindplaneResourceName("migrate"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedEEImage(bindplane.Spec.Version)))
		Expect(jobsDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedEEImage(bindplane.Spec.Version)))
		Expect(natsStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedEEImage(bindplane.Spec.Version)))
		Expect(migrateJob.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedEEImage(bindplane.Spec.Version)))
		Expect(transformAgentDeployment.Spec.Template.Spec.Containers[0].Image).To(
			Equal(expectedTransformAgentImage(bindplane.Spec.Version)),
		)
		Expect(tsdbStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedTSDBImage(bindplane.Spec.Version)))
	})

	It("exposes the Bindplane node service to in-cluster clients", func() {
		By("creating a curl pod to probe the Bindplane node service")
		cmd := exec.Command("kubectl", "run", "curl-bindplane-node", "--restart=Never",
			"--namespace", bindplaneNamespace,
			"--image=curlimages/curl:latest",
			"--overrides",
			fmt.Sprintf(`{
				"spec": {
					"containers": [{
						"name": "curl",
						"image": "curlimages/curl:latest",
						"command": ["/bin/sh", "-c"],
						"args": ["curl -sv -o /dev/null http://%s.%s.svc.cluster.local:3001 2>&1 || true"],
						"securityContext": {
							"allowPrivilegeEscalation": false,
							"capabilities": {
								"drop": ["ALL"]
							},
							"runAsNonRoot": true,
							"runAsUser": 1000,
							"seccompProfile": {
								"type": "RuntimeDefault"
							}
						}
					}]
				}
			}`, bindplaneResourceName("node"), bindplaneNamespace))
		_, err := runCmd(cmd)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the curl pod to complete")
		Eventually(func(g Gomega) {
			output, err := runCmd(kubectl(
				bindplaneNamespace,
				"get",
				"pod",
				"curl-bindplane-node",
				"-o",
				"jsonpath={.status.phase}",
			))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("Succeeded"))
		}, defaultEventuallyServiceTimeout, defaultEventuallyPollInterval).Should(Succeed())

		By("verifying the node service accepted an HTTP connection")
		logs := getPodLogs("curl-bindplane-node", bindplaneNamespace)
		Expect(logs).To(SatisfyAny(
			ContainSubstring("Connected to"),
			ContainSubstring("< HTTP/"),
			ContainSubstring("Empty reply from server"),
		))
	})

	It("supports cert-manager mTLS between Bindplane and the Transform Agent", func() {
		By("creating a cert-manager issuer for Transform Agent TLS")
		_, err := applyFixture("transform-agent-tls-issuer.yaml", bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the Transform Agent CA issuer to become ready")
		Eventually(func(g Gomega) {
			output, err := runCmd(kubectl(
				bindplaneNamespace,
				"get",
				"issuer",
				"transform-agent-tls-issuer",
				"-o",
				`jsonpath={.status.conditions[?(@.type=="Ready")].status}`,
			))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("True"))
		}, defaultEventuallyLongTimeout, defaultEventuallyPollInterval).Should(Succeed())

		By("patching the Bindplane resource to enable Transform Agent TLS")
		_, err = runCmd(kubectl(
			bindplaneNamespace,
			"patch",
			"bindplane",
			bindplaneName,
			"--type=merge",
			"-p",
			`{"spec":{"transformAgent":{"tls":{"certManager":{"name":"transform-agent-tls-issuer"}}}}}`,
		))
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the certificate and secret to be issued")
		Eventually(func(g Gomega) {
			_, err := runCmd(kubectl(
				bindplaneNamespace,
				"get",
				"certificate.cert-manager.io",
				bindplaneResourceName("transform-agent-tls"),
			))
			g.Expect(err).NotTo(HaveOccurred())
			_, err = runCmd(kubectl(
				bindplaneNamespace,
				"get",
				"secret",
				bindplaneResourceName("transform-agent-tls"),
			))
			g.Expect(err).NotTo(HaveOccurred())
		}, defaultEventuallyLongTimeout, defaultEventuallyPollInterval).Should(Succeed())

		By("waiting for workloads to remain ready with Transform Agent TLS enabled")
		waitForDeploymentAvailable(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("jobs"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForStatefulSetReady(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForBindplaneCondition(
			bindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionTrue,
			"Reconciled",
			defaultEventuallyLongTimeout,
		)

		transformAgentDeployment, err := getDeployment(bindplaneResourceName("transform-agent"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		nodeDeployment, err := getDeployment(bindplaneResourceName("node"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		jobsDeployment, err := getDeployment(bindplaneResourceName("jobs"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(transformAgentDeployment.Spec.Template.Spec.Volumes).To(ContainElement(
			HaveField("Name", Equal("transform-agent-tls")),
		))
		Expect(envVarValue(transformAgentDeployment.Spec.Template.Spec.Containers[0].Env, "BINDPLANE_TRANSFORM_AGENT_TLS_CERT")).To(Equal("/etc/bindplane/transform-agent-tls/tls.crt"))
		Expect(envVarValue(nodeDeployment.Spec.Template.Spec.Containers[0].Env, "BINDPLANE_TRANSFORM_AGENT_TLS_CA")).To(Equal("/etc/bindplane/transform-agent-tls/ca.crt"))
		Expect(envVarValue(jobsDeployment.Spec.Template.Spec.Containers[0].Env, "BINDPLANE_TRANSFORM_AGENT_TLS_KEY")).To(Equal("/etc/bindplane/transform-agent-tls/tls.key"))
	})
})
