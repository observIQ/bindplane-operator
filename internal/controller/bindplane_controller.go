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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	bindplanev1alpha1 "github.com/bindplane-operator/bindplane-operator/api/v1alpha1"
)

// Label key constants for Kubernetes standard labels
const (
	labelKeyName      = "app.kubernetes.io/name"
	labelKeyInstance  = "app.kubernetes.io/instance"
	labelKeyComponent = "app.kubernetes.io/component"
	labelKeyManagedBy = "app.kubernetes.io/managed-by"
	labelKeyPartOf    = "app.kubernetes.io/part-of"
)

// Label value constants
const (
	labelValueName      = "bindplane"
	labelValueManagedBy = "bindplane-operator"
	labelValuePartOf    = "bindplane"
)

// Kubernetes environment variable name constants
const (
	kubernetesNamespaceNameEnvVar = "KUBERNETES_NAMESPACE_NAME"
	kubernetesPodNameEnvVar       = "KUBERNETES_POD_NAME"
	kubernetesContainerNameEnvVar = "KUBERNETES_CONTAINER_NAME"
)

// Bindplane environment variable name constants
const (
	// Core Bindplane configuration
	bindplaneModeEnvVar      = "BINDPLANE_MODE"
	bindplaneLicenseEnvVar   = "BINDPLANE_LICENSE"
	bindplaneAuthTypeEnvVar  = "BINDPLANE_AUTH_TYPE"
	bindplaneUsernameEnvVar  = "BINDPLANE_USERNAME"
	bindplanePasswordEnvVar  = "BINDPLANE_PASSWORD"
	bindplaneHostEnvVar      = "BINDPLANE_HOST"
	bindplanePortEnvVar      = "BINDPLANE_PORT"
	bindplaneRemoteURLEnvVar = "BINDPLANE_REMOTE_URL"

	// Store configuration
	bindplaneStoreTypeEnvVar = "BINDPLANE_STORE_TYPE"

	// Postgres configuration
	bindplanePostgresHostEnvVar             = "BINDPLANE_POSTGRES_HOST"
	bindplanePostgresPortEnvVar             = "BINDPLANE_POSTGRES_PORT"
	bindplanePostgresConnectTimeoutEnvVar   = "BINDPLANE_POSTGRES_CONNECT_TIMEOUT"
	bindplanePostgresStatementTimeoutEnvVar = "BINDPLANE_POSTGRES_STATEMENT_TIMEOUT"
	bindplanePostgresDatabaseEnvVar         = "BINDPLANE_POSTGRES_DATABASE"
	bindplanePostgresSSLModeEnvVar          = "BINDPLANE_POSTGRES_SSL_MODE"
	bindplanePostgresUsernameEnvVar         = "BINDPLANE_POSTGRES_USERNAME"
	bindplanePostgresPasswordEnvVar         = "BINDPLANE_POSTGRES_PASSWORD"
	bindplanePostgresMaxConnectionsEnvVar   = "BINDPLANE_POSTGRES_MAX_CONNECTIONS"
	bindplanePostgresMaxLifetimeEnvVar      = "BINDPLANE_POSTGRES_MAX_LIFETIME"
	bindplanePostgresSchemaEnvVar           = "BINDPLANE_POSTGRES_SCHEMA"

	// Prometheus configuration
	bindplanePrometheusEnableRemoteEnvVar = "BINDPLANE_PROMETHEUS_ENABLE_REMOTE"
	bindplanePrometheusHostEnvVar         = "BINDPLANE_PROMETHEUS_HOST"
	bindplanePrometheusPortEnvVar         = "BINDPLANE_PROMETHEUS_PORT"

	// Transform Agent configuration
	bindplaneTransformAgentEnableRemoteEnvVar = "BINDPLANE_TRANSFORM_AGENT_ENABLE_REMOTE"
	bindplaneTransformAgentRemoteAgentsEnvVar = "BINDPLANE_TRANSFORM_AGENT_REMOTE_AGENTS"

	// Event Bus configuration
	bindplaneEventBusTypeEnvVar = "BINDPLANE_EVENT_BUS_TYPE"

	// NATS client configuration
	bindplaneNatsClientNameEnvVar     = "BINDPLANE_NATS_CLIENT_NAME"
	bindplaneNatsClientEndpointEnvVar = "BINDPLANE_NATS_CLIENT_ENDPOINT"
	bindplaneNatsClientSubjectEnvVar  = "BINDPLANE_NATS_CLIENT_SUBJECT"

	// NATS server configuration
	bindplaneNatsServerEnableEnvVar        = "BINDPLANE_NATS_SERVER_ENABLE"
	bindplaneNatsServerNameEnvVar          = "BINDPLANE_NATS_SERVER_NAME"
	bindplaneNatsServerClientHostEnvVar    = "BINDPLANE_NATS_SERVER_CLIENT_HOST"
	bindplaneNatsServerClientPortEnvVar    = "BINDPLANE_NATS_SERVER_CLIENT_PORT"
	bindplaneNatsServerHTTPHostEnvVar      = "BINDPLANE_NATS_SERVER_HTTP_HOST"
	bindplaneNatsServerHTTPPortEnvVar      = "BINDPLANE_NATS_SERVER_HTTP_PORT"
	bindplaneNatsServerClusterNameEnvVar   = "BINDPLANE_NATS_SERVER_CLUSTER_NAME"
	bindplaneNatsServerClusterHostEnvVar   = "BINDPLANE_NATS_SERVER_CLUSTER_HOST"
	bindplaneNatsServerClusterPortEnvVar   = "BINDPLANE_NATS_SERVER_CLUSTER_PORT"
	bindplaneNatsServerClusterRoutesEnvVar = "BINDPLANE_NATS_SERVER_CLUSTER_ROUTES"
)

