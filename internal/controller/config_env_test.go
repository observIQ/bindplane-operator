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
	"testing"

	corev1 "k8s.io/api/core/v1"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// requireEnvVar asserts that an env var with the given name and value is present.
func requireEnvVar(t *testing.T, envVars []corev1.EnvVar, name, wantValue string) {
	t.Helper()
	ev := findEnvVar(envVars, name)
	if ev == nil {
		t.Errorf("expected env var %q to be set, but it was not found", name)
		return
	}
	if ev.Value != wantValue {
		t.Errorf("env var %q: got %q, want %q", name, ev.Value, wantValue)
	}
}

// requireNoEnvVar asserts that an env var with the given name is NOT present.
func requireNoEnvVar(t *testing.T, envVars []corev1.EnvVar, name string) {
	t.Helper()
	ev := findEnvVar(envVars, name)
	if ev != nil {
		t.Errorf("expected env var %q to be absent, but got value %q", name, ev.Value)
	}
}

// ------- getOIDCEnvVars -------

func TestGetOIDCEnvVars_DisableInvitations(t *testing.T) {
	oidc := &bindplanev1alpha1.OIDCConfig{DisableInvitations: true}
	envVars := getOIDCEnvVars(oidc)
	requireEnvVar(t, envVars, bindplaneOIDCDisableInvitationsEnvVar, "true")
}

func TestGetOIDCEnvVars_DisableInvitationsFalse(t *testing.T) {
	oidc := &bindplanev1alpha1.OIDCConfig{DisableInvitations: false}
	envVars := getOIDCEnvVars(oidc)
	requireNoEnvVar(t, envVars, bindplaneOIDCDisableInvitationsEnvVar)
}

// ------- getErrorsConfigEnvVars -------

func TestGetErrorsConfigEnvVars_Release(t *testing.T) {
	e := &bindplanev1alpha1.ErrorsConfig{Enabled: true, Release: "2.0.0"}
	envVars := getErrorsConfigEnvVars(e)
	requireEnvVar(t, envVars, bindplaneErrorsReleaseEnvVar, "2.0.0")
}

func TestGetErrorsConfigEnvVars_TracesSampleRate(t *testing.T) {
	e := &bindplanev1alpha1.ErrorsConfig{Enabled: true, TracesSampleRate: "0.25"}
	envVars := getErrorsConfigEnvVars(e)
	requireEnvVar(t, envVars, bindplaneErrorsTracesSampleRateEnvVar, "0.25")
}

func TestGetErrorsConfigEnvVars_DebugTrue(t *testing.T) {
	e := &bindplanev1alpha1.ErrorsConfig{Enabled: true, Debug: true}
	envVars := getErrorsConfigEnvVars(e)
	requireEnvVar(t, envVars, bindplaneErrorsDebugEnvVar, "true")
}

func TestGetErrorsConfigEnvVars_DebugFalseAlwaysEmitted(t *testing.T) {
	e := &bindplanev1alpha1.ErrorsConfig{Enabled: true, Debug: false}
	envVars := getErrorsConfigEnvVars(e)
	requireEnvVar(t, envVars, bindplaneErrorsDebugEnvVar, "false")
}

func TestGetErrorsConfigEnvVars_NilReturnsNil(t *testing.T) {
	envVars := getErrorsConfigEnvVars(nil)
	if envVars != nil {
		t.Errorf("expected nil for nil errors config, got %v", envVars)
	}
}

// ------- getLLMConfigEnvVars (Gemini additions) -------

func TestGetLLMConfigEnvVars_GeminiCredentialsFile(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Gemini: &bindplanev1alpha1.GeminiConfig{CredentialsFile: "/path/to/creds.json"},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMGeminiCredentialsFileEnvVar, "/path/to/creds.json")
}

func TestGetLLMConfigEnvVars_GeminiMaxTokens(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Gemini: &bindplanev1alpha1.GeminiConfig{MaxTokens: 1000},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMGeminiMaxTokensEnvVar, "1000")
}

func TestGetLLMConfigEnvVars_GeminiMaxTokensZeroOmitted(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Gemini: &bindplanev1alpha1.GeminiConfig{MaxTokens: 0},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireNoEnvVar(t, envVars, bindplaneLLMGeminiMaxTokensEnvVar)
}

