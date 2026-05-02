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
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	sessionSecretSuffix = "session-secret"
	sessionSecretKey    = "session-secret"
)

// reconcileSessionSecret creates an operator-managed session-secret Kubernetes Secret when
// the user has not provided their own via spec.config.auth.sessionSecret or
// spec.config.auth.sessionSecretSecretRef. The Secret is created once and never updated.
func (r *BindplaneReconciler) reconcileSessionSecret(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	auth := bindplane.Spec.Config.Auth
	if auth != nil && (auth.SessionSecret != "" || auth.SessionSecretSecretRef != nil) {
		return nil
	}

	secretName := getResourceName(bindplane, sessionSecretSuffix)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, sessionSecretSuffix),
		},
	}

	if err := controllerutil.SetControllerReference(bindplane, secret, r.Scheme); err != nil {
		return err
	}

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: bindplane.Namespace}, existing)
	if err == nil {
		// Secret exists; do not overwrite data
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate session secret: %w", err)
	}
	secret.Data = map[string][]byte{
		sessionSecretKey: []byte(base64.URLEncoding.EncodeToString(b)),
	}

	log.Info("Creating session secret", "name", secretName, "namespace", bindplane.Namespace)
	return r.Create(ctx, secret)
}