// Common security and pod constants
const (
	// defaultRunAsUser is the default user ID for security contexts
	defaultRunAsUser = int64(65534)
	// defaultRunAsGroup is the default group ID for security contexts
	defaultRunAsGroup = int64(65534)
	// defaultTerminationGracePeriodSeconds is the default termination grace period
	defaultTerminationGracePeriodSeconds = int64(60)
	// defaultContainerName is the default container name used across deployments
	defaultContainerName = "server"
	// defaultHTTPPortName is the default HTTP port name
	defaultHTTPPortName = "http"
	// metadataNameFieldPath is the field path for pod metadata.name
	metadataNameFieldPath = "metadata.name"
	// metadataNamespaceFieldPath is the field path for pod metadata.namespace
	metadataNamespaceFieldPath = "metadata.namespace"
	// preStopCommand is the command used in preStop lifecycle hooks
	preStopCommand = "sh"
	// preStopArgs is the arguments for preStop lifecycle hooks
	preStopArgs = "-c"
	// preStopSleep is the sleep command for preStop hooks
	preStopSleep = "sleep 5"
)

// Health check path constants
const (
	// healthCheckPath is the HTTP path for health checks (used by jobs)
	healthCheckPath = "/health"
	// healthzCheckPath is the HTTP path for healthz checks (used by NATS and node)
	healthzCheckPath = "/healthz"
)

// NATS constants
const (
	// natsServiceClientSuffix is the suffix for NATS client service name
	natsServiceClientSuffix = "-client"
	// natsServiceClusterSuffix is the suffix for NATS cluster service name
	natsServiceClusterSuffix = "-cluster"
	// natsEventBusType is the event bus type value for NATS
	natsEventBusType = "nats"
	// natsClientSubject is the NATS client subject name
	natsClientSubject = "bindplane-event-bus"
	// natsProtocolPrefix is the NATS protocol prefix
	natsProtocolPrefix = "nats://"
	// natsLocalhostEndpoint is the localhost NATS endpoint
	natsLocalhostEndpoint = "127.0.0.1:4222"
	// natsBindAddress is the bind address for NATS servers
	natsBindAddress = "0.0.0.0"
	// natsModeValue is the BINDPLANE_MODE value for NATS nodes
	natsModeValue = "node"
	// natsServerEnableValue is the value to enable NATS server
	natsServerEnableValue = "true"
	// enableRemoteValue is the value to enable remote services
	enableRemoteValue = "true"
)

// BindplaneReconciler reconciles a Bindplane object
type BindplaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.bindplane.com,resources=bindplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.bindplane.com,resources=bindplanes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.bindplane.com,resources=bindplanes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *BindplaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Bindplane instance
	bindplane := &bindplanev1alpha1.Bindplane{}
	if err := r.Get(ctx, req.NamespacedName, bindplane); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Bindplane")
		return ctrl.Result{}, err
	}

	// Reconcile Transform Agent resources
	if err := r.reconcileTransformAgent(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile Transform Agent")
		return ctrl.Result{}, err
	}

	// Reconcile Prometheus resources
	if err := r.reconcilePrometheus(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile Prometheus")
		return ctrl.Result{}, err
	}

	// Reconcile Bindplane Jobs resources
	if err := r.reconcileBindplaneJobs(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile Bindplane Jobs")
		return ctrl.Result{}, err
	}

	// Reconcile NATS resources
	if err := r.reconcileNats(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile NATS")
		return ctrl.Result{}, err
	}

	// Reconcile Node resources
	if err := r.reconcileNode(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile Node")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BindplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bindplanev1alpha1.Bindplane{}).
		Named("bindplane").
		Complete(r)
}

