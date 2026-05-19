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
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
)

const (
	// natsComponent is the component name for NATS
	natsComponent = "nats"
	// natsContainerName is the container name for NATS
	natsContainerName = "server"
	// natsClientPort is the NATS client port
	natsClientPort = int32(4222)
	// natsClientPortName is the name of the NATS client port
	natsClientPortName = "client"
	// natsHTTPPort is the NATS HTTP/monitoring port
	natsHTTPPort = int32(8222)
	// natsHTTPPortName is the name of the NATS HTTP port
	natsHTTPPortName = "http"
	// natsClusterPort is the NATS cluster port
	natsClusterPort = int32(6222)
	// natsClusterPortName is the name of the NATS cluster port
	natsClusterPortName = "cluster"
)

// reconcileNats reconciles all NATS resources
func (r *BindplaneReconciler) reconcileNats(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	// Reconcile ServiceAccount
	sa := r.natsServiceAccount(bindplane)
	if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
		return err
	}

	// Reconcile StatefulSet
	statefulSet := r.natsStatefulSet(bindplane)
	if err := r.reconcileStatefulSet(ctx, bindplane, statefulSet, log); err != nil {
		return err
	}

	// Reconcile Headless Service for cluster communication
	headlessService := r.natsHeadlessService(bindplane)
	if err := r.reconcileService(ctx, bindplane, headlessService, log); err != nil {
		return err
	}

	// Reconcile regular Service for client connections
	service := r.natsService(bindplane)
	if err := r.reconcileService(ctx, bindplane, service, log); err != nil {
		return err
	}

	// Reconcile PodDisruptionBudget
	if bindplane.Spec.Nats == nil || !bindplane.Spec.Nats.DisablePodDisruptionBudget {
		pdb := newPodDisruptionBudget(bindplane, natsComponent)
		if err := r.reconcilePodDisruptionBudget(ctx, bindplane, pdb, log); err != nil {
			return err
		}
	} else {
		if err := r.deletePodDisruptionBudgetIfExists(ctx, bindplane, natsComponent, log); err != nil {
			return err
		}
	}

	return nil
}

func (r *BindplaneReconciler) natsServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, natsComponent)
}

func (r *BindplaneReconciler) natsStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := *bindplane.Spec.Nats.Replicas
	labels := getLabels(bindplane, natsComponent)
	selectorLabels := getSelectorLabels(bindplane, natsComponent)
	serviceName := getResourceName(bindplane, natsComponent)
	headlessServiceName := getNatsClusterServiceName(bindplane)
	configVols, configMounts := getConfigTLSVolumesAndMounts(bindplane)

	natsResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("500Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("500Mi"),
		},
	}
	if bindplane.Spec.Nats.Resources != nil {
		natsResources = *bindplane.Spec.Nats.Resources
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         headlessServiceName,
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
						Volumes:            configVols,
						ServiceAccountName: serviceName,
						SecurityContext:    newPodSecurityContext(),
						Affinity:           getNatsAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:         natsContainerName,
								Image:        getNatsImage(bindplane),
								VolumeMounts: configMounts,
								Ports: []corev1.ContainerPort{
									{
										Name:          natsClientPortName,
										ContainerPort: natsClientPort,
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          natsHTTPPortName,
										ContainerPort: natsHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          natsClusterPortName,
										ContainerPort: natsClusterPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: prependExtraEnv(
									getNatsExtraEnv(bindplane),
									getKubernetesEnvVars(natsContainerName),
									getNatsEnvVars(bindplane, headlessServiceName, replicas),
									getBindplaneCommonEnvVars(bindplane),
								),
								Resources: natsResources,
								// TODO(jsirianni): When NATS TLS is enabled the HTTP port serves TLS; Kubernetes HTTPGet does not support TLS. Use TCPSocket for now; add Bindplane CLI healthchecks that support exec probes for proper TLS healthcheck when available.
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.FromString(natsHTTPPortName),
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
											Port: intstr.FromString(natsHTTPPortName),
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
											Port: intstr.FromString(natsHTTPPortName),
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
						TerminationGracePeriodSeconds: new(defaultTerminationGracePeriodSeconds),
					},
				},
				getNatsPodTemplate(bindplane),
			),
		},
	}
}

