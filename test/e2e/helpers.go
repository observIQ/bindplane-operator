//lint:file-ignore ST1001 Ginkgo and Gomega require dot imports by convention

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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint:revive
	. "github.com/onsi/gomega"    // nolint:revive
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// defaultCertManagerVersion is the pinned cert-manager release used by default.
	// Bump this when upgrading the github.com/cert-manager/cert-manager Go module in go.mod.
	// Override at runtime with CERT_MANAGER_VERSION=vX.Y.Z to test a different version.
	defaultCertManagerVersion = "v1.20.2"
	certManagerVersionEnvVar  = "CERT_MANAGER_VERSION"
	certManagerVersionURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
	projectDirE2ESuffix       = "/test/e2e"

	// defaultArgoRolloutsVersion is the pinned Argo Rollouts release used by default.
	// Must match the github.com/argoproj/argo-rollouts version in go.mod.
	// Override at runtime with ARGO_ROLLOUTS_VERSION=vX.Y.Z to test a different version.
	defaultArgoRolloutsVersion = "v1.9.0"
	argoRolloutsVersionEnvVar  = "ARGO_ROLLOUTS_VERSION"
	argoRolloutsInstallURLTmpl = "https://github.com/argoproj/argo-rollouts/releases/download/%s/install.yaml"
)

var controllerPodName string

var (
	licenseLiteralPattern       = regexp.MustCompile(`--from-literal=license=[^\s]+`)
	bearerTokenPattern          = regexp.MustCompile(`Authorization: Bearer [^'"\s]+`)
	bindplaneLicenseFlagPattern = regexp.MustCompile(`(BINDPLANE_LICENSE=)[^\s]+`)
)

func run(cmd *exec.Cmd) (string, error) {
	dir, _ := getProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := sanitizeCommand(strings.Join(cmd.Args, " "))
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

func sanitizeCommand(command string) string {
	command = licenseLiteralPattern.ReplaceAllString(command, "--from-literal=license=REDACTED")
	command = bearerTokenPattern.ReplaceAllString(command, "Authorization: Bearer REDACTED")
	command = bindplaneLicenseFlagPattern.ReplaceAllString(command, "${1}REDACTED")
	return command
}

func getProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, projectDirE2ESuffix, "")
	return wd, nil
}

// runCmd runs the provided command and returns its combined output. Used by e2e tests.
func runCmd(cmd *exec.Cmd) (string, error) {
	return run(cmd)
}

func kubectl(namespace string, args ...string) *exec.Cmd {
	cmdArgs := make([]string, 0, len(args)+2)
	if namespace != "" {
		cmdArgs = append(cmdArgs, "-n", namespace)
	}
	cmdArgs = append(cmdArgs, args...)
	return exec.Command("kubectl", cmdArgs...) // #nosec G204 -- e2e helper builds kubectl commands from test-owned inputs
}

// getNonEmptyLines converts command output into lines and drops empty elements.
func getNonEmptyLines(output string) []string {
	var res []string
	for line := range strings.SplitSeq(strings.TrimSuffix(output, "\n"), "\n") {
		if line != "" {
			res = append(res, line)
		}
	}
	return res
}

func fixturePath(name string) string {
	projectDir, err := getProjectDir()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(projectDir, "test", "e2e", "kubectl", name)
}

func applyFixture(name, namespace string) (string, error) {
	return runCmd(kubectl(namespace, "apply", "-f", fixturePath(name)))
}

func deleteFixture(name, namespace string) {
	_, _ = runCmd(kubectl(namespace, "delete", "-f", fixturePath(name), "--ignore-not-found=true"))
}

func ensureNamespace(namespace string) {
	_, err := runCmd(kubectl("", "get", "namespace", namespace))
	if err == nil {
		return
	}
	_, err = runCmd(kubectl("", "create", "namespace", namespace))
	Expect(err).NotTo(HaveOccurred(), "Failed to create namespace %s", namespace)
}

func labelNamespaceRestricted(namespace string) {
	_, err := runCmd(kubectl("", "label", "--overwrite", "namespace", namespace,
		"pod-security.kubernetes.io/enforce=restricted"))
	Expect(err).NotTo(HaveOccurred(), "Failed to label namespace %s", namespace)
}

