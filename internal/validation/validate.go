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

// Package validation provides shared validation functions for Bindplane resources.
// Both the controller and the validating webhook call into this package so that
// field-level constraints are defined exactly once.
package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"

	corev1 "k8s.io/api/core/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// maxResourceNamePrefixLen is the maximum length for the Bindplane name prefix so that
// derived resource names (e.g. "<name>-transform-agent") fit within the 63-character limit.
const maxResourceNamePrefixLen = 63 - 1 - len("transform-agent") // 47

// uuidRegex matches standard UUID format (case-insensitive).
var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// reservedExtraEnvNames is the set of exact env var names that may never be set via extraEnv
// because they are always managed by the operator.
var reservedExtraEnvNames = map[string]struct{}{
	"KUBERNETES_NAMESPACE_NAME": {},
	"KUBERNETES_POD_NAME":       {},
	"KUBERNETES_CONTAINER_NAME": {},
	"GOMEMLIMIT":                {},
	"GOMAXPROCS":                {},
}

// AllowBindplaneExtraEnv controls whether extraEnv entries with names starting with
// "BINDPLANE_" are accepted. Set this once at operator startup via the
// --allow-bindplane-extra-env flag; it is safe to read from concurrent goroutines
// after initialization because it is written only once before any webhook or
// controller goroutines are started.
var AllowBindplaneExtraEnv bool

// ValidateBindplane runs all validations for a Bindplane resource in order,
// returning the first error encountered.
func ValidateBindplane(bindplane *bindplanev1alpha1.Bindplane) error {
	if err := ValidateBindplaneName(bindplane.Name); err != nil {
		return err
	}

	if bindplane.Spec.Version == "" {
		return fmt.Errorf("spec.version must not be empty")
	}

	if bindplane.Spec.Bindplane.Replicas != nil && *bindplane.Spec.Bindplane.Replicas < 1 {
		return fmt.Errorf("spec.bindplane.replicas must be >= 1, got %d", *bindplane.Spec.Bindplane.Replicas)
	}

	if bindplane.Spec.Nats != nil && bindplane.Spec.Nats.Replicas != nil && *bindplane.Spec.Nats.Replicas < 1 {
		return fmt.Errorf("spec.nats.replicas must be >= 1, got %d", *bindplane.Spec.Nats.Replicas)
	}

	if bindplane.Spec.TransformAgent != nil && bindplane.Spec.TransformAgent.Replicas != nil && *bindplane.Spec.TransformAgent.Replicas < 1 {
		return fmt.Errorf("spec.transformAgent.replicas must be >= 1, got %d", *bindplane.Spec.TransformAgent.Replicas)
	}
	if err := ValidateTransformAgentTLSConfig(bindplane.Spec.TransformAgent); err != nil {
		return err
	}

	if err := validateAllExtraEnv(bindplane); err != nil {
		return err
	}

	if err := ValidateLicenseConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateProfilingConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateStatusConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidatePprofConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateAuthConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateMetricsConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateTracingConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidatePostgresConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateAdvancedCacheConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateAgentVersionsConfig(&bindplane.Spec.Config); err != nil {
		return err
	}

	return nil
}

