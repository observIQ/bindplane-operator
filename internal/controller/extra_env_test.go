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
)

func TestPrependExtraEnv_NilExtraEnv(t *testing.T) {
	op := []corev1.EnvVar{{Name: "OP_VAR", Value: "op"}}
	result := prependExtraEnv(nil, op)
	if len(result) != 1 || result[0].Name != "OP_VAR" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestPrependExtraEnv_EmptyExtraEnv(t *testing.T) {
	op := []corev1.EnvVar{{Name: "OP_VAR", Value: "op"}}
	result := prependExtraEnv([]corev1.EnvVar{}, op)
	if len(result) != 1 || result[0].Name != "OP_VAR" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestPrependExtraEnv_UserVarsFirst(t *testing.T) {
	user := []corev1.EnvVar{
		{Name: "HTTP_PROXY", Value: "http://proxy:3128"},
		{Name: "NO_PROXY", Value: "localhost"},
	}
	op1 := []corev1.EnvVar{{Name: "KUBERNETES_NAMESPACE_NAME", Value: "ns"}}
	op2 := []corev1.EnvVar{{Name: "BINDPLANE_MODE", Value: "node"}}

	result := prependExtraEnv(user, op1, op2)

	if len(result) != 4 {
		t.Fatalf("expected 4 vars, got %d", len(result))
	}
	if result[0].Name != "HTTP_PROXY" {
		t.Errorf("expected HTTP_PROXY first, got %s", result[0].Name)
	}
	if result[1].Name != "NO_PROXY" {
		t.Errorf("expected NO_PROXY second, got %s", result[1].Name)
	}
	if result[2].Name != "KUBERNETES_NAMESPACE_NAME" {
		t.Errorf("expected KUBERNETES_NAMESPACE_NAME third, got %s", result[2].Name)
	}
	if result[3].Name != "BINDPLANE_MODE" {
		t.Errorf("expected BINDPLANE_MODE last, got %s", result[3].Name)
	}
}

func TestPrependExtraEnv_OperatorWinsOnDuplicate(t *testing.T) {
	// If the user sets a name that the operator also sets, the operator's value
	// (appended last) takes precedence — Kubernetes uses the last entry.
	user := []corev1.EnvVar{{Name: "SOME_VAR", Value: "user-value"}}
	op := []corev1.EnvVar{{Name: "SOME_VAR", Value: "operator-value"}}

	result := prependExtraEnv(user, op)

	if len(result) != 2 {
		t.Fatalf("expected 2 vars (both kept), got %d", len(result))
	}
	// Operator value is last — Kubernetes uses this one.
	if result[1].Value != "operator-value" {
		t.Errorf("expected operator-value last, got %s", result[1].Value)
	}
}

func TestPrependExtraEnv_NoOperatorSlices(t *testing.T) {
	user := []corev1.EnvVar{{Name: "MY_VAR", Value: "x"}}
	result := prependExtraEnv(user)
	if len(result) != 1 || result[0].Name != "MY_VAR" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestPrependExtraEnv_AllEmpty(t *testing.T) {
	result := prependExtraEnv(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}
