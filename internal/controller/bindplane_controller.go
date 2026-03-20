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
	"maps"
	"net"
	"regexp"
	"slices"
	"unicode"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	bindplanev1alpha1 "github.com/observiq/bindplane-operator/api/v1alpha1"
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
	bindplaneModeEnvVar                    = "BINDPLANE_MODE"
	bindplaneLicenseEnvVar                 = "BINDPLANE_LICENSE"
	bindplaneAuthTypeEnvVar                = "BINDPLANE_AUTH_TYPE"
	bindplaneAuthSessionsStrictModeEnvVar  = "BINDPLANE_AUTH_SESSIONS_STRICT_MODE"
	bindplaneUsernameEnvVar                = "BINDPLANE_USERNAME"
	bindplanePasswordEnvVar                = "BINDPLANE_PASSWORD" // #nosec G101 -- env var name, not a credential
	bindplaneHostEnvVar                    = "BINDPLANE_HOST"
	bindplanePortEnvVar                    = "BINDPLANE_PORT"
	bindplaneRemoteURLEnvVar               = "BINDPLANE_REMOTE_URL"
	bindplaneWebURLEnvVar                  = "BINDPLANE_WEB_URL"
	bindplaneCorsAllowedOriginsEnvVar      = "BINDPLANE_CORS_ALLOWED_ORIGINS"
	bindplaneMaxConcurrencyEnvVar          = "BINDPLANE_MAX_CONCURRENCY"
	bindplaneAuditTrailRetentionDaysEnvVar = "BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS"

	// LDAP / Active Directory configuration
	bindplaneLDAPProtocolEnvVar      = "BINDPLANE_LDAP_PROTOCOL"
	bindplaneLDAPServerEnvVar        = "BINDPLANE_LDAP_SERVER"
	bindplaneLDAPPortEnvVar          = "BINDPLANE_LDAP_PORT"
	bindplaneLDAPBaseDNEnvVar        = "BINDPLANE_LDAP_BASE_DN"
	bindplaneLDAPBindUserEnvVar      = "BINDPLANE_LDAP_BIND_USER"
	bindplaneLDAPBindPasswordEnvVar  = "BINDPLANE_LDAP_BIND_PASSWORD" // #nosec G101 -- env var name, not a credential
	bindplaneLDAPSearchFilterEnvVar  = "BINDPLANE_LDAP_SEARCH_FILTER"
	bindplaneLDAPTLSCertEnvVar       = "BINDPLANE_LDAP_TLS_CERT"
	bindplaneLDAPTLSKeyEnvVar        = "BINDPLANE_LDAP_TLS_KEY"
	bindplaneLDAPTLSCAEnvVar         = "BINDPLANE_LDAP_TLS_CA"
	bindplaneLDAPTLSSkipVerifyEnvVar = "BINDPLANE_LDAP_TLS_SKIP_VERIFY"

	// LDAP TLS volume mount (operator-managed path; user specifies only Secret name and keys)
	ldapTLSVolumeName = "ldap-tls"
	ldapTLSMountPath  = "/etc/bindplane/ldap-tls"

	// Network TLS (server or mutual TLS); operator mounts Secret and sets BINDPLANE_TLS_* env vars
	bindplaneTLSMinVersionEnvVar = "BINDPLANE_TLS_MIN_VERSION"
	bindplaneTLSCertEnvVar       = "BINDPLANE_TLS_CERT"
	bindplaneTLSKeyEnvVar        = "BINDPLANE_TLS_KEY"
	bindplaneTLSCAEnvVar         = "BINDPLANE_TLS_CA"
	bindplaneTLSSkipVerifyEnvVar = "BINDPLANE_TLS_SKIP_VERIFY"

	// Network TLS volume mount (operator-managed path; user specifies only Secret name and keys)
	networkTLSVolumeName = "network-tls"
	networkTLSMountPath  = "/etc/bindplane/network-tls"

	// OIDC configuration
	bindplaneOIDCClientIDEnvVar     = "BINDPLANE_OIDC_OAUTH2_CLIENT_ID"
	bindplaneOIDCClientSecretEnvVar = "BINDPLANE_OIDC_OAUTH2_CLIENT_SECRET" // #nosec G101 -- env var name, not a credential
	bindplaneOIDCIssuerEnvVar       = "BINDPLANE_OIDC_ISSUER"
	bindplaneOIDCScopesEnvVar       = "BINDPLANE_OIDC_SCOPES"

	// Store configuration
	bindplaneStoreTypeEnvVar = "BINDPLANE_STORE_TYPE"

	// Tracing configuration
	bindplaneTracingTypeEnvVar         = "BINDPLANE_TRACING_TYPE"
	bindplaneTracingOTLPEndpointEnvVar = "BINDPLANE_TRACING_OTLP_ENDPOINT"
	bindplaneTracingOTLPInsecureEnvVar = "BINDPLANE_TRACING_OTLP_INSECURE"
	bindplaneTracingSamplingRateEnvVar = "BINDPLANE_TRACING_SAMPLING_RATE"

	// Metrics configuration
	bindplaneMetricsTypeEnvVar               = "BINDPLANE_METRICS_TYPE"
	bindplaneMetricsIntervalEnvVar           = "BINDPLANE_METRICS_INTERVAL"
	bindplaneMetricsPrometheusEndpointEnvVar = "BINDPLANE_METRICS_PROMETHEUS_ENDPOINT"
	bindplaneMetricsPrometheusUsernameEnvVar = "BINDPLANE_METRICS_PROMETHEUS_USERNAME"
	bindplaneMetricsPrometheusPasswordEnvVar = "BINDPLANE_METRICS_PROMETHEUS_PASSWORD" // #nosec G101 -- env var name, not a credential
	bindplaneMetricsOTLPEndpointEnvVar       = "BINDPLANE_METRICS_OTLP_ENDPOINT"
	bindplaneMetricsOTLPInsecureEnvVar       = "BINDPLANE_METRICS_OTLP_INSECURE"

	// Postgres configuration
	bindplanePostgresHostEnvVar               = "BINDPLANE_POSTGRES_HOST"
	bindplanePostgresPortEnvVar               = "BINDPLANE_POSTGRES_PORT"
	bindplanePostgresConnectTimeoutEnvVar     = "BINDPLANE_POSTGRES_CONNECT_TIMEOUT"
	bindplanePostgresStatementTimeoutEnvVar   = "BINDPLANE_POSTGRES_STATEMENT_TIMEOUT"
	bindplanePostgresDatabaseEnvVar           = "BINDPLANE_POSTGRES_DATABASE"
	bindplanePostgresSSLModeEnvVar            = "BINDPLANE_POSTGRES_SSL_MODE"
	bindplanePostgresSSLRootCertEnvVar        = "BINDPLANE_POSTGRES_SSL_ROOT_CERT"
	bindplanePostgresSSLCertEnvVar            = "BINDPLANE_POSTGRES_SSL_CERT"
	bindplanePostgresSSLKeyEnvVar             = "BINDPLANE_POSTGRES_SSL_KEY"
	bindplanePostgresUsernameEnvVar           = "BINDPLANE_POSTGRES_USERNAME"
	bindplanePostgresPasswordEnvVar           = "BINDPLANE_POSTGRES_PASSWORD" // #nosec G101 -- env var name, not a credential
	bindplanePostgresMaxConnectionsEnvVar     = "BINDPLANE_POSTGRES_MAX_CONNECTIONS"
	bindplanePostgresMaxIdleConnectionsEnvVar = "BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS"
	bindplanePostgresMaxLifetimeEnvVar        = "BINDPLANE_POSTGRES_MAX_LIFETIME"
	bindplanePostgresMaxIdleTimeEnvVar        = "BINDPLANE_POSTGRES_MAX_IDLE_TIME"
	bindplanePostgresSchemaEnvVar             = "BINDPLANE_POSTGRES_SCHEMA"

	// Postgres SSL mode values (must match CRD enum: disable|require|verify-ca|verify-full)
	postgresSSLModeDisable    = "disable"
	postgresSSLModeRequire    = "require"
	postgresSSLModeVerifyCA   = "verify-ca"
	postgresSSLModeVerifyFull = "verify-full"

	// Postgres TLS volume mount (operator-managed path; user specifies only Secret name and keys)
	postgresTLSVolumeName = "postgres-tls"
	postgresTLSMountPath  = "/etc/bindplane/postgres-tls"

	// Internal TLS (cert-manager) volume mount for TSDB remote write client cert
	internalTLSTSDBClientVolumeName = "tsdb-remote-write-tls"
	internalTLSTSDBClientMountPath  = "/etc/bindplane/tsdb-remote-write-tls"

	// Internal TLS (cert-manager) volume mount for NATS (client, cluster, HTTP)
	internalTLSNatsVolumeName = "nats-tls"
	internalTLSNatsMountPath  = "/etc/bindplane/nats-tls"

	// Prometheus configuration
	bindplaneTSDBEnableRemoteEnvVar        = "BINDPLANE_PROMETHEUS_ENABLE_REMOTE"
	bindplaneTSDBHostEnvVar                = "BINDPLANE_PROMETHEUS_HOST"
	bindplaneTSDBPortEnvVar                = "BINDPLANE_PROMETHEUS_PORT"
	bindplaneTSDBQueryPathPrefixEnvVar     = "BINDPLANE_PROMETHEUS_QUERY_PATH_PREFIX"
	bindplaneTSDBRemoteWriteHostEnvVar     = "BINDPLANE_PROMETHEUS_REMOTE_WRITE_HOST"
	bindplaneTSDBRemoteWritePortEnvVar     = "BINDPLANE_PROMETHEUS_REMOTE_WRITE_PORT"
	bindplaneTSDBRemoteWriteEndpointEnvVar = "BINDPLANE_PROMETHEUS_REMOTE_WRITE_ENDPOINT"
	bindplaneTSDBAuthUsernameEnvVar        = "BINDPLANE_PROMETHEUS_AUTH_USERNAME"
	bindplaneTSDBAuthPasswordEnvVar        = "BINDPLANE_PROMETHEUS_AUTH_PASSWORD" // #nosec G101 -- env var name, not a credential

	// Prometheus remote write TLS (cert-manager internal mTLS)
	bindplaneTSDBEnableTLSEnvVar     = "BINDPLANE_PROMETHEUS_ENABLE_TLS"
	bindplaneTSDBTLSCertEnvVar       = "BINDPLANE_PROMETHEUS_TLS_CERT"
	bindplaneTSDBTLSKeyEnvVar        = "BINDPLANE_PROMETHEUS_TLS_KEY"
	bindplaneTSDBTLSCAEnvVar         = "BINDPLANE_PROMETHEUS_TLS_CA"
	bindplaneTSDBTLSSkipVerifyEnvVar = "BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY"

	// Transform Agent configuration
	bindplaneTransformAgentEnableRemoteEnvVar = "BINDPLANE_TRANSFORM_AGENT_ENABLE_REMOTE"
	bindplaneTransformAgentRemoteAgentsEnvVar = "BINDPLANE_TRANSFORM_AGENT_REMOTE_AGENTS"

	// Event Bus configuration
	bindplaneEventBusTypeEnvVar                = "BINDPLANE_EVENT_BUS_TYPE"
	bindplaneEventBusHealthRequiredHostsEnvVar = "BINDPLANE_EVENT_BUS_HEALTH_REQUIRED_HOSTS"
	bindplaneEventBusHealthIntervalEnvVar      = "BINDPLANE_EVENT_BUS_HEALTH_INTERVAL"

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

	// NATS TLS (cert-manager; no skip-verify exposed)
	bindplaneNatsEnableTLSEnvVar = "BINDPLANE_NATS_ENABLE_TLS"
	bindplaneNatsTLSCertEnvVar   = "BINDPLANE_NATS_TLS_CERT"
	bindplaneNatsTLSKeyEnvVar    = "BINDPLANE_NATS_TLS_KEY"
	bindplaneNatsTLSCAEnvVar     = "BINDPLANE_NATS_TLS_CA"

	// Profiling (Google Cloud Profiler)
	bindplaneProfilingEnabledEnvVar     = "BINDPLANE_PROFILING_ENABLED"
	bindplaneProfilingProjectIDEnvVar   = "BINDPLANE_PROFILING_PROJECT_ID"
	bindplaneProfilingServiceNameEnvVar = "BINDPLANE_PROFILING_SERVICE_NAME"
	bindplaneProfilingNoCPUEnvVar       = "BINDPLANE_PROFILING_NO_CPU"
	bindplaneProfilingNoAllocEnvVar     = "BINDPLANE_PROFILING_NO_ALLOC"
	bindplaneProfilingNoHeapEnvVar      = "BINDPLANE_PROFILING_NO_HEAP"
	bindplaneProfilingNoGoroutineEnvVar = "BINDPLANE_PROFILING_NO_GOROUTINE"
	bindplaneProfilingMutexEnvVar       = "BINDPLANE_PROFILING_MUTEX"

	// Pprof
	bindplanePprofEnabledEnvVar  = "BINDPLANE_PPROF_ENABLED"
	bindplanePprofEndpointEnvVar = "BINDPLANE_PPROF_ENDPOINT"

	// Status check endpoints
	bindplaneStatusEnabledEnvVar = "BINDPLANE_STATUS_ENABLED"
	bindplaneStatusKeysEnvVar    = "BINDPLANE_STATUS_KEYS"
)

