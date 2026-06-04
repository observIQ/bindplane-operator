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

package validation_test

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
	"github.com/observiq/bindplane-operator/internal/validation"
)

// ---- ValidateBindplaneName ----

func TestValidateBindplaneName_RejectsEmpty(t *testing.T) {
	if err := validation.ValidateBindplaneName(""); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateBindplaneName_RejectsStartingWithNumber(t *testing.T) {
	for _, name := range []string{"7539", "123-abc"} {
		if err := validation.ValidateBindplaneName(name); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestValidateBindplaneName_RejectsStartingWithUppercase(t *testing.T) {
	if err := validation.ValidateBindplaneName("MyBindplane"); err == nil {
		t.Error("expected error for name starting with uppercase")
	}
}

func TestValidateBindplaneName_RejectsEndingWithHyphen(t *testing.T) {
	if err := validation.ValidateBindplaneName("my-name-"); err == nil {
		t.Error("expected error for name ending with hyphen")
	}
}

func TestValidateBindplaneName_RejectsInvalidCharacters(t *testing.T) {
	for _, name := range []string{"my_name", "my.name"} {
		if err := validation.ValidateBindplaneName(name); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestValidateBindplaneName_RejectsTooLong(t *testing.T) {
	// maxResourceNamePrefixLen is 47; a name of length 48 must be rejected.
	long := "a" + strings.Repeat("x", 47)
	if len(long) != 48 {
		t.Fatalf("test setup error: long name has length %d, expected 48", len(long))
	}
	if err := validation.ValidateBindplaneName(long); err == nil {
		t.Error("expected error for name that exceeds max prefix length")
	}
}

func TestValidateBindplaneName_AcceptsValid(t *testing.T) {
	for _, name := range []string{"a", "my-name", "abc-123", "bindplane"} {
		if err := validation.ValidateBindplaneName(name); err != nil {
			t.Errorf("unexpected error for valid name %q: %v", name, err)
		}
	}
}

// ---- ValidateTransformAgentTLSConfig ----

func TestValidateTransformAgentTLSConfig_AcceptsNil(t *testing.T) {
	if err := validation.ValidateTransformAgentTLSConfig(nil); err != nil {
		t.Errorf("unexpected error for nil transform agent spec: %v", err)
	}
}

func TestValidateTransformAgentTLSConfig_RejectsMissingCertManagerName(t *testing.T) {
	spec := &bindplanev1alpha1.TransformAgentComponentSpec{
		TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
			CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{},
		},
	}
	if err := validation.ValidateTransformAgentTLSConfig(spec); err == nil {
		t.Error("expected error when transform agent TLS is enabled without certManager.name")
	}
}

func TestValidateTransformAgentTLSConfig_AcceptsCertManagerName(t *testing.T) {
	spec := &bindplanev1alpha1.TransformAgentComponentSpec{
		TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
			CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{Name: "ta-issuer"},
		},
	}
	if err := validation.ValidateTransformAgentTLSConfig(spec); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateBindplane_RejectsInvalidTransformAgentTLS(t *testing.T) {
	replicas := int32(1)
	bp := &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "bindplane"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.99.1",
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				License: "test-license",
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{Host: "postgres"},
				},
			},
			Bindplane: bindplanev1alpha1.BindplaneComponentSpec{Replicas: &replicas},
			Nats:      &bindplanev1alpha1.NatsComponentSpec{Replicas: &replicas},
			TransformAgent: &bindplanev1alpha1.TransformAgentComponentSpec{
				Replicas: &replicas,
				TLS: &bindplanev1alpha1.TransformAgentTLSConfig{
					CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{},
				},
			},
		},
	}
	if err := validation.ValidateBindplane(bp); err == nil {
		t.Error("expected error for invalid transform agent TLS config")
	}
}

// ---- ValidateLicenseConfig ----

func TestValidateLicenseConfig_RejectsNeither(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{}
	if err := validation.ValidateLicenseConfig(cfg); err == nil {
		t.Error("expected error when neither license nor licenseSecretRef is set")
	}
}

