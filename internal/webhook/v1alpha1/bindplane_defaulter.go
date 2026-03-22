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

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const defaultVersion = "1.98.1"

var defaulterLog = logf.Log.WithName("bindplane-defaulter")

// BindplaneDefaulter sets defaults on Bindplane resources.
type BindplaneDefaulter struct{}

var _ admission.Defaulter[*bindplanev1alpha1.Bindplane] = &BindplaneDefaulter{}

// SetupBindplaneDefaulterWithManager registers the defaulting webhook with the manager.
func SetupBindplaneDefaulterWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &bindplanev1alpha1.Bindplane{}).
		WithDefaulter(&BindplaneDefaulter{}).
		Complete()
}

// Default sets default values on the Bindplane resource.
func (d *BindplaneDefaulter) Default(_ context.Context, bindplane *bindplanev1alpha1.Bindplane) error {
	defaulterLog.Info("Default", "name", bindplane.Name)

	if bindplane.Spec.Version == "" {
		defaulterLog.Info("Defaulting spec.version", "version", defaultVersion)
		bindplane.Spec.Version = defaultVersion
	}

	return nil
}
