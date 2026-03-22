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
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	It("scales workloads and reports ready replica status", Label(ginkgoLabelExtended), func() {
		scalePatch := `{"spec":{"bindplane":{"replicas":1},"nats":{"replicas":1},"transformAgent":{"replicas":1}}}`
		patchBindplane(bindplaneName, bindplaneNamespace, scalePatch)

		waitForStatefulSetReady(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForBindplanePhase(bindplaneName, bindplaneNamespace, "Ready", defaultEventuallyLongTimeout)

		bindplane, err := getBindplane(bindplaneName, bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(bindplane.Status.NodeReadyReplicas).To(Equal(int32(1)))
		Expect(bindplane.Status.NatsReadyReplicas).To(Equal(int32(1)))
		Expect(bindplane.Status.TransformAgentReadyReplicas).To(Equal(int32(1)))

		nodeDeployment, err := getDeployment(bindplaneResourceName("node"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		natsStatefulSet, err := getStatefulSet(bindplaneResourceName("nats"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		transformAgentDeployment, err := getDeployment(bindplaneResourceName("transform-agent"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		Expect(*nodeDeployment.Spec.Replicas).To(Equal(int32(1)))
		Expect(*natsStatefulSet.Spec.Replicas).To(Equal(int32(1)))
		Expect(*transformAgentDeployment.Spec.Replicas).To(Equal(int32(1)))
	})

	It("supports disabling PodDisruptionBudgets and pausing reconciliation", Label(ginkgoLabelExtended), func() {
		disablePDBPatch := `{"spec":{"bindplane":{"disablePodDisruptionBudget":true},` +
			`"nats":{"disablePodDisruptionBudget":true},` +
			`"transformAgent":{"disablePodDisruptionBudget":true}}}`
		patchBindplane(bindplaneName, bindplaneNamespace, disablePDBPatch)

		waitForPodDisruptionBudgetDeleted(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForPodDisruptionBudgetDeleted(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyShortTimeout)
		waitForPodDisruptionBudgetDeleted(
			bindplaneResourceName("transform-agent"),
			bindplaneNamespace,
			defaultEventuallyShortTimeout,
		)

		annotateBindplane(bindplaneName, bindplaneNamespace, pauseReconciliationAnnotation, "true")
		waitForBindplaneCondition(
			bindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionFalse,
			"Paused",
			defaultEventuallyShortTimeout,
		)

		pausedAnnotationPatch := `{"spec":{"bindplane":{"podTemplate":{"metadata":{"annotations":` +
			`{"e2e.bindplane.com/paused":"true"}}}}}}`
		patchBindplane(bindplaneName, bindplaneNamespace, pausedAnnotationPatch)

		Consistently(func(g Gomega) {
			annotation, err := getDeploymentTemplateAnnotation(
				bindplaneResourceName("node"),
				bindplaneNamespace,
				"e2e.bindplane.com/paused",
			)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(annotation).To(BeEmpty())
		}, 20*time.Second, 2*time.Second).Should(Succeed())

		annotateBindplane(bindplaneName, bindplaneNamespace, pauseReconciliationAnnotation, "false")
		waitForBindplaneCondition(
			bindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionTrue,
			"Reconciled",
			defaultEventuallyLongTimeout,
		)

		Eventually(func(g Gomega) {
			annotation, err := getDeploymentTemplateAnnotation(
				bindplaneResourceName("node"),
				bindplaneNamespace,
				"e2e.bindplane.com/paused",
			)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(annotation).To(Equal("true"))
		}, defaultEventuallyLongTimeout, defaultEventuallyPollInterval).Should(Succeed())
	})

	It("reruns migration after recovery from a bad postgres host", Label(ginkgoLabelExtended), func() {
		By("breaking the postgres host and forcing migration to run again")
		badPostgresPatch := `{"spec":{"config":{"store":{"postgres":{"host":"postgres.invalid.svc",` +
			`"port":"5432","database":"postgres","username":"postgres","password":"password"}}}}}`
		patchBindplane(bindplaneName, bindplaneNamespace, badPostgresPatch)
		_, err := runCmd(kubectl(
			bindplaneNamespace,
			"delete",
			"job",
			bindplaneResourceName("migrate"),
			"--ignore-not-found=true",
		))
		Expect(err).NotTo(HaveOccurred())
		annotateBindplane(bindplaneName, bindplaneNamespace, forceMigrateAnnotation, "true")

		Eventually(func(g Gomega) {
			job, err := getJob(bindplaneResourceName("migrate"), bindplaneNamespace)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(hasJobCondition(job, "Failed")).To(BeTrue())
		}, defaultEventuallyLongTimeout, defaultEventuallyPollInterval).Should(Succeed())
		waitForBindplaneCondition(
			bindplaneName,
			bindplaneNamespace,
			"Reconciled",
			metav1.ConditionFalse,
			"MigrationFailed",
			defaultEventuallyLongTimeout,
		)

		By("restoring the postgres host and forcing migration again")
		goodPostgresPatch := `{"spec":{"config":{"store":{"postgres":{"host":"postgres.postgres.svc",` +
			`"port":"5432","database":"postgres","username":"postgres","password":"password"}}}}}`
		patchBindplane(bindplaneName, bindplaneNamespace, goodPostgresPatch)
		_, err = runCmd(kubectl(
			bindplaneNamespace,
			"delete",
			"job",
			bindplaneResourceName("migrate"),
			"--ignore-not-found=true",
		))
		Expect(err).NotTo(HaveOccurred())
		annotateBindplane(bindplaneName, bindplaneNamespace, forceMigrateAnnotation, "true")

		waitForJobComplete(bindplaneResourceName("migrate"), bindplaneNamespace, defaultEventuallyLongTimeout)
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
		waitForBindplanePhase(bindplaneName, bindplaneNamespace, "Ready", defaultEventuallyLongTimeout)
	})

	It("upgrades component images after migration completes", Label(ginkgoLabelExtended), func() {
		upgradeVersion := strings.TrimSpace(os.Getenv(bindplaneUpgradeVersionEnvVar))
		if upgradeVersion == "" {
			Skip(fmt.Sprintf("%s is not set", bindplaneUpgradeVersionEnvVar))
		}

		expectedImage := expectedEEImage(upgradeVersion)
		nodeDeployment, err := getDeployment(bindplaneResourceName("node"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		oldNodeImage := nodeDeployment.Spec.Template.Spec.Containers[0].Image

		patchBindplane(bindplaneName, bindplaneNamespace, fmt.Sprintf(`{"spec":{"version":"%s"}}`, upgradeVersion))

		Consistently(func(g Gomega) {
			bindplane, err := getBindplane(bindplaneName, bindplaneNamespace)
			g.Expect(err).NotTo(HaveOccurred())
			image, err := getDeploymentImage(bindplaneResourceName("node"), bindplaneNamespace)
			g.Expect(err).NotTo(HaveOccurred())
			if bindplane.Status.MigratedImage != expectedImage {
				g.Expect(image).To(Equal(oldNodeImage))
			}
		}, defaultEventuallyShortTimeout, 2*time.Second).Should(Succeed())

		Eventually(func(g Gomega) {
			bindplane, err := getBindplane(bindplaneName, bindplaneNamespace)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(bindplane.Status.MigratedImage).To(Equal(expectedImage))
		}, defaultEventuallyLongTimeout, defaultEventuallyPollInterval).Should(Succeed())

		waitForJobComplete(bindplaneResourceName("migrate"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("jobs"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("node"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForDeploymentAvailable(bindplaneResourceName("transform-agent"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForStatefulSetReady(bindplaneResourceName("nats"), bindplaneNamespace, defaultEventuallyLongTimeout)
		waitForStatefulSetReady(bindplaneResourceName("tsdb"), bindplaneNamespace, defaultEventuallyLongTimeout)

		nodeDeployment, err = getDeployment(bindplaneResourceName("node"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		jobsDeployment, err := getDeployment(bindplaneResourceName("jobs"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		transformAgentDeployment, err := getDeployment(bindplaneResourceName("transform-agent"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		natsStatefulSet, err := getStatefulSet(bindplaneResourceName("nats"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())
		tsdbStatefulSet, err := getStatefulSet(bindplaneResourceName("tsdb"), bindplaneNamespace)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
		Expect(jobsDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
		Expect(natsStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImage))
		Expect(transformAgentDeployment.Spec.Template.Spec.Containers[0].Image).To(
			Equal(expectedTransformAgentImage(upgradeVersion)),
		)
		Expect(tsdbStatefulSet.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedTSDBImage(upgradeVersion)))
	})
})