func TestValidateLicenseConfig_RejectsBoth(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		License: "test-license",
		LicenseSecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "bindplane-license"},
			Key:                  "license",
		},
	}
	if err := validation.ValidateLicenseConfig(cfg); err == nil {
		t.Error("expected error when both license and licenseSecretRef are set")
	}
}

func TestValidateLicenseConfig_AcceptsDirectLicense(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{License: "test-license"}
	if err := validation.ValidateLicenseConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLicenseConfig_AcceptsSecretRef(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		LicenseSecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "bindplane-license"},
			Key:                  "license",
		},
	}
	if err := validation.ValidateLicenseConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateAuthConfig ----

func TestValidateAuthConfig_AcceptsNilAuth(t *testing.T) {
	if err := validation.ValidateAuthConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for nil auth: %v", err)
	}
}

func TestValidateAuthConfig_LDAPRequiresLDAPConfig(t *testing.T) {
	for _, authType := range []string{"ldap", "active-directory"} {
		cfg := &bindplanev1alpha1.BindplaneConfigSpec{
			Auth: &bindplanev1alpha1.AuthConfig{Type: authType},
		}
		if err := validation.ValidateAuthConfig(cfg); err == nil {
			t.Errorf("expected error for type %q without ldap config", authType)
		}
	}
}

func TestValidateAuthConfig_LDAPRejectsMissingServer(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "ldap",
			LDAP: &bindplanev1alpha1.LDAPConfig{
				BaseDN:       "dc=example,dc=com",
				BindUser:     "admin",
				BindPassword: "pass",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when ldap.server is empty")
	}
}

func TestValidateAuthConfig_LDAPRejectsMissingBaseDN(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "ldap",
			LDAP: &bindplanev1alpha1.LDAPConfig{
				Server:       "ldap.example.com",
				BindUser:     "admin",
				BindPassword: "pass",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when ldap.baseDN is empty")
	}
}

func TestValidateAuthConfig_LDAPRejectsMissingBindUser(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "ldap",
			LDAP: &bindplanev1alpha1.LDAPConfig{
				Server:       "ldap.example.com",
				BaseDN:       "dc=example,dc=com",
				BindPassword: "pass",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when ldap.bindUser and bindUserSecretRef are both absent")
	}
}

func TestValidateAuthConfig_LDAPRejectsMissingBindPassword(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "ldap",
			LDAP: &bindplanev1alpha1.LDAPConfig{
				Server:   "ldap.example.com",
				BaseDN:   "dc=example,dc=com",
				BindUser: "admin",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when ldap.bindPassword and bindPasswordSecretRef are both absent")
	}
}

func TestValidateAuthConfig_LDAPAcceptsValid(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "ldap",
			LDAP: &bindplanev1alpha1.LDAPConfig{
				Server:       "ldap.example.com",
				BaseDN:       "dc=example,dc=com",
				BindUser:     "admin",
				BindPassword: "pass",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthConfig_OIDCRequiresOIDCConfig(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{Type: "oidc"},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error for type oidc without oidc config")
	}
}

func TestValidateAuthConfig_OIDCRejectsMissingIssuer(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "oidc",
			OIDC: &bindplanev1alpha1.OIDCConfig{
				ClientID:     "client-id",
				ClientSecret: "secret",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when oidc.issuer is empty")
	}
}

func TestValidateAuthConfig_OIDCRejectsMissingClientID(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "oidc",
			OIDC: &bindplanev1alpha1.OIDCConfig{
				Issuer:       "https://issuer.example.com",
				ClientSecret: "secret",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when oidc.clientID and clientIDSecretRef are both absent")
	}
}

func TestValidateAuthConfig_OIDCRejectsMissingClientSecret(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "oidc",
			OIDC: &bindplanev1alpha1.OIDCConfig{
				Issuer:   "https://issuer.example.com",
				ClientID: "client-id",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when oidc.clientSecret and clientSecretSecretRef are both absent")
	}
}

func TestValidateAuthConfig_OIDCAcceptsValid(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type: "oidc",
			OIDC: &bindplanev1alpha1.OIDCConfig{
				Issuer:       "https://issuer.example.com",
				ClientID:     "client-id",
				ClientSecret: "secret",
			},
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuthConfig_SystemRejectsMissingUsername(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type:     "system",
			Password: "pass",
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when system auth has no username")
	}
}

func TestValidateAuthConfig_SystemRejectsMissingPassword(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type:     "system",
			Username: "admin",
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err == nil {
		t.Error("expected error when system auth has no password")
	}
}

func TestValidateAuthConfig_SystemAcceptsValid(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Auth: &bindplanev1alpha1.AuthConfig{
			Type:     "system",
			Username: "admin",
			Password: "pass",
		},
	}
	if err := validation.ValidateAuthConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateMetricsConfig ----

func TestValidateMetricsConfig_AcceptsNilOrNonOTLP(t *testing.T) {
	if err := validation.ValidateMetricsConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidateMetricsConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
	if err := validation.ValidateMetricsConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		Metrics: &bindplanev1alpha1.MetricsConfig{Type: "prometheus"},
	}); err != nil {
		t.Errorf("unexpected error for prometheus type: %v", err)
	}
}