// validateAllExtraEnv validates extraEnv for every component in one call,
// keeping ValidateBindplane under the cyclomatic-complexity threshold.
func validateAllExtraEnv(bindplane *bindplanev1alpha1.Bindplane) error {
	allow := AllowBindplaneExtraEnv
	if err := ValidateExtraEnv("spec.bindplane.extraEnv", bindplane.Spec.Bindplane.ExtraEnv, allow); err != nil {
		return err
	}
	if bindplane.Spec.BindplaneJobs != nil {
		if err := ValidateExtraEnv("spec.bindplaneJobs.extraEnv", bindplane.Spec.BindplaneJobs.ExtraEnv, allow); err != nil {
			return err
		}
	}
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		if err := ValidateExtraEnv("spec.bindplaneJobsMigrate.extraEnv", bindplane.Spec.BindplaneJobsMigrate.ExtraEnv, allow); err != nil {
			return err
		}
	}
	if bindplane.Spec.TransformAgent != nil {
		if err := ValidateExtraEnv("spec.transformAgent.extraEnv", bindplane.Spec.TransformAgent.ExtraEnv, allow); err != nil {
			return err
		}
	}
	if bindplane.Spec.TSDB != nil {
		if err := ValidateExtraEnv("spec.tsdb.extraEnv", bindplane.Spec.TSDB.ExtraEnv, allow); err != nil {
			return err
		}
	}
	if bindplane.Spec.Nats != nil {
		if err := ValidateExtraEnv("spec.nats.extraEnv", bindplane.Spec.Nats.ExtraEnv, allow); err != nil {
			return err
		}
	}
	return nil
}

// ValidateExtraEnv validates a list of extra environment variables.
// fieldPath is used in error messages (e.g. "spec.bindplane.extraEnv").
// allowBindplanePrefix, when false, rejects names starting with "BINDPLANE_" because
// those vars are managed by the operator. Pass true only when the operator flag
// --allow-bindplane-extra-env is set.
func ValidateExtraEnv(fieldPath string, envVars []corev1.EnvVar, allowBindplanePrefix bool) error {
	for i, ev := range envVars {
		if _, reserved := reservedExtraEnvNames[ev.Name]; reserved {
			return fmt.Errorf("%s[%d]: name %q is reserved and managed by the operator", fieldPath, i, ev.Name)
		}
		if !allowBindplanePrefix && strings.HasPrefix(ev.Name, "BINDPLANE_") {
			return fmt.Errorf("%s[%d]: name %q starts with BINDPLANE_ which is reserved for operator-managed variables; use spec.config fields instead, or start the operator with --allow-bindplane-extra-env to override", fieldPath, i, ev.Name)
		}
	}
	return nil
}

// ValidateTransformAgentTLSConfig ensures cert-manager config is complete when Transform Agent TLS is enabled.
func ValidateTransformAgentTLSConfig(transformAgent *bindplanev1alpha1.TransformAgentComponentSpec) error {
	if transformAgent == nil || transformAgent.TLS == nil {
		return nil
	}
	if transformAgent.TLS.CertManager == nil || transformAgent.TLS.CertManager.Name == "" {
		return fmt.Errorf("spec.transformAgent.tls: certManager.name is required when TLS is enabled")
	}
	return nil
}

// ValidateBindplaneName validates that the name produces valid Kubernetes resource names (DNS-1035).
func ValidateBindplaneName(name string) error {
	if name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if len(name) > maxResourceNamePrefixLen {
		return fmt.Errorf("name %q is too long: must be at most %d characters so that derived resource names (e.g. <name>-transform-agent) stay within the 63-character limit", name, maxResourceNamePrefixLen)
	}
	// Must start with a lowercase letter [a-z]
	if r := rune(name[0]); !unicode.IsLetter(r) || !unicode.IsLower(r) {
		return fmt.Errorf("name %q must start with a lowercase letter (a-z); Kubernetes resource names are DNS-1035 labels", name)
	}
	// Must end with a letter or digit [a-z0-9]
	last := name[len(name)-1]
	if last == '-' || ((last < 'a' || last > 'z') && (last < '0' || last > '9')) {
		return fmt.Errorf("name %q must end with a lowercase letter or digit (a-z, 0-9)", name)
	}
	// All characters must be [a-z0-9-]
	for i, c := range name {
		if c != '-' && !unicode.IsLower(c) && !unicode.IsDigit(c) {
			return fmt.Errorf("name %q contains invalid character %q at position %d: only lowercase letters (a-z), digits (0-9), and hyphens are allowed", name, string(c), i)
		}
	}
	return nil
}

