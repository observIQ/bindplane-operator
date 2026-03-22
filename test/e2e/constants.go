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
	"os"
	"time"
)

const (
	operatorNamespace  = "bindplane-operator-system"
	bindplaneNamespace = "default"
	postgresNamespace  = "postgres"

	bindplaneName        = "bindplane"
	webhookBindplaneName = "bindplane-webhook"

	// #nosec G101 -- Kubernetes Secret resource names, not credentials
	bindplaneLicenseSecretName = "bindplane-license"
	// #nosec G101 -- Kubernetes Secret resource names, not credentials
	bindplanePostgresTLSSecretName = "bindplane-postgres-tls"
	bindplaneTLSIssuerName         = "bindplane-internal-issuer"
	bindplaneTLSCAName             = "bindplane-internal-ca"
	// #nosec G101 -- Kubernetes Secret resource names, not credentials
	postgresTLSServerSecretName = "postgres-server-tls"
	// #nosec G101 -- Kubernetes Secret resource names, not credentials
	postgresTLSClientSecretName = "bindplane-postgres-client"
	postgresTLSIssuerName       = "postgres-ca-issuer"
	// #nosec G101 -- Kubernetes Secret resource names, not credentials
	postgresTLSCASecretName = "postgres-ca"

	bindplaneFixtureName    = "bindplane-minimal-secret-license.yaml"
	bindplaneTLSFixtureName = "bindplane-minimal-secret-license-tls.yaml"
	postgresFixtureName     = "postgres.yaml"
	postgresTLSFixtureName  = "postgres.tls.yaml"
	tlsFixtureName          = "tls-cert-manager.yaml"

	operatorMetricsServiceAccountName = "bindplane-operator-controller-manager"
	operatorMetricsServiceName        = "bindplane-operator-controller-manager-metrics-service"
	operatorMetricsRoleBindingName    = "bindplane-operator-metrics-binding"
	operatorControllerDeploymentName  = "bindplane-operator-controller-manager"
	operatorWebhookServiceName        = "bindplane-operator-webhook-service"

	bindplaneFinalizer            = "k8s.bindplane.com/finalizer"
	pauseReconciliationAnnotation = "k8s.bindplane.com/pause-reconciliation"
	bindplaneLicenseEnvVar        = "BINDPLANE_LICENSE"
	e2eEnableTLSEnvVar            = "E2E_ENABLE_TLS"
	ginkgoLabelRequiresLicense    = "requires-license"
)

var (
	projectImage                    = "example.com/bindplane-operator:v0.0.1"
	enableTLSE2E                    = os.Getenv(e2eEnableTLSEnvVar) == "true"
	defaultEventuallyPollInterval   = time.Second
	defaultEventuallyLongTimeout    = 15 * time.Minute
	defaultEventuallyShortTimeout   = 2 * time.Minute
	defaultEventuallyServiceTimeout = 5 * time.Minute
)

type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

func bindplaneResourceName(component string) string {
	return bindplaneName + "-" + component
}

func bindplaneNATSClientServiceName() string {
	return bindplaneResourceName("nats") + "-client"
}

func bindplaneNATSClusterServiceName() string {
	return bindplaneResourceName("nats") + "-cluster"
}