const (
	// defaultPprofEndpoint is the default host:port for the pprof server (matches Bindplane)
	defaultPprofEndpoint = "127.0.0.1:6060"
)

// Common security and pod constants
const (
	// defaultRunAsUser is the default user ID for security contexts
	defaultRunAsUser = int64(65534)
	// defaultRunAsGroup is the default group ID for security contexts
	defaultRunAsGroup = int64(65534)
	// defaultTerminationGracePeriodSeconds is the default termination grace period
	defaultTerminationGracePeriodSeconds = int64(60)
	// metadataNameFieldPath is the field path for pod metadata.name
	metadataNameFieldPath = "metadata.name"
	// preStopCommand is the command used in preStop lifecycle hooks
	preStopCommand = "sh"
	// preStopArgs is the arguments for preStop lifecycle hooks
	preStopArgs = "-c"
	// preStopSleep is the sleep command for preStop hooks
	preStopSleep = "sleep 5"
)

// Probe timing constants
const (
	// Startup probe: allow up to 100s (20 × 5s) for the container to become ready.
	probeStartupInitialDelaySeconds int32 = 0
	probeStartupPeriodSeconds       int32 = 5
	probeStartupFailureThreshold    int32 = 20
	probeStartupSuccessThreshold    int32 = 1
	probeStartupTimeoutSeconds      int32 = 1

	// Liveness and readiness probes rely on the startup probe for the initial window,
	// so InitialDelaySeconds is omitted (defaults to 0).
	probePeriodSeconds    int32 = 10
	probeFailureThreshold int32 = 3
	probeSuccessThreshold int32 = 1
	probeTimeoutSeconds   int32 = 5
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
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers;clusterissuers,verbs=get;list;watch

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

	// Validate that the Bindplane name produces valid Kubernetes resource names (DNS-1035).
	// This avoids repeated reconciler errors when the name would create invalid Service/Resource names.
	if err := validateBindplaneName(bindplane.Name); err != nil {
		log.Error(err, "invalid Bindplane name: resource names must be DNS-1035 compliant")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidName",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}
	if err := validateLicenseConfig(&bindplane.Spec.Config); err != nil {
		log.Error(err, "invalid Bindplane config: license must be set via license or licenseSecretRef")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidConfig",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}
	if err := validateProfilingConfig(&bindplane.Spec.Config); err != nil {
		log.Error(err, "invalid Bindplane config: profiling")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidConfig",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}
	if err := validatePprofConfig(&bindplane.Spec.Config); err != nil {
		log.Error(err, "invalid Bindplane config: pprof")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidConfig",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}
	if err := validateStatusConfig(&bindplane.Spec.Config); err != nil {
		log.Error(err, "invalid Bindplane config: status")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidConfig",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// Reconcile internal TLS certificates (cert-manager) before workloads that mount them.
	if err := r.reconcileInternalTLSCertificates(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile internal TLS certificates")
		return ctrl.Result{}, err
	}

	// Reconcile Transform Agent resources
	if err := r.reconcileTransformAgent(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile Transform Agent")
		return ctrl.Result{}, err
	}

	// Reconcile Prometheus resources
	if err := r.reconcileTSDB(ctx, bindplane, log); err != nil {
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

	// Mark as reconciled so any previous InvalidName condition is cleared
	condition := metav1.Condition{
		Type:               "Reconciled",
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "All resources reconciled successfully",
		ObservedGeneration: bindplane.Generation,
		LastTransitionTime: metav1.Now(),
	}
	meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
	if err := r.Status().Update(ctx, bindplane); err != nil {
		log.Error(err, "failed to update Bindplane status")
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

// maxResourceNamePrefixLen is the maximum length for the Bindplane name prefix so that
// derived names (e.g. "<name>-transform-agent") stay within the DNS-1035 label limit of 63 characters.
const maxResourceNamePrefixLen = 63 - 1 - len("transform-agent") // 47

// validateBindplaneName returns an error if the Bindplane name would produce invalid
// Kubernetes resource names. Resource names must be DNS-1035 compliant: start with a
// lowercase letter, contain only lowercase letters, digits, or hyphens, and end with
// a letter or digit (e.g. "my-name", "abc-123").
func validateBindplaneName(name string) error {
	if name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if len(name) > maxResourceNamePrefixLen {
		return fmt.Errorf("name %q is too long: must be at most %d characters so that derived resource names (e.g. <name>-transform-agent) stay within the 63-character limit", name, maxResourceNamePrefixLen)
	}
	// Must start with a lowercase letter [a-z]
	if r := rune(name[0]); !unicode.IsLetter(r) || !unicode.IsLower(r) {
		return fmt.Errorf("name %q must start with a lowercase letter (a-z); Kubernetes resource names are DNS-1035 labels", name)
	}
	// Must end with a letter or digit [a-z0-9]
	last := name[len(name)-1]
	if last == '-' || ((last < 'a' || last > 'z') && (last < '0' || last > '9')) {
		return fmt.Errorf("name %q must end with a lowercase letter or digit (a-z, 0-9)", name)
	}
	// All characters must be [a-z0-9-]
	for i, c := range name {
		if c != '-' && !unicode.IsLower(c) && !unicode.IsDigit(c) {
			return fmt.Errorf("name %q contains invalid character %q at position %d: only lowercase letters (a-z), digits (0-9), and hyphens are allowed", name, string(c), i)
		}
	}
	return nil
}

// validateLicenseConfig ensures exactly one license source is configured.
func validateLicenseConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil {
		return fmt.Errorf("spec.config is required")
	}
	hasLicense := config.License != ""
	hasLicenseSecretRef := config.LicenseSecretRef != nil
	if hasLicense == hasLicenseSecretRef {
		return fmt.Errorf("exactly one of spec.config.license or spec.config.licenseSecretRef must be set")
	}
	return nil
}

// validateProfilingConfig ensures projectID is set when profiling is enabled.
func validateProfilingConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Profiling == nil || !config.Profiling.Enabled {
		return nil
	}
	if config.Profiling.ProjectID == "" {
		return fmt.Errorf("projectID is required when profiling is enabled")
	}
	return nil
}

// uuidRegex matches standard UUID format (case-insensitive).
var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// validateStatusConfig ensures status check keys are valid UUIDs when set inline.
func validateStatusConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Status == nil {
		return nil
	}
	s := config.Status
	if s.Enabled && len(s.Keys) == 0 && s.KeysSecretRef == nil {
		return fmt.Errorf("at least one key must be configured when status is enabled")
	}
	for i, key := range s.Keys {
		if !uuidRegex.MatchString(key) {
			return fmt.Errorf("spec.config.status.keys[%d]: %q is not a valid UUID", i, key)
		}
	}
	return nil
}

// validatePprofConfig ensures pprof endpoint is a valid host:port when set.
func validatePprofConfig(config *bindplanev1alpha1.BindplaneConfigSpec) error {
	if config == nil || config.Pprof == nil || !config.Pprof.Enabled || config.Pprof.Endpoint == "" {
		return nil
	}
	if _, _, err := net.SplitHostPort(config.Pprof.Endpoint); err != nil {
		return fmt.Errorf("invalid pprof endpoint %q: %w", config.Pprof.Endpoint, err)
	}
	return nil
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
		runAsUser: new(int64(65534)), // Default to nobody user
	}

	// Apply all option functions
	for _, opt := range opts {
		opt(options)
	}

	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: new(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		ReadOnlyRootFilesystem: new(true),
		RunAsNonRoot:           new(true),
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

// getLDAPTLSVolumeAndMount returns a Secret volume and mount for LDAP TLS when config.Auth.LDAP.TLS is set.
// The Secret is mounted at ldapTLSMountPath; TLS env vars are set to the computed file paths (mountPath/key).
// Returns (nil, nil) when LDAP TLS is not configured.
func getLDAPTLSVolumeAndMount(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	tls := getLDAPTLSConfig(bindplane)
	if tls == nil || tls.SecretName == "" {
		return nil, nil
	}
	vol := corev1.Volume{
		Name: ldapTLSVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tls.SecretName,
			},
		},
	}
	mount := corev1.VolumeMount{
		Name:      ldapTLSVolumeName,
		MountPath: ldapTLSMountPath,
		ReadOnly:  true,
	}
	return []corev1.Volume{vol}, []corev1.VolumeMount{mount}
}

// getLDAPTLSConfig returns the LDAP TLS config when present.
func getLDAPTLSConfig(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.LDAPTLSConfig {
	if bindplane.Spec.Config.Auth == nil || bindplane.Spec.Config.Auth.LDAP == nil {
		return nil
	}
	return bindplane.Spec.Config.Auth.LDAP.TLS
}

// getNetworkTLSVolumeAndMount returns a Secret volume and mount for network TLS when config.Network.TLS is set
// with secretName and both certKey and keyKey (server or mutual TLS). The Secret is mounted at networkTLSMountPath;
// TLS env vars are set to the computed file paths (mountPath/key). Returns (nil, nil) when network TLS is not configured.
func getNetworkTLSVolumeAndMount(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	tls := getNetworkTLSConfig(bindplane)
	if tls == nil || tls.SecretName == "" || tls.CertKey == "" || tls.KeyKey == "" {
		return nil, nil
	}
	vol := corev1.Volume{
		Name: networkTLSVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tls.SecretName,
			},
		},
	}
	mount := corev1.VolumeMount{
		Name:      networkTLSVolumeName,
		MountPath: networkTLSMountPath,
		ReadOnly:  true,
	}
	return []corev1.Volume{vol}, []corev1.VolumeMount{mount}
}

// getNetworkTLSConfig returns the network TLS config when present.
func getNetworkTLSConfig(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.NetworkTLSConfig {
	if bindplane.Spec.Config.Network == nil {
		return nil
	}
	return bindplane.Spec.Config.Network.TLS
}

// getPostgresTLSVolumeAndMount returns a Secret volume and mount for Postgres TLS when config.Store.Postgres.TLS is set
// with secretName and caKey (server-side TLS) or with caKey, certKey, and keyKey (mutual TLS). Returns (nil, nil) when not configured.
func getPostgresTLSVolumeAndMount(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	tls := getPostgresTLSConfig(bindplane)
	if tls == nil || tls.SecretName == "" || tls.CAKey == "" {
		return nil, nil
	}
	vol := corev1.Volume{
		Name: postgresTLSVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: tls.SecretName,
			},
		},
	}
	mount := corev1.VolumeMount{
		Name:      postgresTLSVolumeName,
		MountPath: postgresTLSMountPath,
		ReadOnly:  true,
	}
	return []corev1.Volume{vol}, []corev1.VolumeMount{mount}
}

// getPostgresTLSConfig returns the Postgres TLS config when present.
func getPostgresTLSConfig(bindplane *bindplanev1alpha1.Bindplane) *bindplanev1alpha1.PostgresTLSConfig {
	if bindplane.Spec.Config.Store.Postgres == nil {
		return nil
	}
	return bindplane.Spec.Config.Store.Postgres.TLS
}

// getConfigTLSVolumesAndMounts returns combined volumes and volume mounts for LDAP TLS, network TLS, Postgres TLS,
// and internal TLS (cert-manager: Prometheus remote write client cert, NATS TLS).
// Used by Node, Jobs, Jobs Migrate, and NATS so they receive all config TLS secrets when configured.
func getConfigTLSVolumesAndMounts(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	ldapVols, ldapMounts := getLDAPTLSVolumeAndMount(bindplane)
	netVols, netMounts := getNetworkTLSVolumeAndMount(bindplane)
	pgVols, pgMounts := getPostgresTLSVolumeAndMount(bindplane)
	internalVols, internalMounts := getInternalTLSVolumesAndMounts(bindplane)
	natsVols, natsMounts := getNatsTLSVolumesAndMounts(bindplane)
	vols := append(append(append(append(ldapVols, netVols...), pgVols...), internalVols...), natsVols...)
	mounts := append(append(append(append(ldapMounts, netMounts...), pgMounts...), internalMounts...), natsMounts...)
	return vols, mounts
}

// getInternalTLSVolumesAndMounts returns volumes and mounts for Prometheus remote write client TLS (config.prometheus.tls).
// Uses operator-created client cert secret when certManager is set, or user secret when secretName is set.
func getInternalTLSVolumesAndMounts(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	if !isTSDBClientTLSEnabled(bindplane) {
		return nil, nil
	}
	tls := bindplane.Spec.Config.TSDB.TLS
	var vol corev1.Volume
	if tls.CertManager != nil && tls.CertManager.Name != "" {
		vol = corev1.Volume{
			Name: internalTLSTSDBClientVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{SecretName: getResourceName(bindplane, tsdbRemoteWriteClientCertSuffix)},
			},
		}
	} else {
		certKey, keyKey, caKey := tls.CertKey, tls.KeyKey, tls.CAKey
		if certKey == "" {
			certKey = "tls.crt"
		}
		if keyKey == "" {
			keyKey = "tls.key"
		}
		if caKey == "" {
			caKey = "ca.crt"
		}
		vol = corev1.Volume{
			Name: internalTLSTSDBClientVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tls.SecretName,
					Items: []corev1.KeyToPath{
						{Key: certKey, Path: "tls.crt"},
						{Key: keyKey, Path: "tls.key"},
						{Key: caKey, Path: "ca.crt"},
					},
				},
			},
		}
	}
	mount := corev1.VolumeMount{
		Name:      internalTLSTSDBClientVolumeName,
		MountPath: internalTLSTSDBClientMountPath,
		ReadOnly:  true,
	}
	return []corev1.Volume{vol}, []corev1.VolumeMount{mount}
}

