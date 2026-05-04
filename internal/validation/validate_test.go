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

// ---- ValidateProfilingConfig ----

func TestValidateProfilingConfig_AcceptsNilOrDisabled(t *testing.T) {
	if err := validation.ValidateProfilingConfig(nil); err != nil {
		t.Errorf("unexpected error for nil config: %v", err)
	}
	if err := validation.ValidateProfilingConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty config: %v", err)
	}
	if err := validation.ValidateProfilingConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		Profiling: &bindplanev1alpha1.ProfilingConfig{Enabled: false},
	}); err != nil {
		t.Errorf("unexpected error when profiling disabled: %v", err)
	}
}

func TestValidateProfilingConfig_RejectsMissingProjectID(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Profiling: &bindplanev1alpha1.ProfilingConfig{Enabled: true},
	}
	if err := validation.ValidateProfilingConfig(cfg); err == nil {
		t.Error("expected error when profiling enabled but projectID empty")
	}
}

func TestValidateProfilingConfig_AcceptsEnabledWithProjectID(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Profiling: &bindplanev1alpha1.ProfilingConfig{Enabled: true, ProjectID: "my-project"},
	}
	if err := validation.ValidateProfilingConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidatePprofConfig ----

func TestValidatePprofConfig_AcceptsNilOrDisabledOrNoEndpoint(t *testing.T) {
	if err := validation.ValidatePprofConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidatePprofConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
	if err := validation.ValidatePprofConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		Pprof: &bindplanev1alpha1.PprofConfig{Enabled: true},
	}); err != nil {
		t.Errorf("unexpected error when endpoint unset: %v", err)
	}
}

func TestValidatePprofConfig_RejectsInvalidEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Pprof: &bindplanev1alpha1.PprofConfig{Enabled: true, Endpoint: "not-host-port"},
	}
	if err := validation.ValidatePprofConfig(cfg); err == nil {
		t.Error("expected error for invalid endpoint")
	}
}

func TestValidatePprofConfig_AcceptsValidEndpoint(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Pprof: &bindplanev1alpha1.PprofConfig{Enabled: true, Endpoint: "127.0.0.1:6060"},
	}
	if err := validation.ValidatePprofConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateStatusConfig ----

func TestValidateStatusConfig_AcceptsNilOrEmpty(t *testing.T) {
	if err := validation.ValidateStatusConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidateStatusConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
}

func TestValidateStatusConfig_AcceptsDisabledWithNoKeys(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{Enabled: false},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStatusConfig_AcceptsEnabledWithValidUUIDs(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			Keys:    []string{"550e8400-e29b-41d4-a716-446655440000"},
		},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStatusConfig_AcceptsEnabledWithSecretRef(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{
			Enabled: true,
			KeysSecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
				Key:                  "keys",
			},
		},
	}
	if err := validation.ValidateStatusConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStatusConfig_RejectsEnabledWithNoKeys(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Status: &bindplanev1alpha1.StatusConfig{Enabled: true},
	}
	if err := validation.ValidateStatusConfig(cfg); err == nil {
		t.Error("expected error when enabled but no keys configured")
	}
}

func TestValidateStatusConfig_RejectsInvalidUUID(t *testing.T) {
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

// ---- ValidateAdvancedCacheConfig ----

func TestValidateAdvancedCacheConfig_AcceptsNilOrNoCache(t *testing.T) {
	if err := validation.ValidateAdvancedCacheConfig(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
	if err := validation.ValidateAdvancedCacheConfig(&bindplanev1alpha1.BindplaneConfigSpec{}); err != nil {
		t.Errorf("unexpected error for empty: %v", err)
	}
	if err := validation.ValidateAdvancedCacheConfig(&bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{},
	}); err != nil {
		t.Errorf("unexpected error for no cache: %v", err)
	}
}

func TestValidateAdvancedCacheConfig_RejectsMissingRedisConfig(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{Type: "redis"},
		},
	}
	if err := validation.ValidateAdvancedCacheConfig(cfg); err == nil {
		t.Error("expected error when cache type is redis but redis config is nil")
	}
}

func TestValidateAdvancedCacheConfig_RejectsInvalidAddress(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Type:  "redis",
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{Address: "not-host-port"},
			},
		},
	}
	if err := validation.ValidateAdvancedCacheConfig(cfg); err == nil {
		t.Error("expected error for invalid redis address")
	}
}

func TestValidateAdvancedCacheConfig_AcceptsValidAddress(t *testing.T) {
	cfg := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Cache: &bindplanev1alpha1.AdvancedCacheConfig{
				Type:  "redis",
				Redis: &bindplanev1alpha1.AdvancedCacheRedisConfig{Address: "redis.default.svc:6379"},
			},
		},
	}
	if err := validation.ValidateAdvancedCacheConfig(cfg); err != nil {
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