func TestValidateMetricsConfig_RejectsMissingOTLPEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Metrics: &bindplanev1alpha1.MetricsConfig{Type: "otlp"},
	}
	if err := validation.ValidateMetricsConfig(cfg); err == nil {
		t.Error("expected error when metrics type is otlp but endpoint is missing")
	}
}

func TestValidateMetricsConfig_AcceptsOTLPWithEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Metrics: &bindplanev1alpha1.MetricsConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.MetricsOTLPConfig{Endpoint: "http://collector:4317"},
		},
	}
	if err := validation.ValidateMetricsConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateTracingConfig ----

func TestValidateTracingConfig_AcceptsNilOrNonOTLP(t *testing.T) {
	if err := validation.ValidateTracingConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidateTracingConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
	if err := validation.ValidateTracingConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		Tracing: &bindplanev1alpha1.TracingConfig{Type: "google"},
	}); err != nil {
		t.Errorf("unexpected error for google type: %v", err)
	}
}

func TestValidateTracingConfig_RejectsMissingOTLPEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Tracing: &bindplanev1alpha1.TracingConfig{Type: "otlp"},
	}
	if err := validation.ValidateTracingConfig(cfg); err == nil {
		t.Error("expected error when tracing type is otlp but endpoint is missing")
	}
}

func TestValidateTracingConfig_AcceptsOTLPWithEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Tracing: &bindplanev1alpha1.TracingConfig{
			Type: "otlp",
			OTLP: &bindplanev1alpha1.TracingOTLPConfig{Endpoint: "http://collector:4317"},
		},
	}
	if err := validation.ValidateTracingConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidatePostgresConfig ----

func TestValidatePostgresConfig_RejectsNilPostgres(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Store: bindplanev1alpha1.StoreConfig{},
	}
	if err := validation.ValidatePostgresConfig(cfg); err == nil {
		t.Error("expected error when postgres is nil")
	}
}

func TestValidatePostgresConfig_RejectsEmptyHost(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Store: bindplanev1alpha1.StoreConfig{
			Postgres: &bindplanev1alpha1.PostgresConfig{},
		},
	}
	if err := validation.ValidatePostgresConfig(cfg); err == nil {
		t.Error("expected error when postgres.host is empty")
	}
}

func TestValidatePostgresConfig_AcceptsValidHost(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Store: bindplanev1alpha1.StoreConfig{
			Postgres: &bindplanev1alpha1.PostgresConfig{Host: "postgres.default.svc"},
		},
	}
	if err := validation.ValidatePostgresConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateAgentVersionsConfig ----

func TestValidateAgentVersionsConfig_AcceptsNilOrEmptyInterval(t *testing.T) {
	if err := validation.ValidateAgentVersionsConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidateAgentVersionsConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
	if err := validation.ValidateAgentVersionsConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		AgentVersions: &bindplanev1alpha1.AgentVersionsConfig{},
	}); err != nil {
		t.Errorf("unexpected error for empty syncInterval: %v", err)
	}
}