// getLabels returns the standard labels for Bindplane resources
func getLabels(bindplane *bindplanev1alpha1.Bindplane, component string) map[string]string {
	return map[string]string{
		labelKeyName:      labelValueName,
		labelKeyInstance:  bindplane.Name,
		labelKeyComponent: component,
		labelKeyManagedBy: labelValueManagedBy,
		labelKeyPartOf:    labelValuePartOf,
	}
}

// getSelectorLabels returns the labels used for selectors (subset of getLabels)
func getSelectorLabels(bindplane *bindplanev1alpha1.Bindplane, component string) map[string]string {
	return map[string]string{
		labelKeyName:      labelValueName,
		labelKeyInstance:  bindplane.Name,
		labelKeyComponent: component,
	}
}

// getResourceName returns a standardized resource name for a component
func getResourceName(bindplane *bindplanev1alpha1.Bindplane, component string) string {
	return fmt.Sprintf("%s-%s", bindplane.Name, component)
}

// getNatsClientServiceName returns the NATS client service name
func getNatsClientServiceName(bindplane *bindplanev1alpha1.Bindplane) string {
	return fmt.Sprintf("%s%s", getResourceName(bindplane, natsComponent), natsServiceClientSuffix)
}

// getNatsClusterServiceName returns the NATS cluster (headless) service name
func getNatsClusterServiceName(bindplane *bindplanev1alpha1.Bindplane) string {
	return fmt.Sprintf("%s%s", getResourceName(bindplane, natsComponent), natsServiceClusterSuffix)
}

// getNatsClientEndpoint returns the NATS client endpoint URL
func getNatsClientEndpoint(bindplane *bindplanev1alpha1.Bindplane) string {
	return fmt.Sprintf("%s%s.%s:%d", natsProtocolPrefix, getNatsClientServiceName(bindplane), bindplane.Namespace, natsClientPort)
}

// Generic reconcile functions

func (r *BindplaneReconciler) reconcileServiceAccount(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, sa *corev1.ServiceAccount, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, sa, r.Scheme); err != nil {
		return err
	}

	found := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: sa.Name, Namespace: sa.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating ServiceAccount", "name", sa.Name, "namespace", sa.Namespace)
		return r.Create(ctx, sa)
	} else if err != nil {
		return err
	}

	// ServiceAccount is mostly immutable, but we can update labels/annotations if needed
	found.Labels = sa.Labels
	if err := r.Update(ctx, found); err != nil {
		return err
	}
	return nil
}

func (r *BindplaneReconciler) reconcileDeployment(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, deployment *appsv1.Deployment, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, deployment, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "name", deployment.Name, "namespace", deployment.Namespace)
		return r.Create(ctx, deployment)
	} else if err != nil {
		return err
	}

	// Update deployment spec if needed
	found.Spec = deployment.Spec
	found.Labels = deployment.Labels
	if err := r.Update(ctx, found); err != nil {
		return err
	}
	return nil
}

func (r *BindplaneReconciler) reconcileStatefulSet(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, statefulSet *appsv1.StatefulSet, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, statefulSet, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating StatefulSet", "name", statefulSet.Name, "namespace", statefulSet.Namespace)
		return r.Create(ctx, statefulSet)
	} else if err != nil {
		return err
	}

	// Update statefulset spec if needed (be careful with StatefulSet updates)
	found.Spec.Replicas = statefulSet.Spec.Replicas
	found.Spec.Template = statefulSet.Spec.Template
	found.Labels = statefulSet.Labels
	if err := r.Update(ctx, found); err != nil {
		return err
	}
	return nil
}

func (r *BindplaneReconciler) reconcileService(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, service *corev1.Service, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, service, r.Scheme); err != nil {
		return err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "name", service.Name, "namespace", service.Namespace)
		return r.Create(ctx, service)
	} else if err != nil {
		return err
	}

	// Update service spec (preserve clusterIP)
	found.Spec.Ports = service.Spec.Ports
	found.Spec.Selector = service.Spec.Selector
	found.Labels = service.Labels
	if err := r.Update(ctx, found); err != nil {
		return err
	}
	return nil
}

// Helper functions

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

