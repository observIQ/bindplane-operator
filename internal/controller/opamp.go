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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// opampComponent is the component name for the dedicated OpAMP deployment.
	opampComponent = "opamp"
	// opampContainerName matches the existing nodeContainerName so that the
	// podTemplate merge keys ("server") work for both deployments.
	opampContainerName = nodeContainerName
	// opampHTTPPort matches nodeHTTPPort because the Bindplane image listens
	// on the same port regardless of which Deployment hosts it.
	opampHTTPPort = nodeHTTPPort
	// opampHTTPPortName matches nodeHTTPPortName.
	opampHTTPPortName = nodeHTTPPortName
	// opampModeValue is BINDPLANE_MODE for the OpAMP deployment. It is the
	// same as nodeModeValue because both Deployments serve node-mode traffic;
	// the OpAMP deployment is just a second instance of node mode that is
	// scaled and routed separately.
	opampModeValue = nodeModeValue
)

// reconcileOpAMP reconciles the optional OpAMP deployment and its
// supporting resources. When spec.bindplane.opamp is nil or not enabled, this
// is a delete-if-exists pass to clean up resources from a previously enabled
// OpAMP deployment.
func (r *BindplaneReconciler) reconcileOpAMP(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	if bindplane.Spec.Bindplane.OpAMP == nil || !bindplane.Spec.Bindplane.OpAMP.Enabled {
		return r.deleteOpAMPResourcesIfExist(ctx, bindplane, log)
	}

	// Reconcile ServiceAccount
	sa := r.opampServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile Deployment
	deployment := r.opampDeployment(bindplane)
	if err := r.reconcileDeployment(ctx, bindplane, deployment, log); err != nil {
		return err
	}

	// Reconcile Service
	service := r.opampService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	// Reconcile PodDisruptionBudget
	if !bindplane.Spec.Bindplane.OpAMP.DisablePodDisruptionBudget {
		pdb := newPodDisruptionBudget(bindplane, opampComponent)
		if err := r.reconcilePodDisruptionBudget(ctx, bindplane, pdb, log); err != nil {
			return err
		}
	} else {
		if err := r.deletePodDisruptionBudgetIfExists(ctx, bindplane, opampComponent, log); err != nil {
			return err
		}
	}

	// Reconcile HorizontalPodAutoscaler
	return r.reconcileOpAMPHPA(ctx, bindplane, log)
}

// deleteOpAMPResourcesIfExist deletes Deployment, Service, ServiceAccount, PDB,
// and HPA for the OpAMP component if they exist. Used when the user disables
// OpAMP after previously enabling it.
func (r *BindplaneReconciler) deleteOpAMPResourcesIfExist(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	name := getResourceName(bindplane, opampComponent)
	ns := bindplane.Namespace

	// Delete Deployment
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, dep); err == nil {
		log.Info("Deleting OpAMP Deployment", "name", name)
		if err := r.Delete(ctx, dep); err != nil && !errors.IsNotFound(err) {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Delete Service
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, svc); err == nil {
		log.Info("Deleting OpAMP Service", "name", name)
		if err := r.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Delete ServiceAccount
	sa := &corev1.ServiceAccount{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, sa); err == nil {
		log.Info("Deleting OpAMP ServiceAccount", "name", name)
		if err := r.Delete(ctx, sa); err != nil && !errors.IsNotFound(err) {
			return err
		}
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Delete PDB
	if err := r.deletePodDisruptionBudgetIfExists(ctx, bindplane, opampComponent, log); err != nil {
		return err
	}

	// Delete HPA
	return r.deleteHPAIfExists(ctx, bindplane, opampComponent, log)
}

func (r *BindplaneReconciler) opampServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, opampComponent)
}