func TestValidateAgentVersionsConfig_RejectsInvalidDuration(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		AgentVersions: &bindplanev1alpha1.AgentVersionsConfig{SyncInterval: "not-a-duration"},
	}
	if err := validation.ValidateAgentVersionsConfig(cfg); err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestValidateAgentVersionsConfig_RejectsTooShort(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		AgentVersions: &bindplanev1alpha1.AgentVersionsConfig{SyncInterval: "30m"},
	}
	if err := validation.ValidateAgentVersionsConfig(cfg); err == nil {
		t.Error("expected error for syncInterval less than 1h")
	}
}

func TestValidateAgentVersionsConfig_AcceptsOneHour(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		AgentVersions: &bindplanev1alpha1.AgentVersionsConfig{SyncInterval: "1h"},
	}
	if err := validation.ValidateAgentVersionsConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAgentVersionsConfig_AcceptsMoreThanOneHour(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		AgentVersions: &bindplanev1alpha1.AgentVersionsConfig{SyncInterval: "2h30m"},
	}
	if err := validation.ValidateAgentVersionsConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateExtraEnv ----

func TestValidateExtraEnv_RejectsReservedNames(t *testing.T) {
	reserved := []string{
		"KUBERNETES_NAMESPACE_NAME",
		"KUBERNETES_POD_NAME",
		"KUBERNETES_CONTAINER_NAME",
		"GOMEMLIMIT",
		"GOMAXPROCS",
	}
	for _, name := range reserved {
		envVars := []corev1.EnvVar{{Name: name, Value: "x"}}
		if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", envVars, false); err == nil {
			t.Errorf("expected error for reserved name %q", name)
		}
		// Even with allowBindplanePrefix=true, these should still be rejected.
		if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", envVars, true); err == nil {
			t.Errorf("expected error for reserved name %q even with allowBindplanePrefix=true", name)
		}
	}
}

func TestValidateExtraEnv_RejectsBindplanePrefixByDefault(t *testing.T) {
	envVars := []corev1.EnvVar{{Name: "BINDPLANE_CUSTOM_VAR", Value: "x"}}
	if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", envVars, false); err == nil {
		t.Error("expected error for BINDPLANE_ prefix when allowBindplanePrefix=false")
	}
}

func TestValidateExtraEnv_AllowsBindplanePrefixWhenFlagSet(t *testing.T) {
	envVars := []corev1.EnvVar{{Name: "BINDPLANE_CUSTOM_VAR", Value: "x"}}
	if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", envVars, true); err != nil {
		t.Errorf("unexpected error with allowBindplanePrefix=true: %v", err)
	}
}

func TestValidateExtraEnv_AcceptsNonReservedNames(t *testing.T) {
	envVars := []corev1.EnvVar{
		{Name: "HTTP_PROXY", Value: "http://proxy.example.com:3128"},
		{Name: "HTTPS_PROXY", Value: "http://proxy.example.com:3128"},
		{Name: "NO_PROXY", Value: "localhost,127.0.0.1"},
		{Name: "GOOGLE_APPLICATION_CREDENTIALS", Value: "/var/secrets/google/key.json"},
		{Name: "OTEL_EXPORTER_OTLP_ENDPOINT", Value: "http://collector:4317"},
	}
	if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", envVars, false); err != nil {
		t.Errorf("unexpected error for valid env vars: %v", err)
	}
}

func TestValidateExtraEnv_AcceptsNil(t *testing.T) {
	if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", nil, false); err != nil {
		t.Errorf("unexpected error for nil extraEnv: %v", err)
	}
}

func TestValidateExtraEnv_AcceptsEmpty(t *testing.T) {
	if err := validation.ValidateExtraEnv("spec.bindplane.extraEnv", []corev1.EnvVar{}, false); err != nil {
		t.Errorf("unexpected error for empty extraEnv: %v", err)
	}
}

// ---- ValidateArgoRollout ----