// ------- getLLMConfigEnvVars (Langsmith additions) -------

func TestGetLLMConfigEnvVars_LangsmithURL(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{URL: "https://api.smith.langchain.com/api/v1"},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMLangsmithURLEnvVar, "https://api.smith.langchain.com/api/v1")
}

func TestGetLLMConfigEnvVars_LangsmithSanitizeContentFalse(t *testing.T) {
	f := false
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{SanitizeContent: &f},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMLangsmithSanitizeContentEnvVar, "false")
}

func TestGetLLMConfigEnvVars_LangsmithSanitizeContentTrue(t *testing.T) {
	tr := true
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{SanitizeContent: &tr},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMLangsmithSanitizeContentEnvVar, "true")
}

func TestGetLLMConfigEnvVars_LangsmithSanitizeContentNilOmitted(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{SanitizeContent: nil},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireNoEnvVar(t, envVars, bindplaneLLMLangsmithSanitizeContentEnvVar)
}

func TestGetLLMConfigEnvVars_LangsmithTags(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{Tags: []string{"env", "test"}},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireEnvVar(t, envVars, bindplaneLLMLangsmithTagsEnvVar, "env,test")
}

func TestGetLLMConfigEnvVars_LangsmithTagsEmptyOmitted(t *testing.T) {
	llm := &bindplanev1alpha1.LLMConfig{
		Langsmith: &bindplanev1alpha1.LangsmithConfig{Tags: nil},
	}
	envVars := getLLMConfigEnvVars(llm)
	requireNoEnvVar(t, envVars, bindplaneLLMLangsmithTagsEnvVar)
}

// ------- getPostHogEnvVars -------

func TestGetPostHogEnvVars_FeatureFlagRequestTimeout(t *testing.T) {
	ph := &bindplanev1alpha1.PostHogConfig{FeatureFlagRequestTimeout: "10s"}
	envVars := getPostHogEnvVars(ph)
	requireEnvVar(t, envVars, bindplaneFeaturesPostHogFeatureFlagRequestTimeoutEnvVar, "10s")
}

func TestGetPostHogEnvVars_FeatureFlagRequestTimeoutEmptyOmitted(t *testing.T) {
	ph := &bindplanev1alpha1.PostHogConfig{}
	envVars := getPostHogEnvVars(ph)
	requireNoEnvVar(t, envVars, bindplaneFeaturesPostHogFeatureFlagRequestTimeoutEnvVar)
}

// ------- getFeatureOverridesEnvVars -------

func TestGetFeatureOverridesEnvVars_NewBoolFields(t *testing.T) {
	o := &bindplanev1alpha1.FeatureOverridesConfig{
		SecopsGcsIntegration:                   true,
		SnapshotPipelineIntelligence:           true,
		PipelineIntelligenceSplunkConfigImport: true,
		RawLogMetricViews:                      true,
		Vault:                                  true,
		Auth0SSO:                               true,
		AixPlatform:                            true,
		AdvancedPipelineEditor:                 true,
		IdentityTablesDualWrite:                true,
		IdentityTablesCutover:                  true,
		V2Configuration:                        true,
		V2Connectors:                           true,
		BindplaneBlueprints:                    true,
		Fleets:                                 true,
	}
	envVars := getFeatureOverridesEnvVars(o)
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesSecopsGCSIntegrationEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesSnapshotPipelineIntelligenceEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesPipelineIntelligenceSplunkConfigImportEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesRawLogMetricViewsEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesVaultEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesAuth0SSOEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesAixPlatformEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesAdvancedPipelineEditorEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesIdentityTablesDualWriteEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesIdentityTablesCutoverEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesV2ConfigurationEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesV2ConnectorsEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesBindplaneBlueprintsEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneFeaturesOverridesFleetsEnvVar, "true")
}

func TestGetFeatureOverridesEnvVars_FalseFieldsOmitted(t *testing.T) {
	o := &bindplanev1alpha1.FeatureOverridesConfig{}
	envVars := getFeatureOverridesEnvVars(o)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesSecopsGCSIntegrationEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesSnapshotPipelineIntelligenceEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesPipelineIntelligenceSplunkConfigImportEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesRawLogMetricViewsEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesVaultEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesAuth0SSOEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesAixPlatformEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesAdvancedPipelineEditorEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesIdentityTablesDualWriteEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesIdentityTablesCutoverEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesV2ConfigurationEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesV2ConnectorsEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesBindplaneBlueprintsEnvVar)
	requireNoEnvVar(t, envVars, bindplaneFeaturesOverridesFleetsEnvVar)
}

