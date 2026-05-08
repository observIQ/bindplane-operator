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

	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// nodeHPADefaultMinReplicas is the minimum replica count for the Node HPA.
	nodeHPADefaultMinReplicas = int32(2)
	// nodeHPADefaultMaxReplicas is the maximum replica count for the Node HPA.
	nodeHPADefaultMaxReplicas = int32(10)
)

// reconcileNodeHPA creates, updates, or deletes the HorizontalPodAutoscaler for Bindplane Node.
// When autoscaling is disabled (or the Autoscaling field is nil), any existing HPA is deleted
// so that the static replica count from the Deployment takes effect.
func (r *BindplaneReconciler) reconcileNodeHPA(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	if bindplane.Spec.Bindplane.Autoscaling == nil || !bindplane.Spec.Bindplane.Autoscaling.Enabled {
		return r.deleteHPAIfExists(ctx, bindplane, nodeComponent, log)
	}

	hpa := r.nodeHPA(bindplane)
	return r.reconcileHorizontalPodAutoscaler(ctx, bindplane, hpa, log)
}

// nodeHPA builds the HorizontalPodAutoscaler for Bindplane Node, merging user-provided
// overrides with the default configuration. Fields not set by the user fall back to defaults
// that are tuned for Node's stateful WebSocket (OpAMP) workload.
func (r *BindplaneReconciler) nodeHPA(bindplane *bindplanev1alpha1.Bindplane) *autoscalingv2.HorizontalPodAutoscaler {
	cfg := bindplane.Spec.Bindplane.Autoscaling
	labels := getLabels(bindplane, nodeComponent)

	minReplicas := nodeHPADefaultMinReplicas
	if cfg.MinReplicas != nil {
		minReplicas = *cfg.MinReplicas
	}

	maxReplicas := nodeHPADefaultMaxReplicas
	if cfg.MaxReplicas != nil {
		maxReplicas = *cfg.MaxReplicas
	}

	metrics := cfg.Metrics
	if len(metrics) == 0 {
		metrics = defaultNodeHPAMetrics()
	}

	behavior := cfg.Behavior
	if behavior == nil {
		behavior = defaultNodeHPABehavior()
	}

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, nodeComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: nodeHPAScaleTargetRef(bindplane),
			MinReplicas:    &minReplicas,
			MaxReplicas:    maxReplicas,
			Metrics:        metrics,
			Behavior:       behavior,
		},
	}
}

// nodeHPAScaleTargetRef returns the HPA scale target reference for Bindplane Node.
// When ArgoRollout is enabled, it targets the Rollout resource; otherwise a Deployment.
func nodeHPAScaleTargetRef(bindplane *bindplanev1alpha1.Bindplane) autoscalingv2.CrossVersionObjectReference {
	if bindplane.Spec.Bindplane.ArgoRollout != nil && bindplane.Spec.Bindplane.ArgoRollout.Enabled {
		return autoscalingv2.CrossVersionObjectReference{
			APIVersion: argoRolloutAPIVersion,
			Kind:       argoRolloutKind,
			Name:       getResourceName(bindplane, nodeComponent),
		}
	}
	return autoscalingv2.CrossVersionObjectReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       getResourceName(bindplane, nodeComponent),
	}
}

// defaultNodeHPAMetrics returns the default HPA metrics for Bindplane Node:
// CPU at 50% target utilization.
func defaultNodeHPAMetrics() []autoscalingv2.MetricSpec {
	cpuTarget := int32(50)
	return []autoscalingv2.MetricSpec{
		{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: &cpuTarget,
				},
			},
		},
	}
}

// defaultNodeHPABehavior returns the default HPA behavior for Bindplane Node:
// slow scale-down (1 pod per 5 minutes) to prevent agent reconnection storms.
func defaultNodeHPABehavior() *autoscalingv2.HorizontalPodAutoscalerBehavior {
	stabilizationWindow := int32(300)
	selectPolicyMin := autoscalingv2.MinChangePolicySelect
	return &autoscalingv2.HorizontalPodAutoscalerBehavior{
		ScaleDown: &autoscalingv2.HPAScalingRules{
			StabilizationWindowSeconds: &stabilizationWindow,
			SelectPolicy:               &selectPolicyMin,
			Policies: []autoscalingv2.HPAScalingPolicy{
				{
					Type:          autoscalingv2.PodsScalingPolicy,
					Value:         1,
					PeriodSeconds: 300,
				},
			},
		},
	}
}

// reconcileHorizontalPodAutoscaler creates or updates an HPA resource.
// It follows the same create/update pattern used by other reconcile helpers.
func (r *BindplaneReconciler) reconcileHorizontalPodAutoscaler(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, hpa *autoscalingv2.HorizontalPodAutoscaler, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, hpa, r.Scheme); err != nil {
		return err
	}

	found := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: hpa.Name, Namespace: hpa.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating HorizontalPodAutoscaler", "name", hpa.Name, "namespace", hpa.Namespace)
		return r.Create(ctx, hpa)
	} else if err != nil {
		return err
	}

	found.Spec = hpa.Spec
	found.Labels = hpa.Labels
	if err := r.Update(ctx, found); err != nil {
		return err
	}
	return nil
}

// deleteHPAIfExists deletes the HPA for a component if one exists.
// Called when autoscaling is disabled to clean up a previously-created HPA.
func (r *BindplaneReconciler) deleteHPAIfExists(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string, log logr.Logger) error {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}, hpa)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	log.Info("Deleting HorizontalPodAutoscaler", "name", hpa.Name)
	return r.Delete(ctx, hpa)
}