func newArgoRolloutTestBindplane() *bindplanev1alpha1.Bindplane {
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "bindplane"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.99.1",
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				License: "test-license",
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{Host: "postgres.example.com"},
				},
			},
		},
	}
}

func TestValidateArgoRollout_AcceptsNilArgoRollout(t *testing.T) {
	bp := newArgoRolloutTestBindplane()
	if err := validation.ValidateArgoRollout(bp); err != nil {
		t.Errorf("unexpected error when argoRollout is nil: %v", err)
	}
}

func TestValidateArgoRollout_AcceptsDisabled(t *testing.T) {
	bp := newArgoRolloutTestBindplane()
	bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: false}
	if err := validation.ValidateArgoRollout(bp); err != nil {
		t.Errorf("unexpected error when argoRollout.enabled=false: %v", err)
	}
}

func TestValidateArgoRollout_AcceptsEnabledWithoutStrategy(t *testing.T) {
	bp := newArgoRolloutTestBindplane()
	bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
	if err := validation.ValidateArgoRollout(bp); err != nil {
		t.Errorf("unexpected error when argoRollout.enabled=true and strategy is nil: %v", err)
	}
}

func TestValidateArgoRollout_RejectsMutuallyExclusive(t *testing.T) {
	bp := newArgoRolloutTestBindplane()
	rollingUpdate := appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType}
	bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
	bp.Spec.Bindplane.Strategy = &rollingUpdate
	if err := validation.ValidateArgoRollout(bp); err == nil {
		t.Error("expected error when argoRollout.enabled=true and strategy is set")
	}
}

func TestValidateBindplane_RejectsArgoRolloutWithStrategy(t *testing.T) {
	bp := newArgoRolloutTestBindplane()
	rollingUpdate := appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType}
	bp.Spec.Bindplane.ArgoRollout = &bindplanev1alpha1.ArgoRolloutSpec{Enabled: true}
	bp.Spec.Bindplane.Strategy = &rollingUpdate
	if err := validation.ValidateBindplane(bp); err == nil {
		t.Error("expected ValidateBindplane to fail when argoRollout.enabled=true and strategy is set")
	}
}

// ---- ValidateImageOverrides ----

func newImageOverrideTestBindplane() *bindplanev1alpha1.Bindplane {
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "bindplane"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.99.1",
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				License: "test-license",
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{Host: "postgres.example.com"},
				},
			},
		},
	}
}

func TestValidateImageOverrides_AcceptsEmpty(t *testing.T) {
	bp := newImageOverrideTestBindplane()
	if err := validation.ValidateImageOverrides(bp); err != nil {
		t.Errorf("unexpected error with no image overrides set: %v", err)
	}
}

func TestValidateImageOverrides_AcceptsValidImages(t *testing.T) {
	valid := []string{
		"ghcr.io/observiq/bindplane-ee:1.99.1",
		"myregistry.example.com/bindplane-ee:custom-tag",
		"ghcr.io/observiq/bindplane-ee@sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
		"bindplane-ee:latest",
		"192.168.1.1:5000/bindplane-ee:1.99.1",
	}
	for _, img := range valid {
		bp := newImageOverrideTestBindplane()
		bp.Spec.Bindplane.Image = img
		if err := validation.ValidateImageOverrides(bp); err != nil {
			t.Errorf("unexpected error for valid image %q: %v", img, err)
		}
	}
}

func TestValidateImageOverrides_RejectsWhitespace(t *testing.T) {
	cases := []struct {
		name  string
		image string
	}{
		{"space", "my image:tag"},
		{"tab", "my\timage:tag"},
		{"newline", "myimage:tag\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp := newImageOverrideTestBindplane()
			bp.Spec.Bindplane.Image = tc.image
			if err := validation.ValidateImageOverrides(bp); err == nil {
				t.Errorf("expected error for image with whitespace %q", tc.image)
			}
		})
	}
}