// getNatsTLSVolumesAndMounts returns volumes and mounts for NATS TLS when cert-manager is used (spec.config.nats.tls.certManager).
func getNatsTLSVolumesAndMounts(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	if !isNatsCertManagerTLSEnabled(bindplane) {
		return nil, nil
	}
	vol := corev1.Volume{
		Name: internalTLSNatsVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: getResourceName(bindplane, natsTLSCertSuffix)},
		},
	}
	mount := corev1.VolumeMount{
		Name:      internalTLSNatsVolumeName,
		MountPath: internalTLSNatsMountPath,
		ReadOnly:  true,
	}
	return []corev1.Volume{vol}, []corev1.VolumeMount{mount}
}

// mergePodTemplateSpec merges user-provided pod template spec with operator-managed fields.
// It supports ANY arbitrary field in the pod spec, while protecting only critical operator-managed fields.
// Protected fields: ServiceAccountName, container names/images/ports/env/command/args, protected labels, TerminationGracePeriodSeconds.
// Resource-aware Go runtime env vars are applied after the final effective container Resources are known.
func mergePodTemplateSpec(operatorManaged corev1.PodTemplateSpec, userProvided *bindplanev1alpha1.PodTemplateSpec) corev1.PodTemplateSpec {
	if userProvided == nil {
		merged := operatorManaged.DeepCopy()
		for i := range merged.Spec.Containers {
			applyGoRuntimeEnvVars(&merged.Spec.Containers[i])
		}
		return *merged
	}

	// Deep copy operator-managed spec as the base
	merged := operatorManaged.DeepCopy()

	// Save protected fields before merging
	protectedServiceAccountName := merged.Spec.ServiceAccountName
	protectedTerminationGracePeriodSeconds := copyInt64Ptr(merged.Spec.TerminationGracePeriodSeconds)
	protectedContainers := deepCopyContainers(merged.Spec.Containers)
	protectedVolumes := deepCopyVolumes(merged.Spec.Volumes)

	protectedLabelKeys := []string{labelKeyName, labelKeyInstance, labelKeyComponent}

	mergeTemplateMetadata(merged, userProvided, protectedLabelKeys)

	// Merge all pod spec fields via JSON deep-merge; fall back to operator spec on any error.
	if err := mergeSpecViaJSON(&merged.Spec, userProvided.Spec.DeepCopy()); err != nil {
		return *merged
	}

	// Restore protected fields overwritten by the JSON merge
	merged.Spec.ServiceAccountName = protectedServiceAccountName
	merged.Spec.TerminationGracePeriodSeconds = protectedTerminationGracePeriodSeconds
	merged.Spec.Containers = mergeContainers(protectedContainers, userProvided.Spec.Containers)
	merged.Spec.Volumes = mergeVolumes(protectedVolumes, userProvided.Spec.Volumes)
	for i := range merged.Spec.Containers {
		applyGoRuntimeEnvVars(&merged.Spec.Containers[i])
	}

	return *merged
}

