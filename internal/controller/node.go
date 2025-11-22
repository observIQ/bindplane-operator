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

const (
	// nodeComponent is the component name for Bindplane Node
	nodeComponent = "node"
	// nodeContainerName is the container name for Bindplane Node
	nodeContainerName = "server"
	// nodeImage is the default container image for Bindplane Node (same as NATS)
	nodeImage = natsImage
	// nodeHTTPPort is the HTTP port for Bindplane Node
	nodeHTTPPort = int32(3001)
	// nodeHTTPPortName is the name of the HTTP port for Bindplane Node
	nodeHTTPPortName = "http"
	// nodeModeValue is the value for BINDPLANE_MODE
	nodeModeValue = "node"
	// nodeReplicas is the number of node replicas
	nodeReplicas = int32(2)
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

	return nil
}

func (r *BindplaneReconciler) nodeServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount {
	return newServiceAccount(bindplane, nodeComponent)
}

func (r *BindplaneReconciler) nodeDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment {
	replicas := nodeReplicas
	labels := getLabels(bindplane, nodeComponent)
	selectorLabels := getSelectorLabels(bindplane, nodeComponent)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, nodeComponent),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			// Default rollout strategy (RollingUpdate)
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: mergePodTemplateSpec(
				corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selectorLabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: getResourceName(bindplane, nodeComponent),
						SecurityContext: &corev1.PodSecurityContext{
							FSGroup:    int64Ptr(65534),
							RunAsGroup: int64Ptr(65534),
							RunAsUser:  int64Ptr(65534),
						},
						Affinity: getNodeAffinity(bindplane),
						Containers: []corev1.Container{
							{
								Name:  nodeContainerName,
								Image: nodeImage,
								Ports: []corev1.ContainerPort{
									{
										Name:          nodeHTTPPortName,
										ContainerPort: nodeHTTPPort,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: append(
									getKubernetesEnvVars(nodeContainerName),
									append(
										getNodeEnvVars(bindplane),
										getBindplaneConfigEnvVars(bindplane)...,
									)...,
								),
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("512Mi"),
									},
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("200m"),
										corev1.ResourceMemory: resource.MustParse("512Mi"),
									},
								},
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.FromString(nodeHTTPPortName),
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
											Path: "/healthz",
											Port: intstr.FromString(nodeHTTPPortName),
										},
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.FromString(nodeHTTPPortName),
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
				getNodePodTemplate(bindplane),
			),
		},
	}
}

// getNodeEnvVars returns the Node-specific environment variables
// Same as NATS StatefulSet but without NATS server config, only client config
func getNodeEnvVars(bindplane *bindplanev1alpha1.Bindplane) []corev1.EnvVar {
	natsServiceName := getResourceName(bindplane, natsComponent)

	return []corev1.EnvVar{
		{
			Name:  bindplaneJobsModeEnvVar,
			Value: nodeModeValue,
		},
		{
			Name:  "BINDPLANE_EVENT_BUS_TYPE",
			Value: "nats",
		},
		{
			Name: "BINDPLANE_NATS_CLIENT_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "BINDPLANE_NATS_CLIENT_ENDPOINT",
			Value: fmt.Sprintf("nats://%s.%s:4222", natsServiceName, bindplane.Namespace),
		},
		{
			Name:  "BINDPLANE_NATS_CLIENT_SUBJECT",
			Value: "bindplane-event-bus",
		},
	}
}

// getNodeAffinity returns the affinity configuration for Node pods
// This is a fallback for when user doesn't provide podTemplate - will be overridden by mergePodTemplateSpec
func getNodeAffinity(bindplane *bindplanev1alpha1.Bindplane) *corev1.Affinity {
	// Node doesn't have a pod template in the spec, so return nil
	return nil
}

// getNodePodTemplate returns the user-provided pod template spec for Node
func getNodePodTemplate(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PodTemplateSpec {
	// Node doesn't have a pod template in the spec, so return nil
	return nil
}