func (r *BindplaneReconciler) natsHeadlessService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	labels := getLabels(bindplane, natsComponent)
	selectorLabels := getSelectorLabels(bindplane, natsComponent)
	serviceName := getNatsClusterServiceName(bindplane)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector:  selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       natsClientPortName,
					Port:       natsClientPort,
					TargetPort: intstr.FromInt32(natsClientPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       natsHTTPPortName,
					Port:       natsHTTPPort,
					TargetPort: intstr.FromInt32(natsHTTPPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       natsClusterPortName,
					Port:       natsClusterPort,
					TargetPort: intstr.FromInt32(natsClusterPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func (r *BindplaneReconciler) natsService(bindplane *bindplanev1alpha1.Bindplane) *corev1.Service {
	labels := getLabels(bindplane, natsComponent)
	selectorLabels := getSelectorLabels(bindplane, natsComponent)
	serviceName := getNatsClientServiceName(bindplane)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
			Ports: []corev1.ServicePort{
				{
					Name:       natsClientPortName,
					Port:       natsClientPort,
					TargetPort: intstr.FromInt32(natsClientPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// getNatsEnvVars returns the NATS-specific environment variables
func getNatsEnvVars(bindplane *bindplanev1alpha1.Bindplane, headlessServiceName string, replicas int32) []corev1.EnvVar {
	clusterName := fmt.Sprintf("%s-%s", bindplane.Name, natsComponent)
	clusterRoutes := getNatsClusterRoutes(bindplane, headlessServiceName, replicas)

	tlsVars := getNatsTLSEnvVars(bindplane)
	envVars := make([]corev1.EnvVar, 0, 15+len(tlsVars))
	envVars = append(envVars,
		corev1.EnvVar{Name: bindplaneModeEnvVar, Value: natsModeValue},
		corev1.EnvVar{Name: bindplaneEventBusTypeEnvVar, Value: natsEventBusType},
		corev1.EnvVar{Name: bindplaneNatsServerEnableEnvVar, Value: natsServerEnableValue},
		corev1.EnvVar{
			Name: bindplaneNatsServerNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		corev1.EnvVar{Name: bindplaneNatsServerClientHostEnvVar, Value: natsBindAddress},
		corev1.EnvVar{Name: bindplaneNatsServerClientPortEnvVar, Value: strconv.Itoa(int(natsClientPort))},
		corev1.EnvVar{Name: bindplaneNatsServerHTTPHostEnvVar, Value: natsBindAddress},
		corev1.EnvVar{Name: bindplaneNatsServerHTTPPortEnvVar, Value: strconv.Itoa(int(natsHTTPPort))},
		corev1.EnvVar{Name: bindplaneNatsServerClusterNameEnvVar, Value: clusterName},
		corev1.EnvVar{Name: bindplaneNatsServerClusterHostEnvVar, Value: natsBindAddress},
		corev1.EnvVar{Name: bindplaneNatsServerClusterPortEnvVar, Value: strconv.Itoa(int(natsClusterPort))},
		corev1.EnvVar{Name: bindplaneNatsServerClusterRoutesEnvVar, Value: clusterRoutes},
		corev1.EnvVar{
			Name: bindplaneNatsClientNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		corev1.EnvVar{Name: bindplaneNatsClientEndpointEnvVar, Value: fmt.Sprintf("%s%s", natsProtocolPrefix, natsLocalhostEndpoint)},
		corev1.EnvVar{Name: bindplaneNatsClientSubjectEnvVar, Value: natsClientSubject},
	)
	envVars = append(envVars, tlsVars...)
	return envVars
}

// getNatsClusterRoutes generates the cluster routes string for NATS
// It generates routes based on the actual replica count (e.g., 3 replicas = all 3 hostnames, 1 replica = first hostname only)
func getNatsClusterRoutes(bindplane *bindplanev1alpha1.Bindplane, headlessServiceName string, replicas int32) string {
	routes := make([]string, 0, replicas)
	for i := range replicas {
		route := fmt.Sprintf("%s%s-%d.%s.%s:%d",
			natsProtocolPrefix,
			getResourceName(bindplane, natsComponent),
			i,
			headlessServiceName,
			bindplane.Namespace,
			natsClusterPort)
		routes = append(routes, route)
	}
	var result strings.Builder
	for i, route := range routes {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(route)
	}
	return result.String()
}

// getNatsAffinity returns the default pod anti-affinity for NATS pods.
// Spreads pods across nodes by hostname (preferred, weight 100).
// Overridden by the user's podTemplate.spec.affinity if provided.
func getNatsAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	return defaultPodAntiAffinity(bindplane, natsComponent)
}

// getNatsPodTemplate returns the user-provided pod template spec for NATS
func getNatsPodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	if bindplane.Spec.Nats != nil {
		return bindplane.Spec.Nats.PodTemplate
	}
	return nil
}

// getNatsExtraEnv returns the user-supplied extra env vars for NATS, or nil.
func getNatsExtraEnv(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	if bindplane.Spec.Nats != nil {
		return bindplane.Spec.Nats.ExtraEnv
	}
	return nil
}