// copyInt64Ptr returns a copy of a *int64 to avoid aliasing issues across JSON round-trips.
func copyInt64Ptr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

// deepCopyContainers returns a deep copy of a container slice.
func deepCopyContainers(src []corev1.Container) []corev1.Container {
	out := make([]corev1.Container, len(src))
	for i, c := range src {
		out[i] = *c.DeepCopy()
	}
	return out
}

// deepCopyVolumes returns a deep copy of a volume slice.
func deepCopyVolumes(src []corev1.Volume) []corev1.Volume {
	out := make([]corev1.Volume, len(src))
	for i, v := range src {
		out[i] = *v.DeepCopy()
	}
	return out
}

// mergeTemplateMetadata applies user-provided labels and annotations onto merged,
// skipping any label keys in protectedLabelKeys.
func mergeTemplateMetadata(merged *corev1.PodTemplateSpec, userProvided *bindplanev1alpha1.PodTemplateSpec, protectedLabelKeys []string) {
	if userProvided.Labels != nil {
		if merged.Labels == nil {
			merged.Labels = make(map[string]string)
		}
		for k, v := range userProvided.Labels {
			if !slices.Contains(protectedLabelKeys, k) {
				merged.Labels[k] = v
			}
		}
	}
	if userProvided.Annotations != nil {
		if merged.Annotations == nil {
			merged.Annotations = make(map[string]string)
		}
		maps.Copy(merged.Annotations, userProvided.Annotations)
	}
}