// ------- getQuotasConfigEnvVars -------

func TestGetQuotasConfigEnvVars_Organizations(t *testing.T) {
	q := &bindplanev1alpha1.QuotasConfig{
		Organizations: &bindplanev1alpha1.QuotasTierConfig{
			Enabled:  true,
			Enforced: true,
			Default:  &bindplanev1alpha1.QuotasTierDefaultConfig{MaxAgents: 3000},
		},
	}
	envVars := getQuotasConfigEnvVars(q)
	requireEnvVar(t, envVars, bindplaneQuotasOrganizationsEnabledEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneQuotasOrganizationsEnforcedEnvVar, "true")
	requireEnvVar(t, envVars, bindplaneQuotasOrganizationsDefaultMaxAgentsEnvVar, "3000")
}

func TestGetQuotasConfigEnvVars_ProjectsDefaultMaxAgents(t *testing.T) {
	q := &bindplanev1alpha1.QuotasConfig{
		Projects: &bindplanev1alpha1.QuotasTierConfig{
			Default: &bindplanev1alpha1.QuotasTierDefaultConfig{MaxAgents: 2000},
		},
	}
	envVars := getQuotasConfigEnvVars(q)
	requireEnvVar(t, envVars, bindplaneQuotasProjectsDefaultMaxAgentsEnvVar, "2000")
}

func TestGetQuotasConfigEnvVars_OrganizationsNilOmitted(t *testing.T) {
	q := &bindplanev1alpha1.QuotasConfig{}
	envVars := getQuotasConfigEnvVars(q)
	requireNoEnvVar(t, envVars, bindplaneQuotasOrganizationsEnabledEnvVar)
	requireNoEnvVar(t, envVars, bindplaneQuotasOrganizationsEnforcedEnvVar)
	requireNoEnvVar(t, envVars, bindplaneQuotasOrganizationsDefaultMaxAgentsEnvVar)
	requireNoEnvVar(t, envVars, bindplaneQuotasProjectsDefaultMaxAgentsEnvVar)
}

// ------- getEncryptionProviderEnvVars -------

func TestGetEncryptionProviderEnvVars_Cache(t *testing.T) {
	ep := &bindplanev1alpha1.EncryptionProviderConfig{
		Cache: &bindplanev1alpha1.EncryptionProviderCacheConfig{
			Capacity:     7000,
			CacheTimeout: "7m",
		},
	}
	envVars := getEncryptionProviderEnvVars(ep)
	requireEnvVar(t, envVars, bindplaneEncryptionProviderCacheCapacityEnvVar, "7000")
	requireEnvVar(t, envVars, bindplaneEncryptionProviderCacheCacheTimeoutEnvVar, "7m")
}

func TestGetEncryptionProviderEnvVars_CacheNilOmitted(t *testing.T) {
	ep := &bindplanev1alpha1.EncryptionProviderConfig{}
	envVars := getEncryptionProviderEnvVars(ep)
	requireNoEnvVar(t, envVars, bindplaneEncryptionProviderCacheCapacityEnvVar)
	requireNoEnvVar(t, envVars, bindplaneEncryptionProviderCacheCacheTimeoutEnvVar)
}

// ------- getSaaSStripeEnvVars -------

func TestGetSaaSStripeEnvVars_MeterReportInterval(t *testing.T) {
	stripe := &bindplanev1alpha1.SaaSStripeConfig{MeterReportInterval: "2h5m"}
	envVars := getSaaSStripeEnvVars(stripe)
	requireEnvVar(t, envVars, bindplaneSaaSStripeMeterReportIntervalEnvVar, "2h5m")
}

func TestGetSaaSStripeEnvVars_MeterReportIntervalEmptyOmitted(t *testing.T) {
	stripe := &bindplanev1alpha1.SaaSStripeConfig{}
	envVars := getSaaSStripeEnvVars(stripe)
	requireNoEnvVar(t, envVars, bindplaneSaaSStripeMeterReportIntervalEnvVar)
}

