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

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// transformAgentComponent is the component name for Transform Agent
	transformAgentComponent = "transform-agent"
	// transformAgentContainerName is the container name for Transform Agent
	transformAgentContainerName = "transform-agent"
	// transformAgentHTTPPort is the HTTP port for Transform Agent
	transformAgentHTTPPort = 4568
	// transformAgentHTTPPortName is the name of the HTTP port for Transform Agent
	transformAgentHTTPPortName = "http"
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

	// Reconcile PodDisruptionBudget
	if bindplane.Spec.TransformAgent == nil || !bindplane.Spec.TransformAgent.DisablePodDisruptionBudget {
		pdb := newPodDisruptionBudget(bindplane, transformAgentComponent)
		if err := r.reconcilePodDisruptionBudget(ctx, bindplane, pdb, log); err != nil {
			return err
		}
	} else {
		if err := r.deletePodDisruptionBudgetIfExists(ctx, bindplane, transformAgentComponent, log); err != nil {
			return err
		}
	}

	return nil
}

func (r *BindplaneReconciler) transformAgentServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	var annotations map[string]string
	if bindplane.Spec.TransformAgent != nil && bindplane.Spec.TransformAgent.ServiceAccount != nil {
		annotations = bindplane.Spec.TransformAgent.ServiceAccount.Annotations
	}
	return newServiceAccount(bindplane, transformAgentComponent, annotations)
}

func (r *BindplaneReconciler) transformAgentDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := *bindplane.Spec.TransformAgent.Replicas
	labels := getLabels(bindplane, transformAgentComponent)
	selectorLabels := getSelectorLabels(bindplane, transformAgentComponent)
	tlsVols, tlsMounts := getTransformAgentTLSVolumesAndMounts(bindplane)

	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(1)

	taResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
	if bindplane.Spec.TransformAgent.Resources != nil {
		taResources = *bindplane.Spec.TransformAgent.Resources
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, transformAgentComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						Volumes:            tlsVols,
						ServiceAccountName: getResourceName(bindplane, transformAgentComponent),
						SecurityContext:    newPodSecurityContext(),
						Affinity:           getTransformAgentAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:         transformAgentContainerName,
								Image:        getTransformAgentImage(bindplane),
								VolumeMounts: tlsMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          transformAgentHTTPPortName,
										ContainerPort: transformAgentHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: prependExtraEnv(
									getTransformAgentExtraEnv(bindplane),
									getKubernetesEnvVars(transformAgentContainerName),
									getTransformAgentTLSEnvVars(bindplane),
								),
								Resources: taResources,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(transformAgentHTTPPortName),
										},
									},
									InitialDelaySeconds: probeStartupInitialDelaySeconds,
									PeriodSeconds:       probeStartupPeriodSeconds,
									FailureThreshold:    probeStartupFailureThreshold,
									SuccessThreshold:    probeStartupSuccessThreshold,
									TimeoutSeconds:      probeStartupTimeoutSeconds,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(transformAgentHTTPPortName),
										},
									},
									PeriodSeconds:    probePeriodSeconds,
									FailureThreshold: probeFailureThreshold,
									SuccessThreshold: probeSuccessThreshold,
									TimeoutSeconds:   probeTimeoutSeconds,
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(transformAgentHTTPPortName),
										},
									},
									PeriodSeconds:    probePeriodSeconds,
									FailureThreshold: probeFailureThreshold,
									SuccessThreshold: probeSuccessThreshold,
									TimeoutSeconds:   probeTimeoutSeconds,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
							},
						},
						TerminationGracePeriodSeconds: new(defaultTerminationGracePeriodSeconds),
					},
				},
				getTransformAgentPodTemplate(bindplane),
			),
		},
	}
}

// getTransformAgentAffinity returns the affinity for Transform Agent pods.
// If the user provides podTemplate.spec.affinity, that takes precedence.
// Otherwise returns the default pod anti-affinity (spread by hostname).
func getTransformAgentAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	if bindplane.Spec.TransformAgent != nil && bindplane.Spec.TransformAgent.PodTemplate != nil &&
		bindplane.Spec.TransformAgent.PodTemplate.Spec.Affinity != nil {
		return bindplane.Spec.TransformAgent.PodTemplate.Spec.Affinity
	}
	return defaultPodAntiAffinity(bindplane, transformAgentComponent)
}

// getTransformAgentPodTemplate returns the user-provided pod template spec for Transform Agent
func getTransformAgentPodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.TransformAgent != nil {
		return bindplane.Spec.TransformAgent.PodTemplate
	}
	return nil
}

// getTransformAgentExtraEnv returns the user-supplied extra env vars for Transform Agent, or nil.
func getTransformAgentExtraEnv(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	if bindplane.Spec.TransformAgent != nil {
		return bindplane.Spec.TransformAgent.ExtraEnv
	}
	return nil
}

func (r *BindplaneReconciler) transformAgentService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, transformAgentComponent, WithPort(transformAgentHTTPPortName, transformAgentHTTPPort))
}