// ValidateLicenseConfig ensures exactly one license source is configured.
func ValidateLicenseConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil {
		return fmt.Errorf("spec.config is required")
	}
	hasLicense := config.License != ""
	hasLicenseSecretRef := config.LicenseSecretRef != nil
	if hasLicense == hasLicenseSecretRef {
		return fmt.Errorf("exactly one of spec.config.license or spec.config.licenseSecretRef must be set")
	}
	return nil
}

// ValidateProfilingConfig ensures projectID is set when profiling is enabled.
func ValidateProfilingConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Profiling == nil || !config.Profiling.Enabled {
		return nil
	}
	if config.Profiling.ProjectID == "" {
		return fmt.Errorf("projectID is required when profiling is enabled")
	}
	return nil
}

// ValidateStatusConfig ensures status check keys are valid UUIDs when set inline.
func ValidateStatusConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Status == nil {
		return nil
	}
	s := config.Status
	if s.Enabled && len(s.Keys) == 0 && s.KeysSecretRef == nil {
		return fmt.Errorf("at least one key must be configured when status is enabled")
	}
	for i, key := range s.Keys {
		if !uuidRegex.MatchString(key) {
			return fmt.Errorf("spec.config.status.keys[%d]: %q is not a valid UUID", i, key)
		}
	}
	return nil
}

// ValidatePprofConfig ensures pprof endpoint is a valid host:port when set.
func ValidatePprofConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Pprof == nil || !config.Pprof.Enabled || config.Pprof.Endpoint == "" {
		return nil
	}
	if _, _, err := net.SplitHostPort(config.Pprof.Endpoint); err != nil {
		return fmt.Errorf("invalid pprof endpoint %q: %w", config.Pprof.Endpoint, err)
	}
	return nil
}

// ValidateAuthConfig validates the auth configuration.
// When auth.type is ldap or active-directory, ldap config is required.
// When auth.type is oidc, oidc config is required.
// When auth.type is system, a username and password (or their SecretRefs) are required.
func ValidateAuthConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Auth == nil {
		return nil
	}
	auth := config.Auth
	switch auth.Type {
	case "ldap", "active-directory":
		if auth.LDAP == nil {
			return fmt.Errorf("spec.config.auth.ldap is required when auth type is %q", auth.Type)
		}
		if err := validateLDAPConfig(auth.LDAP); err != nil {
			return err
		}
	case "oidc":
		if auth.OIDC == nil {
			return fmt.Errorf("spec.config.auth.oidc is required when auth type is \"oidc\"")
		}
		if err := validateOIDCConfig(auth.OIDC); err != nil {
			return err
		}
	case "system":
		hasUsername := auth.Username != ""
		hasUsernameSecretRef := auth.UsernameSecretRef != nil
		if !hasUsername && !hasUsernameSecretRef {
			return fmt.Errorf("spec.config.auth.username or spec.config.auth.usernameSecretRef is required when auth type is \"system\"")
		}
		hasPassword := auth.Password != ""
		hasPasswordSecretRef := auth.PasswordSecretRef != nil
		if !hasPassword && !hasPasswordSecretRef {
			return fmt.Errorf("spec.config.auth.password or spec.config.auth.passwordSecretRef is required when auth type is \"system\"")
		}
	}
	return nil
}

// validateLDAPConfig validates required LDAP configuration fields.
func validateLDAPConfig(ldap *bindplanev1alpha1.LDAPConfig) error {
	if ldap.Server == "" {
		return fmt.Errorf("spec.config.auth.ldap.server must not be empty")
	}
	if ldap.BaseDN == "" {
		return fmt.Errorf("spec.config.auth.ldap.baseDN must not be empty")
	}
	hasBindUser := ldap.BindUser != ""
	hasBindUserSecretRef := ldap.BindUserSecretRef != nil
	if !hasBindUser && !hasBindUserSecretRef {
		return fmt.Errorf("spec.config.auth.ldap.bindUser or spec.config.auth.ldap.bindUserSecretRef must be set")
	}
	hasBindPassword := ldap.BindPassword != ""
	hasBindPasswordSecretRef := ldap.BindPasswordSecretRef != nil
	if !hasBindPassword && !hasBindPasswordSecretRef {
		return fmt.Errorf("spec.config.auth.ldap.bindPassword or spec.config.auth.ldap.bindPasswordSecretRef must be set")
	}
	return nil
}