// ------- getAdvancedConfigEnvVars -------

func TestGetAdvancedConfigEnvVars_AgentTelemetryPort(t *testing.T) {
	port := int32(8000)
	config := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Agent: &bindplanev1alpha1.AdvancedAgentConfig{TelemetryPort: &port},
		},
	}
	envVars := getAdvancedConfigEnvVars(config)
	requireEnvVar(t, envVars, bindplaneAdvancedAgentTelemetryPortEnvVar, "8000")
}

func TestGetAdvancedConfigEnvVars_AgentTelemetryPortNilOmitted(t *testing.T) {
	config := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Agent: &bindplanev1alpha1.AdvancedAgentConfig{},
		},
	}
	envVars := getAdvancedConfigEnvVars(config)
	requireNoEnvVar(t, envVars, bindplaneAdvancedAgentTelemetryPortEnvVar)
}

func TestGetAdvancedConfigEnvVars_RolloutRetryInterval(t *testing.T) {
	config := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Rollout: &bindplanev1alpha1.AdvancedRolloutConfig{RetryInterval: "1m"},
		},
	}
	envVars := getAdvancedConfigEnvVars(config)
	requireEnvVar(t, envVars, bindplaneAdvancedRolloutRetryIntervalEnvVar, "1m")
}

func TestGetAdvancedConfigEnvVars_RolloutUpdateWorkerCount(t *testing.T) {
	config := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Rollout: &bindplanev1alpha1.AdvancedRolloutConfig{UpdateWorkerCount: 15},
		},
	}
	envVars := getAdvancedConfigEnvVars(config)
	requireEnvVar(t, envVars, bindplaneAdvancedRolloutUpdateWorkerCountEnvVar, "15")
}

func TestGetAdvancedConfigEnvVars_RolloutZeroWorkerCountOmitted(t *testing.T) {
	config := &bindplanev1alpha1.BindplaneConfigSpec{
		Advanced: &bindplanev1alpha1.AdvancedConfig{
			Rollout: &bindplanev1alpha1.AdvancedRolloutConfig{UpdateWorkerCount: 0},
		},
	}
	envVars := getAdvancedConfigEnvVars(config)
	requireNoEnvVar(t, envVars, bindplaneAdvancedRolloutUpdateWorkerCountEnvVar)
}

// ------- getNatsTLSEnvVars (SkipVerify) -------

func TestGetNatsTLSEnvVars_SkipVerifyEmittedWhenSet(t *testing.T) {
	bindplane := &bindplanev1alpha1.Bindplane{
		Spec: bindplanev1alpha1.BindplaneSpec{
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				Nats: &bindplanev1alpha1.NatsConfig{
					TLS: &bindplanev1alpha1.NatsTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{
							Name: "my-issuer",
							Kind: "ClusterIssuer",
						},
						SkipVerify: true,
					},
				},
			},
		},
	}
	envVars := getNatsTLSEnvVars(bindplane)
	requireEnvVar(t, envVars, bindplaneNatsTLSSkipVerifyEnvVar, "true")
}

func TestGetNatsTLSEnvVars_SkipVerifyFalseOmitted(t *testing.T) {
	bindplane := &bindplanev1alpha1.Bindplane{
		Spec: bindplanev1alpha1.BindplaneSpec{
			Config: bindplanev1alpha1.BindplaneConfigSpec{
				Nats: &bindplanev1alpha1.NatsConfig{
					TLS: &bindplanev1alpha1.NatsTLSConfig{
						CertManager: &bindplanev1alpha1.CertManagerTLSIssuerRef{
							Name: "my-issuer",
							Kind: "ClusterIssuer",
						},
						SkipVerify: false,
					},
				},
			},
		},
	}
	envVars := getNatsTLSEnvVars(bindplane)
	requireNoEnvVar(t, envVars, bindplaneNatsTLSSkipVerifyEnvVar)
}

func TestGetNatsTLSEnvVars_NoCertManagerReturnsNil(t *testing.T) {
	bindplane := &bindplanev1alpha1.Bindplane{
		Spec: bindplanev1alpha1.BindplaneSpec{
			Config: bindplanev1alpha1.BindplaneConfigSpec{},
		},
	}
	envVars := getNatsTLSEnvVars(bindplane)
	if envVars != nil {
		t.Errorf("expected nil when cert-manager TLS is not enabled, got %v", envVars)
	}
}