// mergeSpecViaJSON deep-merges userSpec into dst using JSON marshal/unmarshal.
func mergeSpecViaJSON(dst *corev1.PodSpec, userSpec *corev1.PodSpec) error {
	dstJSON, err := json.Marshal(dst)
	if err != nil {
		return err
	}
	userJSON, err := json.Marshal(userSpec)
	if err != nil {
		return err
	}

	var dstMap, userMap map[string]any
	if err := json.Unmarshal(dstJSON, &dstMap); err != nil {
		return err
	}
	if err := json.Unmarshal(userJSON, &userMap); err != nil {
		return err
	}

	mergeMaps(dstMap, userMap)

	merged, err := json.Marshal(dstMap)
	if err != nil {
		return err
	}
	return json.Unmarshal(merged, dst)
}

// mergeContainers merges user-provided containers into the operator-managed set by name.
// Protected fields (Name, Image, Ports, Env, Command, Args) are always restored from the operator container.
func mergeContainers(protected []corev1.Container, userContainers []corev1.Container) []corev1.Container {
	if len(userContainers) == 0 {
		return protected
	}

	containerMap := make(map[string]corev1.Container, len(protected))
	for _, c := range protected {
		containerMap[c.Name] = c
	}

	for _, userContainer := range userContainers {
		if operatorContainer, exists := containerMap[userContainer.Name]; exists {
			containerMap[userContainer.Name] = mergeSingleContainer(operatorContainer, userContainer)
		}
	}

	result := make([]corev1.Container, len(protected))
	for i, c := range protected {
		if updated, exists := containerMap[c.Name]; exists {
			result[i] = updated
		} else {
			result[i] = c
		}
	}
	return result
}