// validateOIDCConfig validates required OIDC configuration fields.
func validateOIDCConfig(oidc *bindplanev1alpha1.OIDCConfig) error {
	if oidc.Issuer == "" {
		return fmt.Errorf("spec.config.auth.oidc.issuer must not be empty")
	}
	hasClientID := oidc.ClientID != ""
	hasClientIDSecretRef := oidc.ClientIDSecretRef != nil
	if !hasClientID && !hasClientIDSecretRef {
		return fmt.Errorf("spec.config.auth.oidc.clientID or spec.config.auth.oidc.clientIDSecretRef must be set")
	}
	hasClientSecret := oidc.ClientSecret != ""
	hasClientSecretSecretRef := oidc.ClientSecretSecretRef != nil
	if !hasClientSecret && !hasClientSecretSecretRef {
		return fmt.Errorf("spec.config.auth.oidc.clientSecret or spec.config.auth.oidc.clientSecretSecretRef must be set")
	}
	return nil
}

// ValidateMetricsConfig validates metrics configuration.
// When type is otlp, the otlp endpoint is required.
func ValidateMetricsConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Metrics == nil || config.Metrics.Type != "otlp" {
		return nil
	}
	if config.Metrics.OTLP == nil || config.Metrics.OTLP.Endpoint == "" {
		return fmt.Errorf("spec.config.metrics.otlp.endpoint is required when metrics type is \"otlp\"")
	}
	return nil
}

// ValidateTracingConfig validates tracing configuration.
// When type is otlp, the otlp endpoint is required.
func ValidateTracingConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Tracing == nil || config.Tracing.Type != "otlp" {
		return nil
	}
	if config.Tracing.OTLP == nil || config.Tracing.OTLP.Endpoint == "" {
		return fmt.Errorf("spec.config.tracing.otlp.endpoint is required when tracing type is \"otlp\"")
	}
	return nil
}

// ValidatePostgresConfig validates that the postgres host is set.
func ValidatePostgresConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil {
		return nil
	}
	if config.Store.Postgres == nil {
		return fmt.Errorf("spec.config.store.postgres is required")
	}
	if config.Store.Postgres.Host == "" {
		return fmt.Errorf("spec.config.store.postgres.host must not be empty")
	}
	return nil
}

// ValidateAdvancedCacheConfig validates the advanced cache configuration.
// When cache type is redis, the redis address must be a valid host:port.
func ValidateAdvancedCacheConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Advanced == nil || config.Advanced.Cache == nil {
		return nil
	}
	cache := config.Advanced.Cache
	if cache.Type != "redis" {
		return nil
	}
	if cache.Redis == nil {
		return fmt.Errorf("spec.config.advanced.cache.redis is required when cache type is \"redis\"")
	}
	if _, _, err := net.SplitHostPort(cache.Redis.Address); err != nil {
		return fmt.Errorf("spec.config.advanced.cache.redis.address %q is not a valid host:port: %w", cache.Redis.Address, err)
	}
	return nil
}

// ValidateAgentVersionsConfig validates agent versions configuration.
// When syncInterval is set, it must parse as a duration of at least 1 hour.
func ValidateAgentVersionsConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.AgentVersions == nil || config.AgentVersions.SyncInterval == "" {
		return nil
	}
	d, err := time.ParseDuration(config.AgentVersions.SyncInterval)
	if err != nil {
		return fmt.Errorf("spec.config.agentVersions.syncInterval %q is not a valid duration: %w", config.AgentVersions.SyncInterval, err)
	}
	if d < time.Hour {
		return fmt.Errorf("spec.config.agentVersions.syncInterval %q must be at least 1h", config.AgentVersions.SyncInterval)
	}
	return nil
}
