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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

const (
	// prometheusComponent is the component name for Prometheus
	prometheusComponent = "prometheus"
	// prometheusContainerName is the container name for Prometheus
	prometheusContainerName = "prometheus"
	// prometheusImage is the default container image for Prometheus
	prometheusImage = "ghcr.io/observiq/bindplane-prometheus:1.96.3"
	// prometheusDataVolumeSuffix is the suffix for Prometheus data volume names
	prometheusDataVolumeSuffix = "prometheus-data"
	// prometheusHTTPPort is the HTTP port for Prometheus
	prometheusHTTPPort = 9090
	// prometheusHTTPPortName is the name of the HTTP port for Prometheus
	prometheusHTTPPortName = "http"
	// prometheusLivenessProbePath is the HTTP path for the liveness probe
	prometheusLivenessProbePath = "/-/healthy"
	// prometheusReadinessProbePath is the HTTP path for the readiness probe
	prometheusReadinessProbePath = "/-/ready"
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
	return newServiceAccount(bindplane, prometheusComponent)
}

func (r *BindplaneReconciler) prometheusStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := int32(1)
	labels := getLabels(bindplane, prometheusComponent)
	selectorLabels := getSelectorLabels(bindplane, prometheusComponent)
	serviceName := getResourceName(bindplane, prometheusComponent)

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
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: serviceName,
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    int64Ptr(defaultRunAsGroup),
							RunAsGroup: int64Ptr(defaultRunAsGroup),
							RunAsUser:  int64Ptr(defaultRunAsUser),
						},
						Affinity: getPrometheusAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  prometheusContainerName,
								Image: prometheusImage,
								Ports: []corev1.ContainerPort{
									{
										Name:          prometheusHTTPPortName,
										ContainerPort: prometheusHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: getKubernetesEnvVars(prometheusContainerName),
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
										Name:      getResourceName(bindplane, prometheusDataVolumeSuffix),
										MountPath: "/prometheus",
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: prometheusLivenessProbePath,
											Port: intstr.FromString(prometheusHTTPPortName),
										},
									},
									InitialDelaySeconds: 30,
									PeriodSeconds:       10,
									TimeoutSeconds:      5,
									FailureThreshold:    3,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: prometheusReadinessProbePath,
											Port: intstr.FromString(prometheusHTTPPortName),
										},
									},
									InitialDelaySeconds: 5,
									PeriodSeconds:       10,
									TimeoutSeconds:      5,
									FailureThreshold:    3,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						TerminationGracePeriodSeconds: int64Ptr(defaultTerminationGracePeriodSeconds),
					},
				},
				getPrometheusPodTemplateSpec(bindplane),
			),
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				getPrometheusVolumeClaimTemplate(bindplane, labels),
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
				WhenScaled:  appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	}
}

func (r *BindplaneReconciler) prometheusService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, prometheusComponent, WithPort(prometheusHTTPPortName, prometheusHTTPPort))
}

// getPrometheusAffinity returns the affinity configuration for Prometheus pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getPrometheusAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.Prometheus != nil && bindplane.Spec.Prometheus.PodTemplate != nil {
		return bindplane.Spec.Prometheus.PodTemplate.Spec.Affinity
	}
	return nil
}

// getPrometheusPodTemplateSpec returns the user-provided pod template spec for Prometheus
func getPrometheusPodTemplateSpec(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.Prometheus != nil {
		return bindplane.Spec.Prometheus.PodTemplate
	}
	return nil
}

// getPrometheusVolumeClaimTemplate returns the PersistentVolumeClaim template for Prometheus
func getPrometheusVolumeClaimTemplate(bindplane *bindplanev1alpha1.Bindplane, labels map[string]string) corev1.PersistentVolumeClaim {
	volumeName := getResourceName(bindplane, prometheusDataVolumeSuffix)

	// Default PVC spec
	defaultSpec := corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("60Gi"),
			},
		},
	}

	// Use user-provided storage configuration if available
	if bindplane.Spec.Prometheus != nil && bindplane.Spec.Prometheus.Storage != nil && bindplane.Spec.Prometheus.Storage.VolumeClaimTemplate != nil {
		userTemplate := bindplane.Spec.Prometheus.Storage.VolumeClaimTemplate

		// Start with user-provided spec
		pvcSpec := userTemplate.Spec.DeepCopy()

		// Build metadata
		pvcMeta := metav1.ObjectMeta{
			Name:   volumeName,
			Labels: labels,
		}

		// Merge user-provided metadata if present
		if userTemplate.Metadata != nil {
			if userTemplate.Metadata.Labels != nil {
				if pvcMeta.Labels == nil {
					pvcMeta.Labels = make(map[string]string)
				}
				for k, v := range userTemplate.Metadata.Labels {
					pvcMeta.Labels[k] = v
				}
			}
			if userTemplate.Metadata.Annotations != nil {
				pvcMeta.Annotations = make(map[string]string)
				for k, v := range userTemplate.Metadata.Annotations {
					pvcMeta.Annotations[k] = v
				}
			}
		}

		return corev1.PersistentVolumeClaim{
			ObjectMeta: pvcMeta,
			Spec:       *pvcSpec,
		}
	}

	// Return default PVC
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   volumeName,
			Labels: labels,
		},
		Spec: defaultSpec,
	}
}