// getKubernetesEnvVars returns the common Kubernetes environment variables
// that should be present in all pods deployed by this operator
// combineEnvVars combines multiple slices of environment variables into a single slice
func combineEnvVars(envVarSlices ...[]corev1.EnvVar) []corev1.EnvVar {
	var result []corev1.EnvVar
	for _, envVars := range envVarSlices {
		result = append(result, envVars...)
	}
	return result
}

func getKubernetesEnvVars(containerName string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: kubernetesNamespaceNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: kubernetesPodNameEnvVar,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  kubernetesContainerNameEnvVar,
			Value: containerName,
		},
	}
}

// securityContextOptions holds configuration options for creating a SecurityContext
type securityContextOptions struct {
	runAsUser *int64
}

// securityContextOption is a function that configures securityContextOptions
type securityContextOption func(*securityContextOptions)

// WithRunAsUser sets the RunAsUser for the container security context
func WithRunAsUser(userID int64) securityContextOption {
	return func(opts *securityContextOptions) {
		opts.runAsUser = &userID
	}
}

// newContainerSecurityContext creates a secure container security context
// It accepts variadic securityContextOption functions to configure overrides
func newContainerSecurityContext(opts ...securityContextOption) *corev1.SecurityContext {
	// Apply default options
	options := &securityContextOptions{
		runAsUser: int64Ptr(65534), // Default to nobody user
	}

	// Apply all option functions
	for _, opt := range opts {
		opt(options)
	}

	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: boolPtr(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		ReadOnlyRootFilesystem: boolPtr(true),
		RunAsNonRoot:           boolPtr(true),
		RunAsUser:              options.runAsUser,
	}
}

// newServiceAccount creates a ServiceAccount for a component
func newServiceAccount(bindplane *bindplanev1alpha1.Bindplane, component string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, component),
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, component),
		},
	}
}

// serviceOptions holds configuration options for creating a Service
type serviceOptions struct {
	ports []corev1.ServicePort
}

// serviceOption is a function that configures serviceOptions
type serviceOption func(*serviceOptions)

// WithPort adds a single port to the service
// The port will be used for both Port and TargetPort
// Call WithPort multiple times to add multiple ports
func WithPort(name string, port int32) serviceOption {
	return func(opts *serviceOptions) {
		opts.ports = append(opts.ports, corev1.ServicePort{
			Name:       name,
			Port:       port,
			TargetPort: intstr.FromInt(int(port)),
			Protocol:   corev1.ProtocolTCP,
		})
	}
}

// newService creates a ClusterIP Service for a component
// It accepts variadic serviceOption functions to configure ports
func newService(bindplane *bindplanev1alpha1.Bindplane, component string, opts ...serviceOption) *corev1.Service {
	labels := getLabels(bindplane, component)
	selectorLabels := getSelectorLabels(bindplane, component)

	// Apply default options
	options := &serviceOptions{
		ports: []corev1.ServicePort{},
	}

	// Apply all option functions
	for _, opt := range opts {
		opt(options)
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, component),
			Namespace: bindplane.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabels,
			Ports:    options.ports,
		},
	}
}

