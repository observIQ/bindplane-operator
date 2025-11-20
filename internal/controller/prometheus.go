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

// reconcilePrometheus reconciles all Prometheus resources
func (r *BindplaneReconciler) reconcilePrometheus(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.prometheusServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile StatefulSet
	statefulSet := r.prometheusStatefulSet(bindplane)
	if err := r.reconcileStatefulSet(ctx, bindplane, statefulSet, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.prometheusService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	return nil
}

func (r *BindplaneReconciler) prometheusServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prometheus", bindplane.Name),
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, "prometheus"),
		},
	}
}

func (r *BindplaneReconciler) prometheusStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := int32(1)
	labels := getLabels(bindplane, "prometheus")
	selectorLabels := getSelectorLabels(bindplane, "prometheus")
	serviceName := fmt.Sprintf("%s-prometheus", bindplane.Name)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         serviceName,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selectorLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceName,
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: int64Ptr(65534),
					},
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: "ghcr.io/observiq/bindplane-prometheus:1.96.3",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("500Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("500Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf("%s-prometheus-data", bindplane.Name),
									MountPath: "/prometheus",
								},
								{
									Name:      "config",
									MountPath: "/etc/prometheus/prometheus.yml",
									SubPath:   "prometheus.yml",
								},
								{
									Name:      "config",
									MountPath: "/etc/prometheus/rules.yml",
									SubPath:   "rules.yml",
								},
								{
									Name:      "config",
									MountPath: "/etc/prometheus/web.yml",
									SubPath:   "web.yml",
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								RunAsNonRoot:             boolPtr(true),
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsUser:                int64Ptr(65534),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: serviceName,
									},
								},
							},
						},
					},
					TerminationGracePeriodSeconds: int64Ptr(60),
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   fmt.Sprintf("%s-prometheus-data", bindplane.Name),
						Labels: labels,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	}
}

func (r *BindplaneReconciler) prometheusService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	labels := getLabels(bindplane, "prometheus")
	selectorLabels := getSelectorLabels(bindplane, "prometheus")

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-prometheus", bindplane.Name),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