func TestValidateImageOverrides_RejectsURLScheme(t *testing.T) {
	cases := []string{
		"http://myregistry.example.com/bindplane-ee:1.99.1",
		"https://myregistry.example.com/bindplane-ee:1.99.1",
		"docker://myregistry.example.com/bindplane-ee:1.99.1",
	}
	for _, img := range cases {
		bp := newImageOverrideTestBindplane()
		bp.Spec.Bindplane.Image = img
		if err := validation.ValidateImageOverrides(bp); err == nil {
			t.Errorf("expected error for image with URL scheme %q", img)
		}
	}
}

func TestValidateImageOverrides_ValidatesAllComponents(t *testing.T) {
	invalid := "http://bad-image"
	components := []struct {
		name   string
		mutate func(*bindplanev1alpha1.Bindplane)
	}{
		{"opamp", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.OpAMP = &bindplanev1alpha1.OpAMPComponentSpec{Image: invalid}
		}},
		{"nats", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.Nats = &bindplanev1alpha1.NatsComponentSpec{Image: invalid}
		}},
		{"bindplaneJobs", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.BindplaneJobs = &bindplanev1alpha1.BindplaneJobsComponentSpec{Image: invalid}
		}},
		{"bindplaneJobsMigrate", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.BindplaneJobsMigrate = &bindplanev1alpha1.BindplaneJobsMigrateComponentSpec{Image: invalid}
		}},
		{"transformAgent", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.TransformAgent = &bindplanev1alpha1.TransformAgentComponentSpec{Image: invalid}
		}},
		{"tsdb", func(bp *bindplanev1alpha1.Bindplane) {
			bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{Image: invalid}
		}},
	}
	for _, tc := range components {
		t.Run(tc.name, func(t *testing.T) {
			bp := newImageOverrideTestBindplane()
			tc.mutate(bp)
			if err := validation.ValidateImageOverrides(bp); err == nil {
				t.Errorf("expected error for invalid %s.image", tc.name)
			}
		})
	}
}

// ---- ValidateStatusConfig ----

func TestValidateStatusConfig_NilStatusOK(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("expected no error for nil status, got %v", err)
	}
}

func TestValidateStatusConfig_EnabledWithNoKeysOK(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{Enabled: true},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("expected no error for enabled status with no keys (operator auto-manages), got %v", err)
	}
}

func TestValidateStatusConfig_DisabledWithNoKeysOK(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{Enabled: false},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("expected no error for disabled status, got %v", err)
	}
}

func TestValidateStatusConfig_ValidInlineKeysOK(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			Keys:    []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"},
		},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("expected no error for valid inline UUID keys, got %v", err)
	}
}

func TestValidateStatusConfig_InvalidInlineKeyRejected(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			Keys:    []string{"not-a-uuid"},
		},
	}
	if err := validation.ValidateStatusConfig(cfg); err == nil {
		t.Error("expected error for invalid UUID key")
	}
}

// ---- ValidateExtraVolumes ----

func secretVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name:         name,
		VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: name + "-secret"}},
	}
}

func configMapVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: name + "-cm"},
		}},
	}
}

func emptyDirVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name:         name,
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}
}

func mount(volName, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{Name: volName, MountPath: mountPath}
}

func TestValidateExtraVolumes_ValidSecretVolume(t *testing.T) {
	vols := []corev1.Volume{secretVolume("my-ca")}
	mounts := []corev1.VolumeMount{mount("my-ca", "/etc/my-ca")}
	if err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateExtraVolumes_ValidConfigMapVolume(t *testing.T) {
	vols := []corev1.Volume{configMapVolume("prom-rules")}
	mounts := []corev1.VolumeMount{mount("prom-rules", "/etc/prometheus/rules.d")}
	if err := validation.ValidateExtraVolumes("spec.tsdb", vols, mounts, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateExtraVolumes_ValidEmptyDir(t *testing.T) {
	vols := []corev1.Volume{emptyDirVolume("scratch")}
	mounts := []corev1.VolumeMount{mount("scratch", "/tmp/scratch")}
	if err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateExtraVolumes_HostPathRejected(t *testing.T) {
	hostPath := "/host/path"
	vols := []corev1.Volume{{
		Name:         "host-vol",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: hostPath}},
	}}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, nil, nil)
	if err == nil {
		t.Error("expected error for hostPath volume")
	}
	if !strings.Contains(err.Error(), "hostPath") {
		t.Errorf("error should mention hostPath, got: %v", err)
	}
}

func TestValidateExtraVolumes_NoSourceRejected(t *testing.T) {
	vols := []corev1.Volume{{Name: "empty-source"}}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, nil, nil)
	if err == nil {
		t.Error("expected error for volume with no source")
	}
}