func setupOperatorEnvironment() {
	By("creating manager namespace")
	ensureNamespace(operatorNamespace)

	By("labeling the manager namespace to enforce the restricted security policy")
	labelNamespaceRestricted(operatorNamespace)

	By("installing CRDs")
	_, err := runCmd(exec.Command("make", "install"))
	Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

	By("deploying the controller-manager")
	_, err = runCmd(exec.Command( // #nosec G204 -- e2e helper invokes make with a test-owned image tag
		"make",
		"deploy",
		fmt.Sprintf("IMG=%s", projectImage),
	))
	Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

	By("waiting for the controller-manager deployment to become available")
	waitForDeploymentAvailable(
		operatorControllerDeploymentName,
		operatorNamespace,
		defaultEventuallyLongTimeout,
	)

	By("waiting for the webhook service to have ready endpoints")
	waitForServiceExists(
		operatorWebhookServiceName,
		operatorNamespace,
		defaultEventuallyShortTimeout,
	)
	waitForServiceEndpoints(
		operatorWebhookServiceName,
		operatorNamespace,
		"9443",
		defaultEventuallyLongTimeout,
	)

	By("waiting for the validating webhook to accept Bindplane requests")
	waitForBindplaneWebhookReady(
		"bindplane-webhook-valid.yaml",
		bindplaneNamespace,
		defaultEventuallyLongTimeout,
	)
}

func teardownOperatorEnvironment() {
	By("cleaning up operator metrics ClusterRoleBinding")
	_, _ = runCmd(kubectl("", "delete", "clusterrolebinding", operatorMetricsRoleBindingName, "--ignore-not-found=true"))

	By("cleaning up curl helper pods")
	_, _ = runCmd(kubectl(
		operatorNamespace,
		"delete",
		"pod",
		"curl-metrics",
		"curl-bindplane-node",
		"--ignore-not-found=true",
	))
	_, _ = runCmd(kubectl(
		bindplaneNamespace,
		"delete",
		"pod",
		"curl-bindplane-node",
		"--ignore-not-found=true",
	))

	By("undeploying the controller-manager")
	_, _ = runCmd(exec.Command("make", "undeploy"))

	By("uninstalling CRDs")
	_, _ = runCmd(exec.Command("make", "uninstall"))

	By("removing manager namespace")
	_, _ = runCmd(kubectl("", "delete", "namespace", operatorNamespace, "--ignore-not-found=true"))
}

func ensurePostgresReady() {
	By("deploying static postgres")
	_, err := applyFixture("postgres.yaml", "")
	Expect(err).NotTo(HaveOccurred(), "Failed to apply postgres fixture")

	By("waiting for postgres statefulset to be ready")
	waitForStatefulSetReady("postgres", postgresNamespace, defaultEventuallyServiceTimeout)

	By("waiting for postgres service to exist")
	waitForServiceExists("postgres", postgresNamespace, defaultEventuallyShortTimeout)
}

func cleanupPostgres() {
	By("removing static postgres")
	deleteFixture("postgres.yaml", "")
	_, _ = runCmd(kubectl("", "delete", "namespace", postgresNamespace, "--ignore-not-found=true"))
}

func requireBindplaneLicense() string {
	license := strings.TrimSpace(os.Getenv(bindplaneLicenseEnvVar))
	if license == "" {
		Skip(fmt.Sprintf("%s is not set", bindplaneLicenseEnvVar))
	}
	return license
}

func recreateBindplaneLicenseSecret(namespace string) {
	license := requireBindplaneLicense()
	_, _ = runCmd(kubectl(namespace, "delete", "secret", bindplaneLicenseSecretName, "--ignore-not-found=true"))
	_, err := runCmd(kubectl(namespace, "create", "secret", "generic", bindplaneLicenseSecretName,
		fmt.Sprintf("--from-literal=license=%s", license)))
	Expect(err).NotTo(HaveOccurred(), "Failed to create Bindplane license secret")
}

func deleteBindplaneLicenseSecret(namespace string) {
	_, _ = runCmd(kubectl(namespace, "delete", "secret", bindplaneLicenseSecretName, "--ignore-not-found=true"))
}