// mergeSingleContainer merges userContainer fields into operatorContainer via JSON,
// then restores protected fields.
func mergeSingleContainer(operatorContainer, userContainer corev1.Container) corev1.Container {
	mergedContainer := operatorContainer.DeepCopy()

	operatorJSON, _ := json.Marshal(mergedContainer)
	userJSON, _ := json.Marshal(userContainer)

	var operatorMap, userMap map[string]any
	if err := json.Unmarshal(operatorJSON, &operatorMap); err == nil {
		if err := json.Unmarshal(userJSON, &userMap); err == nil {
			mergeMaps(operatorMap, userMap)
			if mergedJSON, err := json.Marshal(operatorMap); err == nil {
				if err := json.Unmarshal(mergedJSON, mergedContainer); err != nil {
					mergedContainer = operatorContainer.DeepCopy()
				}
			}
		}
	}

	// Restore protected container fields
	mergedContainer.Name = operatorContainer.Name
	mergedContainer.Image = operatorContainer.Image
	mergedContainer.Ports = operatorContainer.Ports
	mergedContainer.Env = operatorContainer.Env
	mergedContainer.Command = operatorContainer.Command
	mergedContainer.Args = operatorContainer.Args

	return *mergedContainer
}

// mergeVolumes merges user-provided volumes into the operator-managed set by name.
// Operator volumes are preserved; user volumes may add new entries or override existing ones.
func mergeVolumes(protected []corev1.Volume, userVolumes []corev1.Volume) []corev1.Volume {
	if len(userVolumes) == 0 {
		return protected
	}

	volumeMap := make(map[string]corev1.Volume, len(protected))
	volumeOrder := make([]string, 0, len(protected))
	for _, v := range protected {
		volumeMap[v.Name] = v
		volumeOrder = append(volumeOrder, v.Name)
	}
	for _, userVol := range userVolumes {
		if _, exists := volumeMap[userVol.Name]; !exists {
			volumeOrder = append(volumeOrder, userVol.Name)
		}
		volumeMap[userVol.Name] = userVol
	}

	result := make([]corev1.Volume, 0, len(volumeOrder))
	for _, name := range volumeOrder {
		result = append(result, volumeMap[name])
	}
	return result
}

// mergeMaps recursively merges map b into map a
func mergeMaps(a, b map[string]any) {
	for k, v := range b {
		if v == nil {
			continue
		}
		if av, exists := a[k]; exists {
			if avMap, ok := av.(map[string]any); ok {
				if bvMap, ok := v.(map[string]any); ok {
					mergeMaps(avMap, bvMap)
					continue
				}
			}
		}
		a[k] = v
	}
}
