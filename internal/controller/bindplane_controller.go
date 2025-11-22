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
// Operator-managed fields (ServiceAccountName, containers, labels) take precedence.
func mergePodTemplateSpec(operatorManaged corev1.PodTemplateSpec, userProvided *bindplanev1alpha1.PodTemplateSpec) corev1.PodTemplateSpec {
	if userProvided == nil {
		return operatorManaged
	}

	merged := operatorManaged.DeepCopy()

	// Merge metadata (labels and annotations)
	// Protect operator-managed selector labels from user overrides
	// These labels are critical for service selectors to match pods
	protectedLabelKeys := []string{
		labelKeyName,
		labelKeyInstance,
		labelKeyComponent,
	}
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

	// Merge pod spec - allow user overrides for scheduling-related fields
	userSpec := userProvided.Spec

	// Allow user to override affinity
	if userSpec.Affinity != nil {
		merged.Spec.Affinity = userSpec.Affinity
	}

	// Allow user to override tolerations
	if userSpec.Tolerations != nil {
		merged.Spec.Tolerations = userSpec.Tolerations
	}

	// Allow user to override nodeSelector
	if userSpec.NodeSelector != nil {
		merged.Spec.NodeSelector = userSpec.NodeSelector
	}

	// Allow user to override runtimeClassName
	if userSpec.RuntimeClassName != nil {
		merged.Spec.RuntimeClassName = userSpec.RuntimeClassName
	}

	// Allow user to override priorityClassName
	if userSpec.PriorityClassName != "" {
		merged.Spec.PriorityClassName = userSpec.PriorityClassName
	}

	// Allow user to override schedulerName
	if userSpec.SchedulerName != "" {
		merged.Spec.SchedulerName = userSpec.SchedulerName
	}

	// Allow user to override hostNetwork, hostPID, hostIPC
	if userSpec.HostNetwork {
		merged.Spec.HostNetwork = userSpec.HostNetwork
	}
	if userSpec.HostPID {
		merged.Spec.HostPID = userSpec.HostPID
	}
	if userSpec.HostIPC {
		merged.Spec.HostIPC = userSpec.HostIPC
	}

	// Allow user to override DNS settings
	if userSpec.DNSPolicy != "" {
		merged.Spec.DNSPolicy = userSpec.DNSPolicy
	}
	if userSpec.DNSConfig != nil {
		merged.Spec.DNSConfig = userSpec.DNSConfig
	}

	// Allow user to override initContainers (but preserve operator containers)
	if len(userSpec.InitContainers) > 0 {
		merged.Spec.InitContainers = userSpec.InitContainers
	}

	// Allow user to override volumes (merge with operator volumes)
	if len(userSpec.Volumes) > 0 {
		// Create a map of existing volumes by name to avoid duplicates
		volumeMap := make(map[string]corev1.Volume)
		for _, vol := range merged.Spec.Volumes {
			volumeMap[vol.Name] = vol
		}
		// Add user volumes, allowing them to override operator volumes with same name
		for _, vol := range userSpec.Volumes {
			volumeMap[vol.Name] = vol
		}
		// Convert back to slice
		merged.Spec.Volumes = make([]corev1.Volume, 0, len(volumeMap))
		for _, vol := range volumeMap {
			merged.Spec.Volumes = append(merged.Spec.Volumes, vol)
		}
	}

	// Allow user to override imagePullSecrets
	if len(userSpec.ImagePullSecrets) > 0 {
		merged.Spec.ImagePullSecrets = userSpec.ImagePullSecrets
	}

	// Allow user to override securityContext (merge carefully)
	if userSpec.SecurityContext != nil {
		// Merge security context - user values override operator values
		if merged.Spec.SecurityContext == nil {
			merged.Spec.SecurityContext = &corev1.PodSecurityContext{}
		}
		userSC := userSpec.SecurityContext
		if userSC.FSGroup != nil {
			merged.Spec.SecurityContext.FSGroup = userSC.FSGroup
		}
		if userSC.RunAsGroup != nil {
			merged.Spec.SecurityContext.RunAsGroup = userSC.RunAsGroup
		}
		if userSC.RunAsUser != nil {
			merged.Spec.SecurityContext.RunAsUser = userSC.RunAsUser
		}
		if userSC.RunAsNonRoot != nil {
			merged.Spec.SecurityContext.RunAsNonRoot = userSC.RunAsNonRoot
		}
		if userSC.SupplementalGroups != nil {
			merged.Spec.SecurityContext.SupplementalGroups = userSC.SupplementalGroups
		}
		if userSC.SELinuxOptions != nil {
			merged.Spec.SecurityContext.SELinuxOptions = userSC.SELinuxOptions
		}
		if userSC.SeccompProfile != nil {
			merged.Spec.SecurityContext.SeccompProfile = userSC.SeccompProfile
		}
	}

	// Note: ServiceAccountName, Containers, and TerminationGracePeriodSeconds are
	// operator-managed and are NOT overridden by user input

	return *merged
}