// mergePodTemplateSpec merges user-provided pod template spec with operator-managed fields.
// It supports ANY arbitrary field in the pod spec, while protecting only critical operator-managed fields.
// Protected fields: ServiceAccountName, container names/images/ports/env/command/args, protected labels, TerminationGracePeriodSeconds
func mergePodTemplateSpec(operatorManaged corev1.PodTemplateSpec, userProvided *bindplanev1alpha1.PodTemplateSpec) corev1.PodTemplateSpec {
	if userProvided == nil {
		return operatorManaged
	}

	// Deep copy operator-managed spec as the base
	merged := operatorManaged.DeepCopy()

	// Save protected fields before merging
	protectedServiceAccountName := merged.Spec.ServiceAccountName
	protectedTerminationGracePeriodSeconds := merged.Spec.TerminationGracePeriodSeconds
	protectedContainers := make([]corev1.Container, len(merged.Spec.Containers))
	for i, c := range merged.Spec.Containers {
		protectedContainers[i] = *c.DeepCopy()
	}
	protectedLabelKeys := []string{
		labelKeyName,
		labelKeyInstance,
		labelKeyComponent,
	}
	protectedLabels := make(map[string]string)
	for _, key := range protectedLabelKeys {
		if val, exists := merged.ObjectMeta.Labels[key]; exists {
			protectedLabels[key] = val
		}
	}

	// Deep merge user-provided spec on top of operator spec
	// This allows ANY field to be overridden
	userSpecCopy := userProvided.Spec.DeepCopy()

	// Merge metadata (labels and annotations)
	if userProvided.ObjectMeta.Labels != nil {
		if merged.ObjectMeta.Labels == nil {
			merged.ObjectMeta.Labels = make(map[string]string)
		}
		for k, v := range userProvided.ObjectMeta.Labels {
			// Skip protected labels - operator-managed labels take precedence
			isProtected := false
			for _, protectedKey := range protectedLabelKeys {
				if k == protectedKey {
					isProtected = true
					break
				}
			}
			if !isProtected {
				merged.ObjectMeta.Labels[k] = v
			}
		}
	}
	if userProvided.ObjectMeta.Annotations != nil {
		if merged.ObjectMeta.Annotations == nil {
			merged.ObjectMeta.Annotations = make(map[string]string)
		}
		for k, v := range userProvided.ObjectMeta.Annotations {
			merged.ObjectMeta.Annotations[k] = v
		}
	}

	// Merge all pod spec fields using JSON marshal/unmarshal for deep merge
	// This allows ANY field to be merged, not just a curated list
	operatorSpecJSON, err := json.Marshal(merged.Spec)
	if err != nil {
		// Fallback to operator spec if marshal fails
		return *merged
	}

	userSpecJSON, err := json.Marshal(userSpecCopy)
	if err != nil {
		// Fallback to operator spec if marshal fails
		return *merged
	}

	// Merge JSON objects
	var operatorSpecMap map[string]interface{}
	var userSpecMap map[string]interface{}
	if err := json.Unmarshal(operatorSpecJSON, &operatorSpecMap); err != nil {
		return *merged
	}
	if err := json.Unmarshal(userSpecJSON, &userSpecMap); err != nil {
		return *merged
	}

	// Deep merge user spec into operator spec
	mergeMaps(operatorSpecMap, userSpecMap)

	// Convert back to PodSpec
	mergedJSON, err := json.Marshal(operatorSpecMap)
	if err != nil {
		return *merged
	}
	if err := json.Unmarshal(mergedJSON, &merged.Spec); err != nil {
		return *merged
	}

	// Restore protected fields
	merged.Spec.ServiceAccountName = protectedServiceAccountName
	merged.Spec.TerminationGracePeriodSeconds = protectedTerminationGracePeriodSeconds

	// Merge containers by name - allow user to override any container field except protected ones
	if len(userProvided.Spec.Containers) > 0 {
		containerMap := make(map[string]corev1.Container)
		for _, c := range protectedContainers {
			containerMap[c.Name] = c
		}

		// For each user container, merge it with the operator container
		for _, userContainer := range userProvided.Spec.Containers {
			if operatorContainer, exists := containerMap[userContainer.Name]; exists {
				// Deep copy operator container
				mergedContainer := operatorContainer.DeepCopy()

				// Merge user container fields using JSON
				operatorContainerJSON, _ := json.Marshal(mergedContainer)
				userContainerJSON, _ := json.Marshal(userContainer)
				var operatorContainerMap map[string]interface{}
				var userContainerMap map[string]interface{}
				if err := json.Unmarshal(operatorContainerJSON, &operatorContainerMap); err == nil {
					if err := json.Unmarshal(userContainerJSON, &userContainerMap); err == nil {
						mergeMaps(operatorContainerMap, userContainerMap)
						mergedContainerJSON, _ := json.Marshal(operatorContainerMap)
						json.Unmarshal(mergedContainerJSON, mergedContainer)
					}
				}

				// Restore protected container fields
				mergedContainer.Name = operatorContainer.Name
				mergedContainer.Image = operatorContainer.Image
				mergedContainer.Ports = operatorContainer.Ports
				mergedContainer.Env = operatorContainer.Env
				mergedContainer.Command = operatorContainer.Command
				mergedContainer.Args = operatorContainer.Args

				containerMap[userContainer.Name] = *mergedContainer
			}
		}

		// Update merged containers
		merged.Spec.Containers = make([]corev1.Container, len(protectedContainers))
		for i, c := range protectedContainers {
			if updated, exists := containerMap[c.Name]; exists {
				merged.Spec.Containers[i] = updated
			} else {
				merged.Spec.Containers[i] = c
			}
		}
	} else {
		// No user containers, use protected containers
		merged.Spec.Containers = protectedContainers
	}

	return *merged
}

// mergeMaps recursively merges map b into map a
func mergeMaps(a, b map[string]interface{}) {
	for k, v := range b {
		if v == nil {
			continue
		}
		if av, exists := a[k]; exists {
			if avMap, ok := av.(map[string]interface{}); ok {
				if bvMap, ok := v.(map[string]interface{}); ok {
					mergeMaps(avMap, bvMap)
					continue
				}
			}
		}
		a[k] = v
	}
}
