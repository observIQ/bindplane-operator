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

// dns1123LabelRegexp matches a valid DNS-1123 label (used for Kubernetes volume names).
var dns1123LabelRegexp = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// reservedExtraEnvNames is the set of exact env var names that may never be set via extraEnv
// because they are always managed by the operator.
var reservedExtraEnvNames = map[string]struct{}{
	"KUBERNETES_NAMESPACE_NAME": {},
	"KUBERNETES_POD_NAME":       {},
	"KUBERNETES_CONTAINER_NAME": {},
	"GOMEMLIMIT":                {},
	"GOMAXPROCS":                {},
}

// reservedVolumeNames is the set of static volume names always managed by the operator.
// If you add a new operator-managed volume name constant in the controller, add it here too.
var reservedVolumeNames = map[string]struct{}{
	"ldap-tls":              {},
	"network-tls":           {},
	"postgres-tls":          {},
	"tsdb-remote-write-tls": {},
	"nats-tls":              {},
	"transform-agent-tls":   {},
	"tsdb-web-config":       {},
	"tsdb-tls":              {},
	"tsdb-web-server-tls":   {},
	"tsdb-probe-client-tls": {},
	"tsdb-probe-auth":       {},
}

// reservedMountPaths is the set of mount paths always managed by the operator.
// If you add a new operator-managed mount path constant in the controller, add it here too.
var reservedMountPaths = map[string]struct{}{
	"/etc/bindplane/ldap-tls":              {},
	"/etc/bindplane/network-tls":           {},
	"/etc/bindplane/postgres-tls":          {},
	"/etc/bindplane/tsdb-remote-write-tls": {},
	"/etc/bindplane/nats-tls":              {},
	"/etc/bindplane/transform-agent-tls":   {},
	"/etc/prometheus":                      {},
	"/etc/tsdb-tls":                        {},
	"/etc/tsdb-web-tls":                    {},
	"/etc/tsdb-probe-client":               {},
	"/etc/tsdb-probe-auth":                 {},
	"/prometheus":                          {},
}

// tsdbDataVolumeSuffix mirrors the same constant in the controller package; kept in sync manually.
// Used to compute the per-CR TSDB data volume name: "<bindplane.Name>-tsdb-data".
const tsdbDataVolumeSuffix = "tsdb-data"

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

	if err := validateAllExtraVolumes(bindplane); err != nil {
		return err
	}

	if err := ValidateLicenseConfig(&bindplane.Spec.Config); err != nil {
		return err
	}
	if err := ValidateStatusConfig(&bindplane.Spec.Config); err != nil {
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
	if err := ValidateAgentVersionsConfig(&bindplane.Spec.Config); err != nil {
		return err
	}

	if err := ValidateArgoRollout(bindplane); err != nil {
		return err
	}

	if err := ValidateImageOverrides(bindplane); err != nil {
		return err
	}

	return nil
}

// ValidateImageOverrides validates per-service image override fields.
// It rejects values that contain whitespace or look like a URL (http:// / https:// scheme).
func ValidateImageOverrides(bindplane *bindplanev1alpha1.Bindplane) error {
	type imageField struct {
		path  string
		value string
	}
	fields := []imageField{
		{"spec.bindplane.image", bindplane.Spec.Bindplane.Image},
	}
	if bindplane.Spec.OpAMP != nil {
		fields = append(fields, imageField{"spec.opamp.image", bindplane.Spec.OpAMP.Image})
	}
	if bindplane.Spec.Nats != nil {
		fields = append(fields, imageField{"spec.nats.image", bindplane.Spec.Nats.Image})
	}
	if bindplane.Spec.BindplaneJobs != nil {
		fields = append(fields, imageField{"spec.bindplaneJobs.image", bindplane.Spec.BindplaneJobs.Image})
	}
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		fields = append(fields, imageField{"spec.bindplaneJobsMigrate.image", bindplane.Spec.BindplaneJobsMigrate.Image})
	}
	if bindplane.Spec.TransformAgent != nil {
		fields = append(fields, imageField{"spec.transformAgent.image", bindplane.Spec.TransformAgent.Image})
	}
	if bindplane.Spec.TSDB != nil {
		fields = append(fields, imageField{"spec.tsdb.image", bindplane.Spec.TSDB.Image})
	}
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		if strings.ContainsAny(f.value, " \t\n\r") {
			return fmt.Errorf("%s: image reference must not contain whitespace", f.path)
		}
		if strings.HasPrefix(f.value, "http://") || strings.HasPrefix(f.value, "https://") || strings.HasPrefix(f.value, "docker://") {
			return fmt.Errorf("%s: image reference must not include a URL scheme (http://, https://, docker://)", f.path)
		}
	}
	return nil
}

