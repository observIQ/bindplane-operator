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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// nodeComponent is the component name for Bindplane Node
	nodeComponent = "node"
	// nodeContainerName is the container name for Bindplane Node
	nodeContainerName = "server"
	// nodeHTTPPort is the HTTP port for Bindplane Node
	nodeHTTPPort = int32(3001)
	// nodeHTTPPortName is the name of the HTTP port for Bindplane Node
	nodeHTTPPortName = "http"
	// nodeModeValue is the value for BINDPLANE_MODE
	nodeModeValue = "node"
)

// reconcileNode reconciles all Bindplane Node resources
func (r *BindplaneReconciler) reconcileNode(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.nodeServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.nodeDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.nodeService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	// Reconcile PodDisruptionBudget
	if !bindplane.Spec.Bindplane.DisablePodDisruptionBudget {
		pdb := newPodDisruptionBudget(bindplane, nodeComponent)
		if err := r.reconcilePodDisruptionBudget(ctx, bindplane, pdb, log); err != nil {
			return err
		}
	} else {
		if err := r.deletePodDisruptionBudgetIfExists(ctx, bindplane, nodeComponent, log); err != nil {
			return err
		}
	}

	return nil
}

func (r *BindplaneReconciler) nodeServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, nodeComponent)
}

func (r *BindplaneReconciler) nodeDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := *bindplane.Spec.Bindplane.Replicas
	labels := getLabels(bindplane, nodeComponent)
	selectorLabels := getSelectorLabels(bindplane, nodeComponent)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)
	terminationGracePeriod := nodeTerminationGracePeriodSeconds(bindplane)

	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, nodeComponent),
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
						Volumes:            configVols,
						ServiceAccountName: getResourceName(bindplane, nodeComponent),
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    new(defaultRunAsGroup),
							RunAsGroup: new(defaultRunAsGroup),
							RunAsUser:  new(defaultRunAsUser),
						},
						Affinity: getNodeAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:         nodeContainerName,
								Image:        getBindplaneEEImage(bindplane),
								VolumeMounts: configMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          nodeHTTPPortName,
										ContainerPort: nodeHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: combineEnvVars(
									getKubernetesEnvVars(nodeContainerName),
									getNodeEnvVars(),
									getBindplaneCommonEnvVars(bindplane, nodeComponent),
									getNatsClientEnvVars(bindplane, true),
								),
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("2048Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("2000m"),
										corev1.ResourceMemory: resource.MustParse("2048Mi"),
									},
								},
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

// getNodeEnvVars returns the Node-specific environment variables
// Includes mode and NATS client configuration (but not NATS server config)
func getNodeEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  bindplaneModeEnvVar,
			Value: nodeModeValue,
		},
	}
}

// getNodeAffinity returns the default pod anti-affinity for Node pods.
// Spreads pods across nodes by hostname (preferred, weight 100).
// Overridden by the user's podTemplate.spec.affinity if provided.
func getNodeAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	return defaultPodAntiAffinity(bindplane, nodeComponent)
}

// getNodePodTemplate returns the user-provided pod template spec for Node
func getNodePodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	return bindplane.Spec.Bindplane.PodTemplate
}

func (r *BindplaneReconciler) nodeService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, nodeComponent, WithPort(nodeHTTPPortName, nodeHTTPPort))
}

// nodeTerminationGracePeriodSeconds returns the termination grace period for the Node deployment.
// When spec.config.advanced.server.opampShutdownGracePeriod is set, the grace period is 125% of
// that value (rounded up to the next whole second) to ensure the pod outlives the OpAMP shutdown.
// Falls back to defaultTerminationGracePeriodSeconds when the field is unset or unparseable.
func nodeTerminationGracePeriodSeconds(bindplane *bindplanev1alpha1.Bindplane) int64 {
	if bindplane.Spec.Config.Advanced == nil || bindplane.Spec.Config.Advanced.Server == nil {
		return defaultTerminationGracePeriodSeconds
	}
	opampPeriod := bindplane.Spec.Config.Advanced.Server.OpAMPShutdownGracePeriod
	if opampPeriod == "" {
		return defaultTerminationGracePeriodSeconds
	}
	d, err := time.ParseDuration(opampPeriod)
	if err != nil || d <= 0 {
		return defaultTerminationGracePeriodSeconds
	}
	// 125% of d, rounded up to the next whole second using integer arithmetic.
	ns := d.Nanoseconds() * 5 / 4
	return (ns + int64(time.Second) - 1) / int64(time.Second)
}