// TestGetErrorsConfigEnvVars_PlainValue verifies that when BackendDSN / FrontendDSN
// are set as plain strings, the env var is emitted as a plain Value.
func TestGetErrorsConfigEnvVars_PlainValue(t *testing.T) {
	cfg := &bindplanev1alpha1.ErrorsConfig{
		Enabled:     true,
		BackendDSN:  "https://key@sentry.io/123",
		FrontendDSN: "https://key@sentry.io/456",
		Environment: "production",
	}
	envVars := getErrorsConfigEnvVars(cfg)

	found := map[string]string{}
	for _, ev := range envVars {
		if ev.ValueFrom == nil {
			found[ev.Name] = ev.Value
		}
	}

	if v, ok := found[bindplaneErrorsBackendDSNEnvVar]; !ok || v != "https://key@sentry.io/123" {
		t.Errorf("expected %s = %q, got %q", bindplaneErrorsBackendDSNEnvVar, "https://key@sentry.io/123", v)
	}
	if v, ok := found[bindplaneErrorsFrontendDSNEnvVar]; !ok || v != "https://key@sentry.io/456" {
		t.Errorf("expected %s = %q, got %q", bindplaneErrorsFrontendDSNEnvVar, "https://key@sentry.io/456", v)
	}
}

// TestGetErrorsConfigEnvVars_SecretRef verifies that when BackendDSNSecretRef /
// FrontendDSNSecretRef are set, the env var is emitted with valueFrom.secretKeyRef.
func TestGetErrorsConfigEnvVars_SecretRef(t *testing.T) {
	cfg := &bindplanev1alpha1.ErrorsConfig{
		Enabled: true,
		BackendDSNSecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
			Key:                  "backend-dsn",
		},
		FrontendDSNSecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
			Key:                  "frontend-dsn",
		},
	}
	envVars := getErrorsConfigEnvVars(cfg)

	found := map[string]*corev1.SecretKeySelector{}
	for _, ev := range envVars {
		if ev.ValueFrom != nil && ev.ValueFrom.SecretKeyRef != nil {
			found[ev.Name] = ev.ValueFrom.SecretKeyRef
		}
	}

	if ref, ok := found[bindplaneErrorsBackendDSNEnvVar]; !ok || ref.Name != "my-secret" || ref.Key != "backend-dsn" {
		t.Errorf("expected %s to reference secret my-secret/backend-dsn, got %v", bindplaneErrorsBackendDSNEnvVar, found[bindplaneErrorsBackendDSNEnvVar])
	}
	if ref, ok := found[bindplaneErrorsFrontendDSNEnvVar]; !ok || ref.Name != "my-secret" || ref.Key != "frontend-dsn" {
		t.Errorf("expected %s to reference secret my-secret/frontend-dsn, got %v", bindplaneErrorsFrontendDSNEnvVar, found[bindplaneErrorsFrontendDSNEnvVar])
	}
}

// TestGetErrorsConfigEnvVars_SecretRefPrecedence verifies that when both a plain value
// and a SecretRef are provided, the SecretRef takes precedence.
func TestGetErrorsConfigEnvVars_SecretRefPrecedence(t *testing.T) {
	cfg := &bindplanev1alpha1.ErrorsConfig{
		BackendDSN: "https://plain@sentry.io/123",
		BackendDSNSecretRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "ref-secret"},
			Key:                  "backend-dsn",
		},
	}
	envVars := getErrorsConfigEnvVars(cfg)

	for _, ev := range envVars {
		if ev.Name == bindplaneErrorsBackendDSNEnvVar {
			if ev.ValueFrom == nil || ev.ValueFrom.SecretKeyRef == nil {
				t.Errorf("expected SecretRef to take precedence, but got plain value %q", ev.Value)
			}
			if ev.ValueFrom.SecretKeyRef.Name != "ref-secret" {
				t.Errorf("expected SecretKeyRef name ref-secret, got %s", ev.ValueFrom.SecretKeyRef.Name)
			}
			return
		}
	}
	t.Errorf("env var %s not found", bindplaneErrorsBackendDSNEnvVar)
}
