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
	"sigs.k8s.io/yaml"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

const (
	// bindplaneJobsComponent is the component name for Bindplane Jobs Migrate
	bindplaneJobsComponent = "bindplane-jobs-migrate"
	// bindplaneJobsContainerName is the container name for Bindplane Jobs
	bindplaneJobsContainerName = "server"
	// bindplaneJobsImage is the default container image for Bindplane Jobs
	bindplaneJobsImage = "ghcr.io/observiq/bindplane-ee:1.96.3"
	// bindplaneJobsHTTPPort is the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPort = int32(3001)
	// bindplaneJobsHTTPPortName is the name of the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPortName = "http"
	// bindplaneJobsConfigPath is the path where the config file is mounted
	bindplaneJobsConfigPath = "/config.yaml"
	// bindplaneJobsConfigVolumeName is the name of the volume for the config file
	bindplaneJobsConfigVolumeName = "config"
	// bindplaneJobsConfigMapKey is the key in the ConfigMap for the config file
	bindplaneJobsConfigMapKey = "config.yaml"
	// bindplaneJobsModeEnvVar is the environment variable name for Bindplane mode
	bindplaneJobsModeEnvVar = "BINDPLANE_MODE"
	// bindplaneJobsModeValue is the value for BINDPLANE_MODE
	bindplaneJobsModeValue = "migrate"
	// bindplaneJobsConfigEnvVar is the environment variable name for config file path
	bindplaneJobsConfigEnvVar = "BINDPLANE_CONFIG"
)

// reconcileBindplaneJobs reconciles all Bindplane Jobs resources
// Note: This deployment does NOT create a Service, as traffic should not be routed to it
func (r *BindplaneReconciler) reconcileBindplaneJobs(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.bindplaneJobsServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile ConfigMap
	configMap := r.bindplaneJobsConfigMap(bindplane)
	if err := r.reconcileConfigMap(ctx, bindplane, configMap, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.bindplaneJobsDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	return nil
}

func (r *BindplaneReconciler) bindplaneJobsServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, bindplaneJobsComponent)
}

func (r *BindplaneReconciler) bindplaneJobsConfigMap(bindplane *bindplanev1alpha1.Bindplane) *corev1.ConfigMap {
	// Convert BindplaneConfigSpec to bindplaneConfig
	config := toBindplaneConfig(&bindplane.Spec.Bindplane.Config)

	// Marshal to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		// This should not happen in practice, but if it does, we'll create an empty ConfigMap
		// The error will be logged by the caller
		yamlData = []byte{}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, bindplaneJobsComponent),
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, bindplaneJobsComponent),
		},
		Data: map[string]string{
			bindplaneJobsConfigMapKey: string(yamlData),
		},
	}
}

func (r *BindplaneReconciler) bindplaneJobsDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := int32(1)
	labels := getLabels(bindplane, bindplaneJobsComponent)
	selectorLabels := getSelectorLabels(bindplane, bindplaneJobsComponent)
	configMapName := getResourceName(bindplane, bindplaneJobsComponent)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, bindplaneJobsComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: getResourceName(bindplane, bindplaneJobsComponent),
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    int64Ptr(65534),
							RunAsGroup: int64Ptr(65534),
							RunAsUser:  int64Ptr(65534),
						},
						Affinity: getBindplaneJobsAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  bindplaneJobsContainerName,
								Image: bindplaneJobsImage,
								Ports: []corev1.ContainerPort{
									{
										Name:          bindplaneJobsHTTPPortName,
										ContainerPort: bindplaneJobsHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: append(
									getKubernetesEnvVars(bindplaneJobsContainerName),
									[]corev1.EnvVar{
										{
											Name:  bindplaneJobsModeEnvVar,
											Value: bindplaneJobsModeValue,
										},
										{
											Name:  bindplaneJobsConfigEnvVar,
											Value: bindplaneJobsConfigPath,
										},
									}...,
								),
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("1000Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("1000m"),
										corev1.ResourceMemory: resource.MustParse("1000Mi"),
									},
								},
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
									FailureThreshold:    20,
									InitialDelaySeconds: 0,
									PeriodSeconds:       5,
									SuccessThreshold:    1,
									TimeoutSeconds:      1,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromString(bindplaneJobsHTTPPortName),
										},
									},
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(65534)),
								ImagePullPolicy: corev1.PullIfNotPresent,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      bindplaneJobsConfigVolumeName,
										MountPath: "/",
										ReadOnly:  true,
									},
								},
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{"sh", "-c", "sleep 5"},
										},
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: bindplaneJobsConfigVolumeName,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: configMapName,
										},
										Items: []corev1.KeyToPath{
											{
												Key:  bindplaneJobsConfigMapKey,
												Path: "config.yaml",
											},
										},
									},
								},
							},
						},
						TerminationGracePeriodSeconds: int64Ptr(60),
					},
				},
				getBindplaneJobsPodTemplate(bindplane),
			),
		},
	}
}

// getBindplaneJobsAffinity returns the affinity configuration for Bindplane Jobs pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getBindplaneJobsAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.Bindplane.PodTemplate != nil {
		return bindplane.Spec.Bindplane.PodTemplate.Spec.Affinity
	}
	return nil
}

// getBindplaneJobsPodTemplate returns the user-provided pod template spec for Bindplane Jobs
func getBindplaneJobsPodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	return bindplane.Spec.Bindplane.PodTemplate
}
