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
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

// reconcileTransformAgent reconciles all Transform Agent resources
func (r *BindplaneReconciler) reconcileTransformAgent(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.transformAgentServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.transformAgentDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.transformAgentService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	return nil
}

func (r *BindplaneReconciler) transformAgentServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-transform-agent", bindplane.Name),
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, "transform-agent"),
		},
	}
}

func (r *BindplaneReconciler) transformAgentDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := int32(2)
	labels := getLabels(bindplane, "transform-agent")
	selectorLabels := getSelectorLabels(bindplane, "transform-agent")

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-transform-agent", bindplane.Name),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-transform-agent", bindplane.Name),
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:    int64Ptr(65534),
						RunAsGroup: int64Ptr(65534),
						RunAsUser:  int64Ptr(65534),
					},
					Containers: []corev1.Container{
						{
							Name:  "transform-agent",
							Image: "ghcr.io/observiq/bindplane-transform-agent:1.96.3-bindplane",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 4568,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/collector-version",
										Port: intstr.FromString("http"),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/collector-version",
										Port: intstr.FromString("http"),
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/collector-version",
										Port: intstr.FromString("http"),
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								ReadOnlyRootFilesystem: boolPtr(true),
								RunAsNonRoot:           boolPtr(true),
								RunAsUser:              int64Ptr(65534),
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					TerminationGracePeriodSeconds: int64Ptr(60),
				},
			},
		},
	}
}

func (r *BindplaneReconciler) transformAgentService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	labels := getLabels(bindplane, "transform-agent")
	selectorLabels := getSelectorLabels(bindplane, "transform-agent")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-transform-agent", bindplane.Name),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       4568,
					TargetPort: intstr.FromInt(4568),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
