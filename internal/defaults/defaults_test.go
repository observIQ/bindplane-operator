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

package defaults_test

import (
	"testing"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
	"github.com/observiq/bindplane-operator/internal/defaults"
)

func TestApplyDefaults_SetsVersionWhenEmpty(t *testing.T) {
	bp := &bindplanev1alpha1.Bindplane{}
	defaults.ApplyDefaults(bp)
	if bp.Spec.Version != defaults.DefaultVersion {
		t.Errorf("expected spec.version %q, got %q", defaults.DefaultVersion, bp.Spec.Version)
	}
}

func TestApplyDefaults_PreservesExistingVersion(t *testing.T) {
	const customVersion = "2.0.0"
	bp := &bindplanev1alpha1.Bindplane{}
	bp.Spec.Version = customVersion
	defaults.ApplyDefaults(bp)
	if bp.Spec.Version != customVersion {
		t.Errorf("expected spec.version %q to be preserved, got %q", customVersion, bp.Spec.Version)
	}
}

func TestApplyDefaults_Idempotent(t *testing.T) {
	bp := &bindplanev1alpha1.Bindplane{}
	defaults.ApplyDefaults(bp)
	first := bp.Spec.Version
	defaults.ApplyDefaults(bp)
	if bp.Spec.Version != first {
		t.Errorf("ApplyDefaults is not idempotent: first=%q second=%q", first, bp.Spec.Version)
	}
}
