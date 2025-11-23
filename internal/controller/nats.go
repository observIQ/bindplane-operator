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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

const (
	// natsComponent is the component name for NATS
	natsComponent = "nats"
	// natsContainerName is the container name for NATS
	natsContainerName = "server"
	// natsImage is the default container image for NATS (same as jobs)
	natsImage = bindplaneJobsImage
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
	// natsReplicas is the number of NATS replicas
	natsReplicas = int32(3)
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

	return nil
}

func (r *BindplaneReconciler) natsServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, natsComponent)
}

func (r *BindplaneReconciler) natsStatefulSet(bindplane *bindplanev1alpha1.Bindplane) *appsv1.StatefulSet {
	replicas := natsReplicas
	labels := getLabels(bindplane, natsComponent)
	selectorLabels := getSelectorLabels(bindplane, natsComponent)
	serviceName := getResourceName(bindplane, natsComponent)
	headlessServiceName := getNatsClusterServiceName(bindplane)

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
						ServiceAccountName: serviceName,
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    int64Ptr(defaultRunAsGroup),
							RunAsGroup: int64Ptr(defaultRunAsGroup),
							RunAsUser:  int64Ptr(defaultRunAsUser),
						},
						Affinity: getNatsAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  natsContainerName,
								Image: natsImage,
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
								Env: combineEnvVars(
									getKubernetesEnvVars(natsContainerName),
									getNatsEnvVars(bindplane, serviceName, headlessServiceName),
									getBindplaneConfigEnvVars(bindplane),
									getPrometheusEnvVars(bindplane),
									getTransformAgentEnvVars(bindplane),
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
											Path: healthzCheckPath,
											Port: intstr.FromString(natsHTTPPortName),
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
											Path: healthzCheckPath,
											Port: intstr.FromString(natsHTTPPortName),
										},
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: healthzCheckPath,
											Port: intstr.FromString(natsHTTPPortName),
										},
									},
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
						TerminationGracePeriodSeconds: int64Ptr(defaultTerminationGracePeriodSeconds),
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
func getNatsEnvVars(bindplane *bindplanev1alpha1.Bindplane, serviceName, headlessServiceName string) []corev1.EnvVar {
	clusterName := fmt.Sprintf("%s-%s", bindplane.Name, natsComponent)
	clusterRoutes := getNatsClusterRoutes(bindplane, headlessServiceName)

	return []corev1.EnvVar{
		{
			Name:  bindplaneModeEnvVar,
			Value: natsModeValue,
		},
		{
			Name:  bindplaneEventBusTypeEnvVar,
			Value: natsEventBusType,
		},
		{
			Name:  bindplaneNatsServerEnableEnvVar,
			Value: natsServerEnableValue,
		},
		{
			Name: bindplaneNatsServerNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		{
			Name:  bindplaneNatsServerClientHostEnvVar,
			Value: natsBindAddress,
		},
		{
			Name:  bindplaneNatsServerClientPortEnvVar,
			Value: strconv.Itoa(int(natsClientPort)),
		},
		{
			Name:  bindplaneNatsServerHTTPHostEnvVar,
			Value: natsBindAddress,
		},
		{
			Name:  bindplaneNatsServerHTTPPortEnvVar,
			Value: strconv.Itoa(int(natsHTTPPort)),
		},
		{
			Name:  bindplaneNatsServerClusterNameEnvVar,
			Value: clusterName,
		},
		{
			Name:  bindplaneNatsServerClusterHostEnvVar,
			Value: natsBindAddress,
		},
		{
			Name:  bindplaneNatsServerClusterPortEnvVar,
			Value: strconv.Itoa(int(natsClusterPort)),
		},
		{
			Name:  bindplaneNatsServerClusterRoutesEnvVar,
			Value: clusterRoutes,
		},
		{
			Name: bindplaneNatsClientNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: metadataNameFieldPath,
				},
			},
		},
		{
			Name:  bindplaneNatsClientEndpointEnvVar,
			Value: fmt.Sprintf("%s%s", natsProtocolPrefix, natsLocalhostEndpoint),
		},
		{
			Name:  bindplaneNatsClientSubjectEnvVar,
			Value: natsClientSubject,
		},
	}
}

// getNatsClusterRoutes generates the cluster routes string for NATS
func getNatsClusterRoutes(bindplane *bindplanev1alpha1.Bindplane, headlessServiceName string) string {
	var routes []string
	for i := int32(0); i < natsReplicas; i++ {
		route := fmt.Sprintf("%s%s-%d.%s.%s:%d",
			natsProtocolPrefix,
			getResourceName(bindplane, natsComponent),
			i,
			headlessServiceName,
			bindplane.Namespace,
			natsClusterPort)
		routes = append(routes, route)
	}
	result := ""
	for i, route := range routes {
		if i > 0 {
			result += ","
		}
		result += route
	}
	return result
}

// getNatsAffinity returns the affinity configuration for NATS pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getNatsAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	// NATS doesn't have a pod template in the spec, so return nil
	return nil
}

// getNatsPodTemplate returns the user-provided pod template spec for NATS
func getNatsPodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	// NATS doesn't have a pod template in the spec, so return nil
	return nil
}