// ValidateArgoRollout enforces that spec.bindplane.strategy and
// spec.bindplane.argoRollout.enabled are mutually exclusive.
func ValidateArgoRollout(bindplane *bindplanev1alpha1.Bindplane) error {
	if bindplane.Spec.Bindplane.ArgoRollout == nil || !bindplane.Spec.Bindplane.ArgoRollout.Enabled {
		return nil
	}
	if bindplane.Spec.Bindplane.Strategy != nil {
		return fmt.Errorf("spec.bindplane.strategy and spec.bindplane.argoRollout.enabled are mutually exclusive (Argo Rollout uses BlueGreen)")
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
	if bindplane.Spec.OpAMP != nil {
		if err := ValidateExtraEnv("spec.opamp.extraEnv", bindplane.Spec.OpAMP.ExtraEnv, allow); err != nil {
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

// ValidateStatusConfig ensures any inline status check keys are valid UUIDs.
// Keys are optional: when status is enabled and no keys are supplied, the operator
// generates and manages a key automatically.
func ValidateStatusConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Status == nil {
		return nil
	}
	s := config.Status
	for i, key := range s.Keys {
		if !uuidRegex.MatchString(key) {
			return fmt.Errorf("spec.config.status.keys[%d]: %q is not a valid UUID", i, key)
		}
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

// validateAllExtraVolumes validates extraVolumes and extraVolumeMounts for every component.
func validateAllExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) error {
	// Compute the TSDB data volume name, which is per-CR (not static).
	tsdbDataVolumeName := bindplane.Name + "-" + tsdbDataVolumeSuffix

	if err := ValidateExtraVolumes("spec.bindplane", bindplane.Spec.Bindplane.ExtraVolumes, bindplane.Spec.Bindplane.ExtraVolumeMounts, nil); err != nil {
		return err
	}
	if bindplane.Spec.BindplaneJobs != nil {
		if err := ValidateExtraVolumes("spec.bindplaneJobs", bindplane.Spec.BindplaneJobs.ExtraVolumes, bindplane.Spec.BindplaneJobs.ExtraVolumeMounts, nil); err != nil {
			return err
		}
	}
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		if err := ValidateExtraVolumes("spec.bindplaneJobsMigrate", bindplane.Spec.BindplaneJobsMigrate.ExtraVolumes, bindplane.Spec.BindplaneJobsMigrate.ExtraVolumeMounts, nil); err != nil {
			return err
		}
	}
	if bindplane.Spec.TransformAgent != nil {
		if err := ValidateExtraVolumes("spec.transformAgent", bindplane.Spec.TransformAgent.ExtraVolumes, bindplane.Spec.TransformAgent.ExtraVolumeMounts, nil); err != nil {
			return err
		}
	}
	if bindplane.Spec.TSDB != nil {
		extraNames := map[string]struct{}{tsdbDataVolumeName: {}}
		if err := ValidateExtraVolumes("spec.tsdb", bindplane.Spec.TSDB.ExtraVolumes, bindplane.Spec.TSDB.ExtraVolumeMounts, extraNames); err != nil {
			return err
		}
	}
	if bindplane.Spec.Nats != nil {
		if err := ValidateExtraVolumes("spec.nats", bindplane.Spec.Nats.ExtraVolumes, bindplane.Spec.Nats.ExtraVolumeMounts, nil); err != nil {
			return err
		}
	}
	if bindplane.Spec.OpAMP != nil {
		if err := ValidateExtraVolumes("spec.opamp", bindplane.Spec.OpAMP.ExtraVolumes, bindplane.Spec.OpAMP.ExtraVolumeMounts, nil); err != nil {
			return err
		}
	}
	return nil
}

// ValidateExtraVolumes validates extra volumes and extra volume mounts for a single component.
// componentPath is used in error messages (e.g. "spec.bindplane").
// additionalReservedNames is an optional extra set of reserved volume names (used for TSDB's dynamic data volume name).
func ValidateExtraVolumes(componentPath string, volumes []corev1.Volume, mounts []corev1.VolumeMount, additionalReservedNames map[string]struct{}) error {
	volNames := make(map[string]struct{}, len(volumes))
	volPath := componentPath + ".extraVolumes"
	for i, vol := range volumes {
		fp := fmt.Sprintf("%s[%d]", volPath, i)
		// DNS-1123 label validation
		if !dns1123LabelRegexp.MatchString(vol.Name) || len(vol.Name) > 63 {
			return fmt.Errorf("%s: name %q is not a valid DNS-1123 label (lowercase alphanumeric and hyphens, must start and end with alphanumeric, max 63 chars)", fp, vol.Name)
		}
		// Uniqueness within this component
		if _, dup := volNames[vol.Name]; dup {
			return fmt.Errorf("%s: duplicate volume name %q", fp, vol.Name)
		}
		volNames[vol.Name] = struct{}{}
		// Reserved name collision
		if _, reserved := reservedVolumeNames[vol.Name]; reserved {
			return fmt.Errorf("%s: volume name %q is reserved and managed by the operator", fp, vol.Name)
		}
		if additionalReservedNames != nil {
			if _, reserved := additionalReservedNames[vol.Name]; reserved {
				return fmt.Errorf("%s: volume name %q is reserved and managed by the operator", fp, vol.Name)
			}
		}
		// Source allowlist
		if err := validateExtraVolumeSource(fp, vol.VolumeSource); err != nil {
			return err
		}
	}

	mountPaths := make(map[string]struct{}, len(mounts))
	mountPath := componentPath + ".extraVolumeMounts"
	for i, m := range mounts {
		fp := fmt.Sprintf("%s[%d]", mountPath, i)
		// mountPath must be absolute
		if !strings.HasPrefix(m.MountPath, "/") {
			return fmt.Errorf("%s: mountPath %q must be an absolute path (start with /)", fp, m.MountPath)
		}
		// Uniqueness within this component
		if _, dup := mountPaths[m.MountPath]; dup {
			return fmt.Errorf("%s: duplicate mountPath %q", fp, m.MountPath)
		}
		mountPaths[m.MountPath] = struct{}{}
		// Reserved path collision
		if _, reserved := reservedMountPaths[m.MountPath]; reserved {
			return fmt.Errorf("%s: mountPath %q is reserved and managed by the operator", fp, m.MountPath)
		}
		// Mount must reference a volume in this component's extraVolumes
		if _, ok := volNames[m.Name]; !ok {
			return fmt.Errorf("%s: name %q does not reference a volume in %s (operator-managed volumes cannot be referenced here)", fp, m.Name, volPath)
		}
	}
	return nil
}

// validateExtraVolumeSource checks that a volume uses only an allowed source type.
// hostPath is explicitly rejected; only secret, configMap, projected, csi, emptyDir,
// and downwardAPI are permitted.
func validateExtraVolumeSource(fieldPath string, vs corev1.VolumeSource) error {
	if vs.HostPath != nil {
		return fmt.Errorf("%s: hostPath volumes are not allowed; use secret, configMap, projected, csi, emptyDir, or downwardAPI", fieldPath)
	}
	hasAllowed := vs.Secret != nil || vs.ConfigMap != nil || vs.Projected != nil ||
		vs.CSI != nil || vs.EmptyDir != nil || vs.DownwardAPI != nil
	if !hasAllowed {
		return fmt.Errorf("%s: volume must use one of the allowed sources: secret, configMap, projected, csi, emptyDir, downwardAPI", fieldPath)
	}
	return nil
}