func TestValidateExtraVolumes_InvalidDNS1123Name(t *testing.T) {
	vols := []corev1.Volume{{
		Name:         "Invalid_Name",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, nil, nil)
	if err == nil {
		t.Error("expected error for invalid DNS-1123 name")
	}
}

func TestValidateExtraVolumes_DuplicateName(t *testing.T) {
	vols := []corev1.Volume{secretVolume("my-ca"), secretVolume("my-ca")}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, nil, nil)
	if err == nil {
		t.Error("expected error for duplicate volume name")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention duplicate, got: %v", err)
	}
}

func TestValidateExtraVolumes_ReservedNameRejected(t *testing.T) {
	for _, reservedName := range []string{"ldap-tls", "network-tls", "postgres-tls", "nats-tls", "transform-agent-tls", "tsdb-remote-write-tls"} {
		vols := []corev1.Volume{{
			Name:         reservedName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		}}
		err := validation.ValidateExtraVolumes("spec.bindplane", vols, nil, nil)
		if err == nil {
			t.Errorf("expected error for reserved volume name %q", reservedName)
		}
		if !strings.Contains(err.Error(), "reserved") {
			t.Errorf("error should mention reserved for %q, got: %v", reservedName, err)
		}
	}
}

func TestValidateExtraVolumes_AdditionalReservedName(t *testing.T) {
	extra := map[string]struct{}{"my-bp-tsdb-data": {}}
	vols := []corev1.Volume{{
		Name:         "my-bp-tsdb-data",
		VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	}}
	err := validation.ValidateExtraVolumes("spec.tsdb", vols, nil, extra)
	if err == nil {
		t.Error("expected error for additional reserved volume name")
	}
}

func TestValidateExtraVolumes_ReservedMountPathRejected(t *testing.T) {
	for _, reserved := range []string{
		"/etc/bindplane/ldap-tls",
		"/etc/bindplane/nats-tls",
		"/etc/prometheus",
		"/prometheus",
	} {
		vols := []corev1.Volume{secretVolume("my-vol")}
		mounts := []corev1.VolumeMount{mount("my-vol", reserved)}
		err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil)
		if err == nil {
			t.Errorf("expected error for reserved mount path %q", reserved)
		}
		if !strings.Contains(err.Error(), "reserved") {
			t.Errorf("error should mention reserved for path %q, got: %v", reserved, err)
		}
	}
}

func TestValidateExtraVolumes_DuplicateMountPath(t *testing.T) {
	vols := []corev1.Volume{secretVolume("vol-a"), secretVolume("vol-b")}
	mounts := []corev1.VolumeMount{mount("vol-a", "/etc/mypath"), mount("vol-b", "/etc/mypath")}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil)
	if err == nil {
		t.Error("expected error for duplicate mountPath")
	}
}

func TestValidateExtraVolumes_NonAbsoluteMountPath(t *testing.T) {
	vols := []corev1.Volume{secretVolume("my-ca")}
	mounts := []corev1.VolumeMount{mount("my-ca", "relative/path")}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil)
	if err == nil {
		t.Error("expected error for non-absolute mountPath")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Errorf("error should mention absolute, got: %v", err)
	}
}

func TestValidateExtraVolumes_MountReferencesOperatorVolume(t *testing.T) {
	// No extraVolumes defined, but tries to mount an operator-managed name
	mounts := []corev1.VolumeMount{mount("nats-tls", "/etc/custom")}
	err := validation.ValidateExtraVolumes("spec.bindplane", nil, mounts, nil)
	if err == nil {
		t.Error("expected error when mounting a volume not in extraVolumes")
	}
}

