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
	"github.com/observiq/bindplane-operator/internal/validation"
)

var webhookLog = logf.Log.WithName("bindplane-webhook")

// BindplaneValidator validates Bindplane resources.
type BindplaneValidator struct{}

var _ admission.Validator[*bindplanev1alpha1.Bindplane] = &BindplaneValidator{}

// SetupBindplaneWebhookWithManager registers the validating webhook with the manager.
func SetupBindplaneWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &bindplanev1alpha1.Bindplane{}).
		WithValidator(&BindplaneValidator{}).
		Complete()
}

// ValidateCreate validates a new Bindplane resource.
func (v *BindplaneValidator) ValidateCreate(_ context.Context, bindplane *bindplanev1alpha1.Bindplane) (admission.Warnings, error) {
	webhookLog.Info("ValidateCreate", "name", bindplane.Name)
	return nil, validateBindplane(bindplane)
}

// ValidateUpdate validates an update to a Bindplane resource.
func (v *BindplaneValidator) ValidateUpdate(_ context.Context, _ *bindplanev1alpha1.Bindplane, newObj *bindplanev1alpha1.Bindplane) (admission.Warnings, error) {
	webhookLog.Info("ValidateUpdate", "name", newObj.Name)
	return nil, validateBindplane(newObj)
}

// ValidateDelete is a no-op; deletions are always allowed.
func (v *BindplaneValidator) ValidateDelete(_ context.Context, _ *bindplanev1alpha1.Bindplane) (admission.Warnings, error) {
	return nil, nil
}

// validateBindplane performs common validation for create and update operations.
func validateBindplane(bindplane *bindplanev1alpha1.Bindplane) error {
	return validation.ValidateBindplane(bindplane)
}
