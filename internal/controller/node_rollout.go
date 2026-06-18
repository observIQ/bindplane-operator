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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	argoRolloutAPIVersion = "argoproj.io/v1alpha1"
	argoRolloutKind       = "Rollout"
)

// nodeRollout builds the Argo Rollout for Bindplane Node. The pod template, name,
// labels, and selector are identical to those used by nodeDeployment, so HPAs,
// PDBs, and Services keep working without modification.
func (r *BindplaneReconciler) nodeRollout(bindplane *bindplanev1alpha1.Bindplane) *rolloutsv1alpha1.Rollout {
	var replicaPtr *int32
	if bindplane.Spec.Bindplane.Autoscaling == nil || !bindplane.Spec.Bindplane.Autoscaling.Enabled {
		defaultReplicas := int32(3)
		replicas := defaultReplicas
		if bindplane.Spec.Bindplane.Replicas != nil {
			replicas = *bindplane.Spec.Bindplane.Replicas
		}
		replicaPtr = &replicas
	}

	labels := getLabels(bindplane, nodeComponent)
	selectorLabels := getSelectorLabels(bindplane, nodeComponent)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)
	configVols = appendExtraVolumes(configVols, getNodeExtraVolumes(bindplane))
	configMounts = appendExtraVolumeMounts(configMounts, getNodeExtraVolumeMounts(bindplane))
	terminationGracePeriod := nodeTerminationGracePeriodSeconds(bindplane)

	minReadySeconds := int32(terminationGracePeriod) // #nosec G115 -- grace period is always a small positive value
	if bindplane.Spec.Bindplane.MinReadySeconds != nil {
		minReadySeconds = *bindplane.Spec.Bindplane.MinReadySeconds
	}

	nodeResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
	}
	if bindplane.Spec.Bindplane.Resources != nil {
		nodeResources = *bindplane.Spec.Bindplane.Resources
	}

	autoPromote := true
	if bindplane.Spec.Bindplane.ArgoRollout != nil && bindplane.Spec.Bindplane.ArgoRollout.AutoPromotionEnabled != nil {
		autoPromote = *bindplane.Spec.Bindplane.ArgoRollout.AutoPromotionEnabled
	}

	blueGreen := rolloutsv1alpha1.BlueGreenStrategy{
		ActiveService:        getResourceName(bindplane, nodeComponent),
		AutoPromotionEnabled: &autoPromote,
	}
	if bindplane.Spec.Bindplane.ArgoRollout != nil && bindplane.Spec.Bindplane.ArgoRollout.ScaleDownDelaySeconds != nil {
		blueGreen.ScaleDownDelaySeconds = bindplane.Spec.Bindplane.ArgoRollout.ScaleDownDelaySeconds
	}

	return &rolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, nodeComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: rolloutsv1alpha1.RolloutSpec{
			Replicas:        replicaPtr,
			MinReadySeconds: minReadySeconds,
			Strategy: rolloutsv1alpha1.RolloutStrategy{
				BlueGreen: &blueGreen,
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
						Volumes:            configVols,
						ServiceAccountName: getResourceName(bindplane, nodeComponent),
						SecurityContext:    newPodSecurityContext(),
						Affinity:           getNodeAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:         nodeContainerName,
								Image:        getNodeImage(bindplane),
								VolumeMounts: configMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          nodeHTTPPortName,
										ContainerPort: nodeHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: prependExtraEnv(
									bindplane.Spec.Bindplane.ExtraEnv,
									getKubernetesEnvVars(nodeContainerName),
									getNodeEnvVars(),
									getBindplaneCommonEnvVars(bindplane),
									getNatsClientEnvVars(bindplane),
								),
								Resources: nodeResources,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(nodeHTTPPortName),
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
											Port: intstr.FromString(nodeHTTPPortName),
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
											Port: intstr.FromString(nodeHTTPPortName),
										},
									},
									PeriodSeconds:    probePeriodSeconds,
									FailureThreshold: probeFailureThreshold,
									SuccessThreshold: probeSuccessThreshold,
									TimeoutSeconds:   probeTimeoutSeconds,
								},
								SecurityContext: newContainerSecurityContext(WithRunAsUser(defaultRunAsUser)),
								ImagePullPolicy: corev1.PullIfNotPresent,
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{preStopCommand, preStopArgs, preStopSleep},
										},
									},
								},
							},
						},
						TerminationGracePeriodSeconds: &terminationGracePeriod,
					},
				},
				getNodePodTemplate(bindplane),
			),
		},
	}
}
