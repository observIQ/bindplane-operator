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

	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
)

const (
	certmanagerVersion  = "v1.16.3"
	certmanagerURLTmpl  = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
	projectDirE2ESuffix = "/test/e2e"
)

func run(cmd *exec.Cmd) (string, error) {
	dir, _ := getProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
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

func installCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url) // #nosec G204 -- test utility, args are version-pinned constants
	if _, err := run(cmd); err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)
	if _, err := run(cmd); err != nil {
		return err
	}
	// Wait for the cert-manager-webhook's CA bundle to be injected into its
	// ValidatingWebhookConfiguration. Without this, cert-manager API requests
	// fail with "x509: certificate signed by unknown authority" even after the
	// webhook deployment shows as Available.
	_, _ = fmt.Fprintf(GinkgoWriter, "Waiting for cert-manager webhook CA to be ready...\n")
	deadline := time.Now().Add(5 * time.Minute)
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
	return fmt.Errorf("timed out waiting for cert-manager webhook CA bundle to be injected")
}

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

func uninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url) // #nosec G204 -- test utility, args are version-pinned constants
	if _, err := run(cmd); err != nil {
		warnError(err)
	}
}