func TestValidateExtraVolumes_MountReferencesNonexistentVolume(t *testing.T) {
	vols := []corev1.Volume{secretVolume("vol-a")}
	mounts := []corev1.VolumeMount{mount("nonexistent", "/etc/custom")}
	err := validation.ValidateExtraVolumes("spec.bindplane", vols, mounts, nil)
	if err == nil {
		t.Error("expected error when mount references nonexistent extraVolume")
	}
}

func TestValidateExtraVolumes_EmptyIsValid(t *testing.T) {
	if err := validation.ValidateExtraVolumes("spec.bindplane", nil, nil, nil); err != nil {
		t.Errorf("nil/empty should be valid, got: %v", err)
	}
}

// TestValidateExtraVolumes_RedisCACertExample covers the plan's example:
// bindplane, opamp, jobs each mount a redis CA secret.
func TestValidateExtraVolumes_RedisCACertExample(t *testing.T) {
	redisVol := secretVolume("redis-ca")
	redisMount := mount("redis-ca", "/etc/redis-ca")

	for _, path := range []string{"spec.bindplane", "spec.opamp", "spec.bindplaneJobs"} {
		if err := validation.ValidateExtraVolumes(path, []corev1.Volume{redisVol}, []corev1.VolumeMount{redisMount}, nil); err != nil {
			t.Errorf("redis CA example should be valid for %s: %v", path, err)
		}
	}
}

// TestValidateExtraVolumes_TSDBRulesExample covers the plan's example:
// tsdb mounts a prometheus rules ConfigMap.
func TestValidateExtraVolumes_TSDBRulesExample(t *testing.T) {
	rulesVol := configMapVolume("prom-rules")
	rulesMount := mount("prom-rules", "/etc/prometheus/rules.d")

	if err := validation.ValidateExtraVolumes("spec.tsdb", []corev1.Volume{rulesVol}, []corev1.VolumeMount{rulesMount}, nil); err != nil {
		t.Errorf("tsdb rules example should be valid: %v", err)
	}
}

// TestValidateBindplane_ExtraVolumesValidated ensures ValidateBindplane calls volume validation.
func TestValidateBindplane_ExtraVolumesValidated(t *testing.T) {
	bp := minimalBindplane()
	bp.Spec.Bindplane.ExtraVolumes = []corev1.Volume{{
		Name:         "bad",
		VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/host"}},
	}}
	if err := validation.ValidateBindplane(bp); err == nil {
		t.Error("expected error for hostPath in extraVolumes via ValidateBindplane")
	}
}

// TestValidateBindplane_TSDBDataVolumeReserved ensures the computed TSDB data volume name is reserved.
func TestValidateBindplane_TSDBDataVolumeReserved(t *testing.T) {
	bp := minimalBindplane()
	// The TSDB data volume name is "<bindplane.Name>-tsdb-data"
	dataVolName := bp.Name + "-tsdb-data"
	bp.Spec.TSDB = &bindplanev1alpha1.TSDBComponentSpec{
		ExtraVolumes: []corev1.Volume{{
			Name:         dataVolName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		}},
	}
	if err := validation.ValidateBindplane(bp); err == nil {
		t.Errorf("expected error: tsdb data volume name %q should be reserved", dataVolName)
	}
}

// minimalBindplane returns a minimal valid Bindplane for testing validation.
func minimalBindplane() *bindplanev1alpha1.Bindplane {
	replicas := int32(1)
	return &bindplanev1alpha1.Bindplane{
		ObjectMeta: metav1.ObjectMeta{Name: "test-bp", Namespace: "default"},
		Spec: bindplanev1alpha1.BindplaneSpec{
			Version: "1.99.0",
			Bindplane: bindplanev1alpha1.BindplaneComponentSpec{
				Replicas: &replicas,
				Strategy: &appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
			},
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				License: "test-license",
				Store: bindplanev1alpha1.StoreConfig{
					Postgres: &bindplanev1alpha1.PostgresConfig{Host: "postgres"},
				},
			},
		},
	}
}