func (r *BindplaneReconciler) opampDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	cfg := bindplane.Spec.Bindplane.OpAMP

	// When autoscaling is enabled, do not set Replicas on the Deployment so the
	// HorizontalPodAutoscaler has exclusive control over the replica count.
	var replicaPtr *int32
	if cfg.Autoscaling == nil || !cfg.Autoscaling.Enabled {
		// Use the configured replica count; fall back to the CRD default (3) when the
		// field is nil (e.g. when the object is constructed directly in unit tests
		// rather than through the API server which applies CRD defaulting).
		defaultOpAMPReplicas := int32(3)
		replicas := defaultOpAMPReplicas
		if cfg.Replicas != nil {
			replicas = *cfg.Replicas
		}
		replicaPtr = &replicas
	}

	labels := getLabels(bindplane, opampComponent)
	selectorLabels := getSelectorLabels(bindplane, opampComponent)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)
	terminationGracePeriod := nodeTerminationGracePeriodSeconds(bindplane)

	// Default minReadySeconds to the termination grace period so that agents
	// draining from the outgoing pod have time to reconnect to healthy nodes
	// before the next pod is taken out of service.
	minReadySeconds := int32(terminationGracePeriod) // #nosec G115 -- grace period is always a small positive value
	if cfg.MinReadySeconds != nil {
		minReadySeconds = *cfg.MinReadySeconds
	}

	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxSurge:       &maxSurge,
			MaxUnavailable: &maxUnavailable,
		},
	}
	if cfg.Strategy != nil {
		strategy = *cfg.Strategy
	}

	opampResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("2048Mi"),
		},
	}
	if cfg.Resources != nil {
		opampResources = *cfg.Resources
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, opampComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:        replicaPtr,
			MinReadySeconds: minReadySeconds,
			Strategy:        strategy,
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
						ServiceAccountName: getResourceName(bindplane, opampComponent),
						SecurityContext:    newPodSecurityContext(),
						Affinity:           defaultPodAntiAffinity(bindplane, opampComponent),
						Containers: []corev1.Container{
							{
								Name:         opampContainerName,
								Image:        getBindplaneEEImage(bindplane),
								VolumeMounts: configMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          opampHTTPPortName,
										ContainerPort: opampHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: combineEnvVars(
									getKubernetesEnvVars(opampContainerName),
									getNodeEnvVars(),
									getBindplaneCommonEnvVars(bindplane, opampComponent),
									getNatsClientEnvVars(bindplane, true),
									getOpAMPOverrideEnvVars(bindplane),
								),
								Resources: opampResources,
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(opampHTTPPortName),
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
											Port: intstr.FromString(opampHTTPPortName),
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
											Port: intstr.FromString(opampHTTPPortName),
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
				cfg.PodTemplate,
			),
		},
	}
}

// getOpAMPOverrideEnvVars returns OpAMP-specific environment variable overrides.
// These OVERRIDE the shared environment built by getBindplaneCommonEnvVars (which includes
// the cluster-wide BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS and
// BINDPLANE_PROFILING_SERVICE_NAME). Place the result AFTER getBindplaneCommonEnvVars
// and getNatsClientEnvVars in the env slice so Kubernetes uses the later (OpAMP-specific) value.
func getOpAMPOverrideEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	// Override profiling service name to be <bindplane.Name>-opamp for this specific CR.
	// getBindplaneCommonEnvVars emits "bindplane-opamp" (static). The override below
	// makes it CR-name-specific, matching the Node deployment's <bindplane.Name>-node pattern.
	if bindplane.Spec.Config.Profiling != nil && bindplane.Spec.Config.Profiling.Enabled {
		envVars = append(envVars, corev1.EnvVar{
			Name:  bindplaneProfilingServiceNameEnvVar,
			Value: fmt.Sprintf("%s-opamp", bindplane.Name),
		})
	}

	// Per-deployment override for max simultaneous connections.
	if bindplane.Spec.Bindplane.OpAMP != nil && bindplane.Spec.Bindplane.OpAMP.MaxSimultaneousConnections != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  bindplaneAgentsMaxSimultaneousConnectionsEnvVar,
			Value: fmt.Sprintf("%d", *bindplane.Spec.Bindplane.OpAMP.MaxSimultaneousConnections),
		})
	}

	// Per-deployment override for OpAMP shutdown grace period target.
	if bindplane.Spec.Bindplane.OpAMP != nil && bindplane.Spec.Bindplane.OpAMP.ShutdownGracePeriodTarget != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  bindplaneAdvancedServerOpAMPShutdownGracePeriodTargetEnvVar,
			Value: bindplane.Spec.Bindplane.OpAMP.ShutdownGracePeriodTarget,
		})
	}

	return envVars
}

func (r *BindplaneReconciler) opampService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	return newService(bindplane, opampComponent, WithPort(opampHTTPPortName, opampHTTPPort))
}

// reconcileOpAMPHPA creates, updates, or deletes the HorizontalPodAutoscaler for the OpAMP deployment.
// When autoscaling is disabled (or the Autoscaling field is nil), any existing HPA is deleted
// so that the static replica count from the Deployment takes effect.
func (r *BindplaneReconciler) reconcileOpAMPHPA(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	if bindplane.Spec.Bindplane.OpAMP == nil || bindplane.Spec.Bindplane.OpAMP.Autoscaling == nil || !bindplane.Spec.Bindplane.OpAMP.Autoscaling.Enabled {
		return r.deleteHPAIfExists(ctx, bindplane, opampComponent, log)
	}

	hpa := r.opampHPA(bindplane)
	return r.reconcileHorizontalPodAutoscaler(ctx, bindplane, hpa, log)
}

// opampHPA builds the HorizontalPodAutoscaler for the OpAMP deployment, merging user-provided
// overrides with the same defaults used for the Node HPA.
func (r *BindplaneReconciler) opampHPA(bindplane *bindplanev1alpha1.Bindplane) *autoscalingv2.HorizontalPodAutoscaler {
	cfg := bindplane.Spec.Bindplane.OpAMP.Autoscaling
	labels := getLabels(bindplane, opampComponent)

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
			Name:      getResourceName(bindplane, opampComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       getResourceName(bindplane, opampComponent),
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics:     metrics,
			Behavior:    behavior,
		},
	}
}
