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
	"strconv"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

const (
	// bindplaneJobsComponent is the component name for Bindplane Jobs Migrate
	bindplaneJobsComponent = "jobs-migrate"
	// bindplaneJobsContainerName is the container name for Bindplane Jobs
	bindplaneJobsContainerName = "server"
	// bindplaneJobsImage is the default container image for Bindplane Jobs
	bindplaneJobsImage = "ghcr.io/observiq/bindplane-ee:1.96.3"
	// bindplaneJobsHTTPPort is the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPort = int32(3001)
	// bindplaneJobsHTTPPortName is the name of the HTTP port for Bindplane Jobs
	bindplaneJobsHTTPPortName = "http"
	// bindplaneJobsModeEnvVar is the environment variable name for Bindplane mode
	bindplaneJobsModeEnvVar = "BINDPLANE_MODE"
	// bindplaneJobsModeValue is the value for BINDPLANE_MODE
	bindplaneJobsModeValue = "migrate"
)

// reconcileBindplaneJobs reconciles all Bindplane Jobs resources
// Note: This deployment does NOT create a Service, as traffic should not be routed to it
func (r *BindplaneReconciler) reconcileBindplaneJobs(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.bindplaneJobsServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
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

func (r *BindplaneReconciler) bindplaneJobsDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := int32(1)
	labels := getLabels(bindplane, bindplaneJobsComponent)
	selectorLabels := getSelectorLabels(bindplane, bindplaneJobsComponent)

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
									append(
										[]corev1.EnvVar{
											{
												Name:  bindplaneJobsModeEnvVar,
												Value: bindplaneJobsModeValue,
											},
										},
										getBindplaneConfigEnvVars(&bindplane.Spec.Bindplane.Config)...,
									)...,
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
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{"sh", "-c", "sleep 5"},
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

// getBindplaneConfigEnvVars converts BindplaneConfigSpec to environment variables
// following the naming convention from override_test.go (BINDPLANE_*)
func getBindplaneConfigEnvVars(config *bindplanev1alpha1.BindplaneConfigSpec) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	// License
	if config.License != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "BINDPLANE_LICENSE",
			Value: config.License,
		})
	}

	// Auth configuration
	if config.Auth != nil {
		if config.Auth.Type != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_AUTH_TYPE",
				Value: config.Auth.Type,
			})
		}
		if config.Auth.Username != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_USERNAME",
				Value: config.Auth.Username,
			})
		}
		if config.Auth.Password != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_PASSWORD",
				Value: config.Auth.Password,
			})
		}
	}

	// Network configuration
	if config.Network != nil {
		if config.Network.Host != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_HOST",
				Value: config.Network.Host,
			})
		}
		if config.Network.Port != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_PORT",
				Value: config.Network.Port,
			})
		}
		if config.Network.RemoteURL != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_REMOTE_URL",
				Value: config.Network.RemoteURL,
			})
		}
	}

	// Store configuration
	if config.Store.Type != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "BINDPLANE_STORE_TYPE",
			Value: config.Store.Type,
		})
	}

	// Postgres configuration
	if config.Store.Postgres != nil {
		if config.Store.Postgres.Host != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_HOST",
				Value: config.Store.Postgres.Host,
			})
		}
		if config.Store.Postgres.Port != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_PORT",
				Value: config.Store.Postgres.Port,
			})
		}
		if config.Store.Postgres.ConnectTimeout != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_CONNECT_TIMEOUT",
				Value: config.Store.Postgres.ConnectTimeout,
			})
		}
		if config.Store.Postgres.StatementTimeout != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_STATEMENT_TIMEOUT",
				Value: config.Store.Postgres.StatementTimeout,
			})
		}
		if config.Store.Postgres.Database != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_DATABASE",
				Value: config.Store.Postgres.Database,
			})
		}
		if config.Store.Postgres.SSLMode != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_SSL_MODE",
				Value: config.Store.Postgres.SSLMode,
			})
		}
		if config.Store.Postgres.Username != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_USERNAME",
				Value: config.Store.Postgres.Username,
			})
		}
		if config.Store.Postgres.Password != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_PASSWORD",
				Value: config.Store.Postgres.Password,
			})
		}
		if config.Store.Postgres.MaxConnections > 0 {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_MAX_CONNECTIONS",
				Value: strconv.Itoa(config.Store.Postgres.MaxConnections),
			})
		}
		if config.Store.Postgres.MaxLifetime != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_MAX_LIFETIME",
				Value: config.Store.Postgres.MaxLifetime,
			})
		}
		if config.Store.Postgres.Schema != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_POSTGRES_SCHEMA",
				Value: config.Store.Postgres.Schema,
			})
		}
	}

	// EventBus configuration
	if config.EventBus != nil {
		if config.EventBus.Type != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "BINDPLANE_EVENT_BUS_TYPE",
				Value: config.EventBus.Type,
			})
		}
	}

	return envVars
}
