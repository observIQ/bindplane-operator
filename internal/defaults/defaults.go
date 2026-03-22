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

// Package defaults provides shared Go-based defaulting logic for Bindplane resources.
//
// CRD/OpenAPI defaults (expressed via +kubebuilder:default markers) are the primary
// mechanism for simple scalar defaults and are applied by the Kubernetes API server.
// This package covers only defaults that cannot be expressed in the schema — currently
// spec.version — and acts as a safety net for:
//
//   - Objects created before the webhook existed.
//   - Clusters where the mutating webhook is disabled.
//
// Both the mutating webhook and the controller call ApplyDefaults so that defaulting
// logic has exactly one source of truth. The function is idempotent, deterministic,
// and has no side effects.
package defaults

import (
	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

// DefaultVersion is the Bindplane release version applied to spec.version when the
// field is empty. It mirrors the +kubebuilder:default marker on BindplaneSpec.Version
// and must be kept in sync with that marker.
const DefaultVersion = "1.98.1"

// ApplyDefaults sets Go-based default values on a Bindplane resource.
// It is safe to call multiple times; already-set fields are never overwritten.
func ApplyDefaults(bindplane *bindplanev1alpha1.Bindplane) {
	if bindplane.Spec.Version == "" {
		bindplane.Spec.Version = DefaultVersion
	}
}
