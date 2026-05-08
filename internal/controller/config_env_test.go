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