//nolint:unparam
func waitForDeploymentAvailable(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(
			namespace,
			"wait",
			"deployment/"+name,
			"--for=condition=Available",
			"--timeout", timeout.String(),
		))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed(), "Deployment %s did not become available", name)
}

func waitForStatefulSetReady(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(
			namespace,
			"rollout",
			"status",
			"statefulset/"+name,
			"--timeout", timeout.String(),
		))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed(), "StatefulSet %s did not become ready", name)
}

func waitForJobComplete(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		job, err := getJob(name, namespace)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(hasJobCondition(job, batchv1.JobComplete)).To(BeTrue())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForServiceExists(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(namespace, "get", "service", name))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForServiceEndpoints(name, namespace, expectedPort string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		output, err := runCmd(kubectl(namespace, "get", "endpoints", name))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring(expectedPort))
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForBindplaneWebhookReady(fixtureName, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		output, err := runCmd(kubectl(
			namespace,
			"apply",
			"--dry-run=server",
			"-f",
			fixturePath(fixtureName),
		))
		g.Expect(err).NotTo(HaveOccurred(), output)
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

//nolint:unparam
func waitForServiceAccountExists(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(namespace, "get", "serviceaccount", name))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForPodDisruptionBudgetExists(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(namespace, "get", "poddisruptionbudget", name))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

//nolint:unparam
func waitForBindplaneFinalizer(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		bindplane, err := getBindplane(name, namespace)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(bindplane.Finalizers).To(ContainElement(bindplaneFinalizer))
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForBindplanePhase(name, namespace, phase string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		bindplane, err := getBindplane(name, namespace)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(bindplane.Status.Phase).To(Equal(phase))
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

//nolint:unparam
func waitForBindplaneCondition(
	name, namespace, conditionType string,
	status metav1.ConditionStatus,
	reason string,
	timeout time.Duration,
) {
	Eventually(func(g Gomega) {
		bindplane, err := getBindplane(name, namespace)
		g.Expect(err).NotTo(HaveOccurred())
		condition := meta.FindStatusCondition(bindplane.Status.Conditions, conditionType)
		g.Expect(condition).NotTo(BeNil())
		g.Expect(condition.Status).To(Equal(status))
		if reason != "" {
			g.Expect(condition.Reason).To(Equal(reason))
		}
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

func waitForBindplaneDeleted(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(namespace, "get", "bindplane", name))
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("NotFound"))
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

// bindplaneCRDInstalled reports whether the bindplane CRD is registered on the cluster.
// Used to short-circuit cleanup when BeforeSuite failed before CRDs were installed.
func bindplaneCRDInstalled() bool {
	_, err := runCmd(exec.Command("kubectl", "get", "crd", "bindplanes.k8s.bindplane.com")) // #nosec G204 -- fixed args
	return err == nil
}

//nolint:unparam
func cleanupBindplane(name, namespace string, timeout time.Duration) {
	if !bindplaneCRDInstalled() {
		_, _ = fmt.Fprintf(GinkgoWriter,
			"skipping cleanup of %s/%s: bindplane CRD not installed\n", namespace, name)
		return
	}
	unpauseBindplaneForCleanup(name, namespace)
	deleteBindplane(name, namespace)
	waitForBindplaneDeleted(name, namespace, timeout)
}

func getBindplane(name, namespace string) (*bindplanev1alpha1.Bindplane, error) {
	output, err := runCmd(kubectl(namespace, "get", "bindplane", name, "-o", "json"))
	if err != nil {
		return nil, err
	}
	var bindplane bindplanev1alpha1.Bindplane
	if err := json.Unmarshal([]byte(output), &bindplane); err != nil {
		return nil, err
	}
	return &bindplane, nil
}

//nolint:unparam
func getDeployment(name, namespace string) (*appsv1.Deployment, error) {
	output, err := runCmd(kubectl(namespace, "get", "deployment", name, "-o", "json"))
	if err != nil {
		return nil, err
	}
	var deployment appsv1.Deployment
	if err := json.Unmarshal([]byte(output), &deployment); err != nil {
		return nil, err
	}
	return &deployment, nil
}

//nolint:unparam
func getStatefulSet(name, namespace string) (*appsv1.StatefulSet, error) {
	output, err := runCmd(kubectl(namespace, "get", "statefulset", name, "-o", "json"))
	if err != nil {
		return nil, err
	}
	var statefulSet appsv1.StatefulSet
	if err := json.Unmarshal([]byte(output), &statefulSet); err != nil {
		return nil, err
	}
	return &statefulSet, nil
}

func getJob(name, namespace string) (*batchv1.Job, error) {
	output, err := runCmd(kubectl(namespace, "get", "job", name, "-o", "json"))
	if err != nil {
		return nil, err
	}
	var job batchv1.Job
	if err := json.Unmarshal([]byte(output), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func hasJobCondition(job *batchv1.Job, conditionType batchv1.JobConditionType) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == conditionType && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func expectFixtureApplyFailure(name, namespace, message string) {
	_, err := applyFixture(name, namespace)
	Expect(err).To(HaveOccurred(), "Expected fixture %s to be rejected", name)
	Expect(err.Error()).To(ContainSubstring(message))
}

//nolint:unparam
func deleteBindplane(name, namespace string) {
	_, err := runCmd(kubectl(
		namespace,
		"delete",
		"bindplane",
		name,
		"--ignore-not-found=true",
		"--wait=false",
	))
	Expect(err).NotTo(HaveOccurred(), "Failed to issue delete for Bindplane %s", name)
}

func unpauseBindplaneForCleanup(name, namespace string) {
	_, err := runCmd(kubectl(
		namespace,
		"annotate",
		"bindplane",
		name,
		fmt.Sprintf("%s=false", pauseReconciliationAnnotation),
		"--overwrite",
	))
	if err == nil ||
		strings.Contains(err.Error(), "NotFound") ||
		strings.Contains(err.Error(), "the server doesn't have a resource type") {
		return
	}
	Expect(err).NotTo(HaveOccurred(), "Failed to clear pause annotation on Bindplane %s during cleanup", name)
}

func currentControllerPodName() string {
	goTemplate := "{{ range .items }}" +
		"{{ if not .metadata.deletionTimestamp }}" +
		"{{ .metadata.name }}" +
		"{{ \"\\n\" }}{{ end }}{{ end }}"
	cmd := kubectl(
		operatorNamespace,
		"get",
		"pods",
		"-l",
		"control-plane=controller-manager",
		"-o",
		"go-template="+goTemplate,
	)
	output, err := runCmd(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
	podNames := getNonEmptyLines(output)
	Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
	controllerPodName = podNames[0]
	return controllerPodName
}

func serviceAccountToken(namespace, serviceAccount string) (string, error) {
	// #nosec G101 -- Kubernetes TokenRequest object body, not a credential
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	secretName := fmt.Sprintf("%s-token-request", serviceAccount)
	tokenRequestFile := filepath.Join(os.TempDir(), secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		cmd := kubectl(namespace, "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccount,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation, defaultEventuallyShortTimeout, defaultEventuallyPollInterval).Should(Succeed())

	return out, nil
}

func getPodLogs(name, namespace string) string {
	output, err := runCmd(kubectl(namespace, "logs", name))
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from pod %s", name)
	return output
}

func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	metricsOutput := getPodLogs("curl-metrics", operatorNamespace)
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

func loadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...) // #nosec G204 -- test utility, loads a local image into kind
	_, err := run(cmd)
	return err
}

func isCertManagerCRDsInstalled() bool {
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	cmd := exec.Command("kubectl", "get", "crds")
	output, err := run(cmd)
	if err != nil {
		return false
	}

	crdList := getNonEmptyLines(output)
	for _, crd := range certManagerCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

func getCertManagerVersion() string {
	version := os.Getenv(certManagerVersionEnvVar)
	if version == "" {
		return defaultCertManagerVersion
	}
	return version
}

func getCertManagerURL() string {
	return fmt.Sprintf(certManagerVersionURLTmpl, getCertManagerVersion())
}

func installCertManager() error {
	version := getCertManagerVersion()
	url := getCertManagerURL()
	_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager version %q from %s\n", version, url)
	cmd := exec.Command("kubectl", "apply", "-f", url) // #nosec G204 -- test utility, args are version-pinned constants
	if _, err := run(cmd); err != nil {
		return err
	}
	// Wait for all three cert-manager workloads to become Available before proceeding.
	// Waiting only on the webhook (as was done previously) misses cases where cainjector
	// is still starting, causing "x509: certificate signed by unknown authority" errors.
	for _, d := range []string{"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"} {
		cmd = exec.Command("kubectl", "wait", "deployment.apps/"+d, // #nosec G204 -- deployment name is a loop constant
			"--for", "condition=Available",
			"--namespace", "cert-manager",
			"--timeout", "8m",
		)
		if _, err := run(cmd); err != nil {
			dumpCertManagerDiagnostics()
			return fmt.Errorf("cert-manager deployment %s not Available: %w", d, err)
		}
	}
	// Wait for the cert-manager-webhook's CA bundle to be injected into its
	// ValidatingWebhookConfiguration. Without this, cert-manager API requests
	// fail with "x509: certificate signed by unknown authority" even after the
	// webhook deployment shows as Available.
	_, _ = fmt.Fprintf(GinkgoWriter, "Waiting for cert-manager webhook CA to be ready...\n")
	deadline := time.Now().Add(8 * time.Minute)
	for time.Now().Before(deadline) {
		cmd = exec.Command("kubectl", "get", "validatingwebhookconfiguration", // #nosec G204 -- test utility
			"cert-manager-webhook",
			"-o", `jsonpath={.webhooks[0].clientConfig.caBundle}`,
		)
		out, err := cmd.CombinedOutput()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	dumpCertManagerDiagnostics()
	return fmt.Errorf("timed out waiting for cert-manager webhook CA bundle to be injected")
}

// dumpCertManagerDiagnostics emits pod, event, and webhook state to GinkgoWriter
// to make cert-manager startup failures debuggable without re-running the suite.
func dumpCertManagerDiagnostics() {
	_, _ = fmt.Fprintln(GinkgoWriter, "===== cert-manager diagnostics =====")
	for _, args := range [][]string{
		{"get", "pods", "-n", "cert-manager", "-o", "wide"},
		{"describe", "pods", "-n", "cert-manager"},
		{"get", "events", "-n", "cert-manager", "--sort-by=.lastTimestamp"},
		{"get", "deployments", "-n", "cert-manager", "-o", "wide"},
		{"get", "validatingwebhookconfiguration", "cert-manager-webhook", "-o", "yaml"},
	} {
		out, _ := exec.Command("kubectl", args...).CombinedOutput() // #nosec G204 -- fixed diagnostic args
		_, _ = fmt.Fprintf(GinkgoWriter, "$ kubectl %s\n%s\n", strings.Join(args, " "), out)
	}
	_, _ = fmt.Fprintln(GinkgoWriter, "===== end cert-manager diagnostics =====")
}

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

func uninstallCertManager() {
	url := getCertManagerURL()
	cmd := exec.Command("kubectl", "delete", "-f", url) // #nosec G204 -- test utility, args are version-pinned constants
	if _, err := run(cmd); err != nil {
		warnError(err)
	}
}

func getArgoRolloutsVersion() string {
	version := os.Getenv(argoRolloutsVersionEnvVar)
	if version == "" {
		return defaultArgoRolloutsVersion
	}
	return version
}

func getArgoRolloutsURL() string {
	return fmt.Sprintf(argoRolloutsInstallURLTmpl, getArgoRolloutsVersion())
}

func isArgoRolloutsCRDInstalled() bool {
	cmd := exec.Command("kubectl", "get", "crd", "rollouts.argoproj.io") // #nosec G204 -- test utility
	_, err := run(cmd)
	return err == nil
}

func installArgoRollouts() error {
	version := getArgoRolloutsVersion()
	url := getArgoRolloutsURL()
	_, _ = fmt.Fprintf(GinkgoWriter, "Installing Argo Rollouts %q from %s\n", version, url)

	// Create the namespace first so that kubectl wait can target it immediately.
	// Ignore "already exists" so the function is idempotent.
	// #nosec G204 -- namespace is a constant
	if out, err := run(exec.Command("kubectl", "create", "namespace", argoRolloutsNamespace)); err != nil {
		if !strings.Contains(out, "already exists") {
			return fmt.Errorf("failed to create %s namespace: %w", argoRolloutsNamespace, err)
		}
	}

	applyCmd := exec.Command("kubectl", "apply", "--server-side", // #nosec G204 -- test utility
		"--namespace", argoRolloutsNamespace, "-f", url)
	if _, err := run(applyCmd); err != nil {
		return err
	}

	// Poll until the Deployment object exists — apply returns before objects are created.
	deadline := time.Now().Add(2 * time.Minute)
	for {
		cmd := exec.Command("kubectl", "get", "deployment/argo-rollouts", // #nosec G204 -- constant
			"--namespace", argoRolloutsNamespace)
		if _, err := run(cmd); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for argo-rollouts Deployment to be created")
		}
		time.Sleep(5 * time.Second)
	}

	cmd := exec.Command("kubectl", "wait", "deployment/argo-rollouts", // #nosec G204 -- deployment name is a constant
		"--for", "condition=Available",
		"--namespace", argoRolloutsNamespace,
		"--timeout", "8m",
	)
	if _, err := run(cmd); err != nil {
		return fmt.Errorf("argo-rollouts deployment not Available: %w", err)
	}
	return nil
}

func uninstallArgoRollouts() {
	url := getArgoRolloutsURL()
	// #nosec G204 -- test utility, args are version-pinned constants
	cmd := exec.Command("kubectl", "delete", "--namespace", argoRolloutsNamespace, "-f", url)
	if _, err := run(cmd); err != nil {
		warnError(err)
	}
}

func waitForRolloutExists(name, namespace string, timeout time.Duration) {
	Eventually(func(g Gomega) {
		_, err := runCmd(kubectl(namespace, "get", "rollout", name))
		g.Expect(err).NotTo(HaveOccurred())
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())
}

// skipMigrateJob waits for the operator to create the migrate Job for a Bindplane instance,
// then patches the Bindplane status so the migration gate is bypassed without running the Job.
// This is used in tests that have no postgres, since the migrate Job requires a real postgres host.
func skipMigrateJob(bindplaneName, namespace string, timeout time.Duration) {
	jobName := bindplaneName + "-migrate"

	var jobImage string
	By("waiting for the migrate Job to be created")
	Eventually(func(g Gomega) {
		out, err := runCmd(kubectl(namespace, "get", "job", jobName,
			"-o", "jsonpath={.spec.template.spec.containers[0].image}"))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(out)).NotTo(BeEmpty())
		jobImage = strings.TrimSpace(out)
	}, timeout, defaultEventuallyPollInterval).Should(Succeed())

	By("patching jobsMigrate status to bypass the migration gate")
	patch := fmt.Sprintf(`{"status":{"components":{"jobsMigrate":{"image":%q}}}}`, jobImage)
	_, err := runCmd(kubectl(namespace, "patch", "bindplane", bindplaneName,
		"--subresource=status", "--type=merge", "--patch", patch))
	Expect(err).NotTo(HaveOccurred(), "failed to patch status.components.jobsMigrate.image")
}

// verifyKubectlContext ensures the active kubectl context is the expected Kind
// cluster before any cluster-mutating work begins. The expected context is
// "kind-<KIND_CLUSTER>" (defaulting to "kind-bindplane-operator-test-e2e").
// Set BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT=1 to skip this check.
func verifyKubectlContext() error {
	if os.Getenv("BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT") == "1" {
		return nil
	}
	kindCluster := os.Getenv("KIND_CLUSTER")
	if kindCluster == "" {
		kindCluster = "bindplane-operator-test-e2e"
	}
	expected := "kind-" + kindCluster

	cmd := exec.Command("kubectl", "config", "current-context") // #nosec G204 -- test utility
	out, err := run(cmd)
	if err != nil {
		return fmt.Errorf("failed to read kubectl current-context: %w", err)
	}
	actual := strings.TrimSpace(out)
	if actual != expected {
		return fmt.Errorf(
			"refusing to run e2e suite: current kubectl context is %q but expected %q; "+
				"run `kubectl config use-context %s` or set BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT=1",
			actual, expected, expected,
		)
	}
	return nil
}
