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
	"slices"
	"strings"
	"time"

	rolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
	"github.com/observiq/bindplane-operator/internal/validation"
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

	// Auth session and API key
	bindplaneSessionSecretEnvVar = "BINDPLANE_SESSION_SECRET" // #nosec G101 -- env var name, not a credential
	bindplaneSecretKeyEnvVar     = "BINDPLANE_SECRET_KEY"     // #nosec G101 -- env var name, not a credential

	// Store configuration
	bindplaneStoreTypeEnvVar                       = "BINDPLANE_STORE_TYPE"
	bindplaneStoreMaxEventsEnvVar                  = "BINDPLANE_STORE_MAX_EVENTS"
	bindplaneStoreEventMergeWindowEnvVar           = "BINDPLANE_STORE_EVENT_MERGE_WINDOW"
	bindplaneStoreSummaryRollupRetentionDaysEnvVar = "BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS"

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

	// Internal TLS (cert-manager) volume mount for the Transform Agent
	internalTLSTransformAgentVolumeName = "transform-agent-tls"
	internalTLSTransformAgentMountPath  = "/etc/bindplane/transform-agent-tls"

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
	bindplaneTSDBAuthTypeEnvVar            = "BINDPLANE_PROMETHEUS_AUTH_TYPE"

	// Prometheus remote write TLS (cert-manager internal mTLS)
	bindplaneTSDBEnableTLSEnvVar     = "BINDPLANE_PROMETHEUS_ENABLE_TLS"
	bindplaneTSDBTLSCertEnvVar       = "BINDPLANE_PROMETHEUS_TLS_CERT"
	bindplaneTSDBTLSKeyEnvVar        = "BINDPLANE_PROMETHEUS_TLS_KEY"
	bindplaneTSDBTLSCAEnvVar         = "BINDPLANE_PROMETHEUS_TLS_CA"
	bindplaneTSDBTLSSkipVerifyEnvVar = "BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY"

	// Transform Agent configuration
	bindplaneTransformAgentEnableRemoteEnvVar = "BINDPLANE_TRANSFORM_AGENT_ENABLE_REMOTE"
	bindplaneTransformAgentRemoteAgentsEnvVar = "BINDPLANE_TRANSFORM_AGENT_REMOTE_AGENTS"
	bindplaneTransformAgentTLSCertEnvVar      = "BINDPLANE_TRANSFORM_AGENT_TLS_CERT"
	bindplaneTransformAgentTLSKeyEnvVar       = "BINDPLANE_TRANSFORM_AGENT_TLS_KEY"
	bindplaneTransformAgentTLSCAEnvVar        = "BINDPLANE_TRANSFORM_AGENT_TLS_CA"

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

	// Logging configuration
	bindplaneLoggingLevelEnvVar = "BINDPLANE_LOGGING_LEVEL"
	bindplaneLoggingTypeEnvVar  = "BINDPLANE_LOGGING_TYPE"

	// Status check endpoint configuration
	bindplaneStatusEnabledEnvVar = "BINDPLANE_STATUS_ENABLED"
	bindplaneStatusKeysEnvVar    = "BINDPLANE_STATUS_KEYS"

	// Advanced server (OpAMP shutdown tuning — used by the OpAMP component)
	bindplaneAdvancedServerOpAMPShutdownGracePeriodTargetEnvVar = "BINDPLANE_ADVANCED_SERVER_OPAMP_SHUTDOWN_GRACE_PERIOD_TARGET"

	// Agents configuration
	bindplaneAgentsAuthTypeEnvVar                   = "BINDPLANE_AGENTS_AUTH_TYPE"
	bindplaneAgentsAuthSecretKeyHeadersEnvVar       = "BINDPLANE_AGENTS_AUTH_SECRET_KEY_HEADERS"
	bindplaneAgentsHeartbeatIntervalEnvVar          = "BINDPLANE_AGENTS_HEARTBEAT_INTERVAL"
	bindplaneAgentsHeartbeatTTLEnvVar               = "BINDPLANE_AGENTS_HEARTBEAT_TTL"
	bindplaneAgentsHeartbeatExpiryIntervalEnvVar    = "BINDPLANE_AGENTS_HEARTBEAT_EXPIRY_INTERVAL"
	bindplaneAgentsRebalanceIntervalEnvVar          = "BINDPLANE_AGENTS_REBALANCE_INTERVAL"
	bindplaneAgentsRebalancePercentageEnvVar        = "BINDPLANE_AGENTS_REBALANCE_PERCENTAGE"
	bindplaneAgentsRebalanceJitterEnvVar            = "BINDPLANE_AGENTS_REBALANCE_JITTER"
	bindplaneAgentsMaxSimultaneousConnectionsEnvVar = "BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS"

	// AgentVersions configuration
	bindplaneAgentVersionsSyncIntervalEnvVar = "BINDPLANE_AGENT_VERSIONS_SYNC_INTERVAL"

	// OIDC additional configuration
	bindplaneOIDCDisableInvitationsEnvVar = "BINDPLANE_OIDC_DISABLE_INVITATIONS"

	// NATS TLS additional configuration
	bindplaneNatsTLSSkipVerifyEnvVar = "BINDPLANE_NATS_TLS_SKIP_VERIFY"
)

const (
	// defaultConcurrency is the default value for maxConcurrency and maxSimultaneousConnections.
	defaultConcurrency = 10

	// annotationPauseReconciliation is the annotation key used to pause operator reconciliation
	// for a specific Bindplane CR. Set the value to "true" to pause.
	// Example: kubectl annotate bindplane my-bindplane k8s.bindplane.com/pause-reconciliation=true
	annotationPauseReconciliation = "k8s.bindplane.com/pause-reconciliation"

	// annotationValueTrue is the string value that enables a boolean operator annotation
	// (e.g. pause-reconciliation, force-migrate).
	annotationValueTrue = "true"

	// annotationPreserveSelectorKeys is an internal operator annotation placed on Service objects
	// to signal which selector keys should be preserved from the live Service (e.g.
	// rollouts-pod-template-hash written by Argo Rollouts during a BlueGreen cutover).
	annotationPreserveSelectorKeys = "internal.k8s.bindplane.com/preserve-selector-keys"

	// bindplaneFinalizer is the finalizer added to Bindplane CRs to ensure the operator
	// can perform cleanup before the CR is removed from etcd.
	bindplaneFinalizer = "k8s.bindplane.com/finalizer"
)

// resolveImage returns override when non-empty, otherwise falls back to defaultRef.
func resolveImage(override, defaultRef string) string {
	if override != "" {
		return override
	}
	return defaultRef
}

func defaultBindplaneEEImage(bindplane *bindplanev1alpha1.Bindplane) string {
	return "ghcr.io/observiq/bindplane-ee:" + bindplane.Spec.Version
}

func defaultTransformAgentImage(bindplane *bindplanev1alpha1.Bindplane) string {
	return "ghcr.io/observiq/bindplane-transform-agent:" + bindplane.Spec.Version + "-bindplane"
}

func defaultTSDBImage(bindplane *bindplanev1alpha1.Bindplane) string {
	return "ghcr.io/observiq/bindplane-prometheus:" + bindplane.Spec.Version
}

// getNodeImage returns the container image for Bindplane Node.
func getNodeImage(bindplane *bindplanev1alpha1.Bindplane) string {
	return resolveImage(bindplane.Spec.Bindplane.Image, defaultBindplaneEEImage(bindplane))
}

// getOpAMPImage returns the container image for the OpAMP deployment.
func getOpAMPImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.OpAMP == nil {
		return defaultBindplaneEEImage(bindplane)
	}
	return resolveImage(bindplane.Spec.OpAMP.Image, defaultBindplaneEEImage(bindplane))
}

// getNatsImage returns the container image for NATS.
func getNatsImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.Nats == nil {
		return defaultBindplaneEEImage(bindplane)
	}
	return resolveImage(bindplane.Spec.Nats.Image, defaultBindplaneEEImage(bindplane))
}

// getBindplaneJobsImage returns the container image for Bindplane Jobs.
func getBindplaneJobsImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.BindplaneJobs == nil {
		return defaultBindplaneEEImage(bindplane)
	}
	return resolveImage(bindplane.Spec.BindplaneJobs.Image, defaultBindplaneEEImage(bindplane))
}

// getBindplaneJobsMigrateImage returns the container image for Bindplane Jobs Migrate.
func getBindplaneJobsMigrateImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.BindplaneJobsMigrate == nil {
		return defaultBindplaneEEImage(bindplane)
	}
	return resolveImage(bindplane.Spec.BindplaneJobsMigrate.Image, defaultBindplaneEEImage(bindplane))
}

// getTransformAgentImage returns the container image for Transform Agent.
func getTransformAgentImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.TransformAgent == nil {
		return defaultTransformAgentImage(bindplane)
	}
	return resolveImage(bindplane.Spec.TransformAgent.Image, defaultTransformAgentImage(bindplane))
}

// getTSDBImage returns the container image for TSDB (Prometheus).
func getTSDBImage(bindplane *bindplanev1alpha1.Bindplane) string {
	if bindplane.Spec.TSDB == nil {
		return defaultTSDBImage(bindplane)
	}
	return resolveImage(bindplane.Spec.TSDB.Image, defaultTSDBImage(bindplane))
}

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

// Argo Rollouts constants
const (
	// rolloutsHashLabel is the selector key written by Argo Rollouts to the active Service
	// during BlueGreen promotions. The operator must preserve this key so it does not flap it.
	rolloutsHashLabel = rolloutsv1alpha1.DefaultRolloutUniqueLabelKey
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
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers;clusterissuers,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=argoproj.io,resources=rollouts,verbs=get;list;watch;create;update;patch;delete

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

	// Set phase to Pending on the first reconcile (before any resources exist).
	// Subsequent reconcile paths will overwrite this with the appropriate phase.
	if bindplane.Status.Phase == "" {
		bindplane.Status.Phase = "Pending"
	}

	// Handle deletion: if the object is being deleted and has our finalizer, run cleanup.
	if !bindplane.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(bindplane, bindplaneFinalizer) {
			if err := r.handleDeletion(ctx, bindplane, log); err != nil {
				log.Error(err, "failed to handle deletion")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(bindplane, bindplaneFinalizer) {
		controllerutil.AddFinalizer(bindplane, bindplaneFinalizer)
		if err := r.Update(ctx, bindplane); err != nil {
			log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Check for pause annotation — if set to "true", skip reconciliation entirely.
	if bindplane.Annotations[annotationPauseReconciliation] == annotationValueTrue {
		log.Info("Reconciliation paused via annotation; skipping", "annotation", annotationPauseReconciliation)
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "Paused",
			Message:            fmt.Sprintf("Reconciliation paused. Remove annotation %s or set to false to resume.", annotationPauseReconciliation),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		bindplane.Status.ObservedGeneration = bindplane.Generation
		if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
			log.Error(statusErr, "failed to update Bindplane status for pause")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// Validate the Bindplane resource. The webhook enforces this at admission time; this
	// block is a safety net for clusters where the webhook is disabled or bypassed.
	if err := validation.ValidateBindplane(bindplane); err != nil {
		log.Error(err, "invalid Bindplane resource")
		condition := metav1.Condition{
			Type:               "Reconciled",
			Status:             metav1.ConditionFalse,
			Reason:             "Invalid",
			Message:            err.Error(),
			ObservedGeneration: bindplane.Generation,
			LastTransitionTime: metav1.Now(),
		}
		meta.SetStatusCondition(&bindplane.Status.Conditions, condition)
		bindplane.Status.Phase = "Degraded"
		bindplane.Status.ObservedGeneration = bindplane.Generation
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

	// Reconcile session secret before workloads that consume it via BINDPLANE_SESSION_SECRET.
	if err := r.reconcileSessionSecret(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile session secret")
		return ctrl.Result{}, err
	}

	// Reconcile status secret before workloads that consume it via BINDPLANE_STATUS_KEYS.
	if err := r.reconcileStatusSecret(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile status secret")
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

	// Reconcile the Jobs Migrate batch/v1 Job; block downstream workloads until it completes.
	state, err := r.reconcileMigrateJob(ctx, bindplane, log)
	if err != nil {
		log.Error(err, "unable to reconcile Jobs Migrate Job")
		return ctrl.Result{}, err
	}
	switch state {
	case migrateFailed:
		// The MigrationFailed condition and Degraded phase are set inside
		// reconcileMigrateJob. Do not requeue: a terminal Failed Job will not change
		// state on its own, so requeueing would only spam logs. A spec change or the
		// k8s.bindplane.com/force-migrate annotation triggers a fresh reconcile via
		// the CR watch.
		log.Info("Jobs Migrate Job failed; halting rollout. Inspect the failed Job's pods for logs, "+
			"then set the k8s.bindplane.com/force-migrate annotation to retry",
			"job", getResourceName(bindplane, bindplaneJobsMigrateComponent))
		return ctrl.Result{}, nil
	case migrateInProgress:
		log.Info("waiting for Jobs Migrate Job to complete")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	case migrateComplete:
		// Migration is complete for the desired image; proceed to downstream workloads.
	}

	// Reconcile Bindplane Jobs resources
	if err := r.reconcileBindplaneJobsRegular(ctx, bindplane, log); err != nil {
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

	// Reconcile OpAMP resources (optional dedicated OpAMP deployment)
	if err := r.reconcileOpAMP(ctx, bindplane, log); err != nil {
		log.Error(err, "unable to reconcile OpAMP")
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

	// Populate per-component ready replica counts and overall phase.
	r.updateReadyReplicaStatus(ctx, bindplane)
	// Populate resolved images for each deployed component.
	updateImageStatus(bindplane)
	bindplane.Status.ObservedGeneration = bindplane.Generation

	if err := r.Status().Update(ctx, bindplane); err != nil {
		log.Error(err, "failed to update Bindplane status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// updateReadyReplicaStatus queries the owned workloads for their current ready
// replica counts and sets the Phase field accordingly.
func (r *BindplaneReconciler) updateReadyReplicaStatus(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane) {
	nodeReady := r.deploymentReadyReplicas(ctx, getResourceName(bindplane, nodeComponent), bindplane.Namespace)
	natsReady := r.statefulSetReadyReplicas(ctx, getResourceName(bindplane, natsComponent), bindplane.Namespace)
	taReady := r.deploymentReadyReplicas(ctx, getResourceName(bindplane, transformAgentComponent), bindplane.Namespace)
	jobsReady := r.deploymentReadyReplicas(ctx, getResourceName(bindplane, bindplaneJobsComponent), bindplane.Namespace)

	bindplane.Status.Components.Bindplane.ReadyReplicas = nodeReady
	bindplane.Status.Components.Nats.ReadyReplicas = natsReady
	bindplane.Status.Components.TransformAgent.ReadyReplicas = taReady
	bindplane.Status.Components.Jobs.ReadyReplicas = jobsReady

	// Only report TSDB readiness when local TSDB is deployed (not when remote TSDB is enabled).
	var tsdbReady int32
	if !isTSDBRemoteEnabled(bindplane) {
		tsdbReady = r.statefulSetReadyReplicas(ctx, getResourceName(bindplane, tsdbComponent), bindplane.Namespace)
	}
	bindplane.Status.Components.TSDB.ReadyReplicas = tsdbReady

	// OpAMP is an optional dedicated deployment; track its readiness separately.
	var opampReady int32
	if bindplane.Spec.OpAMP != nil && bindplane.Spec.OpAMP.Enabled {
		opampReady = r.deploymentReadyReplicas(ctx, getResourceName(bindplane, opampComponent), bindplane.Namespace)
	}
	bindplane.Status.Components.OpAMP.ReadyReplicas = opampReady

	nodeDesired := *bindplane.Spec.Bindplane.Replicas
	var natsDesired int32
	if bindplane.Spec.Nats != nil && bindplane.Spec.Nats.Replicas != nil {
		natsDesired = *bindplane.Spec.Nats.Replicas
	}
	var taDesired int32
	if bindplane.Spec.TransformAgent != nil && bindplane.Spec.TransformAgent.Replicas != nil {
		taDesired = *bindplane.Spec.TransformAgent.Replicas
	}
	// TSDB: desired is 1 replica when local, 0 when remote (already satisfied — remote means no local SS).
	var tsdbDesired int32
	if !isTSDBRemoteEnabled(bindplane) {
		tsdbDesired = 1
	}

	if r.allWorkloadsReady(nodeReady, nodeDesired, natsReady, natsDesired, taReady, taDesired, jobsReady, tsdbReady, tsdbDesired) {
		bindplane.Status.Phase = "Ready"
	} else {
		bindplane.Status.Phase = "ApplyingChanges"
	}
}

// updateImageStatus records the resolved container image for each deployed component into
// status.images. JobsMigrate is intentionally omitted — it is managed by the migration gate
// in jobs.go and only reflects images for which migration has completed.
func updateImageStatus(bindplane *bindplanev1alpha1.Bindplane) {
	bindplane.Status.Components.Bindplane.Image = getNodeImage(bindplane)
	bindplane.Status.Components.Jobs.Image = getBindplaneJobsImage(bindplane)
	bindplane.Status.Components.Nats.Image = getNatsImage(bindplane)
	bindplane.Status.Components.TransformAgent.Image = getTransformAgentImage(bindplane)

	if bindplane.Spec.OpAMP != nil && bindplane.Spec.OpAMP.Enabled {
		bindplane.Status.Components.OpAMP.Image = getOpAMPImage(bindplane)
	} else {
		bindplane.Status.Components.OpAMP.Image = ""
	}

	if !isTSDBRemoteEnabled(bindplane) {
		bindplane.Status.Components.TSDB.Image = getTSDBImage(bindplane)
	} else {
		bindplane.Status.Components.TSDB.Image = ""
	}
}

// allWorkloadsReady returns true when every workload has met its desired replica count.
// Optional workloads (nats, transformAgent, tsdb) are considered ready when their desired count is zero.
func (r *BindplaneReconciler) allWorkloadsReady(
	nodeReady, nodeDesired int32,
	natsReady, natsDesired int32,
	taReady, taDesired int32,
	jobsReady int32,
	tsdbReady, tsdbDesired int32,
) bool {
	const jobsDesired int32 = 1
	return nodeReady >= nodeDesired &&
		(natsDesired == 0 || natsReady >= natsDesired) &&
		(taDesired == 0 || taReady >= taDesired) &&
		jobsReady >= jobsDesired &&
		(tsdbDesired == 0 || tsdbReady >= tsdbDesired)
}

// deploymentReadyReplicas returns the ready replica count for a Deployment, or 0 if not found.
func (r *BindplaneReconciler) deploymentReadyReplicas(ctx context.Context, name, namespace string) int32 {
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, dep); err != nil {
		return 0
	}
	return dep.Status.ReadyReplicas
}

// statefulSetReadyReplicas returns the ready replica count for a StatefulSet, or 0 if not found.
func (r *BindplaneReconciler) statefulSetReadyReplicas(ctx context.Context, name, namespace string) int32 {
	ss := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ss); err != nil {
		return 0
	}
	return ss.Status.ReadyReplicas
}

// SetupWithManager sets up the controller with the Manager.
func (r *BindplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&bindplanev1alpha1.Bindplane{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{})

	// Only add a Rollout watch if the Argo Rollouts CRD is installed.
	// Watching a type whose CRD is absent blocks cache sync (CacheSyncTimeout = 2 min)
	// and prevents the controller from starting, even when argoRollout is not enabled.
	if gvks, _, err := mgr.GetScheme().ObjectKinds(&rolloutsv1alpha1.Rollout{}); err == nil && len(gvks) > 0 {
		if _, err := mgr.GetRESTMapper().RESTMapping(gvks[0].GroupKind(), gvks[0].Version); err == nil {
			b = b.Owns(&rolloutsv1alpha1.Rollout{})
		}
	}

	return b.Named("bindplane").Complete(r)
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

	// ServiceAccount is mostly immutable; only update when labels or annotations have changed.
	if equality.Semantic.DeepEqual(found.Labels, sa.Labels) && equality.Semantic.DeepEqual(found.Annotations, sa.Annotations) {
		return nil
	}
	found.Labels = sa.Labels
	found.Annotations = sa.Annotations
	return r.Update(ctx, found)
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

	// When the desired spec has Replicas=nil (HPA is managing scale), preserve the live
	// replica count so we do not fight the HPA on every reconcile.
	desired := deployment.Spec.DeepCopy()
	if desired.Replicas == nil {
		desired.Replicas = found.Spec.Replicas
	}

	// Skip the API call when nothing actually changed.
	if equality.Semantic.DeepEqual(found.Spec, *desired) && equality.Semantic.DeepEqual(found.Labels, deployment.Labels) {
		return nil
	}

	found.Spec = *desired
	found.Labels = deployment.Labels
	return r.Update(ctx, found)
}

// reconcileRollout creates or updates an Argo Rollout resource.
// When the desired spec has Replicas=nil (HPA is managing scale), the live replica
// count is preserved to avoid fighting the HPA on every reconcile.
func (r *BindplaneReconciler) reconcileRollout(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, rollout *rolloutsv1alpha1.Rollout, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, rollout, r.Scheme); err != nil {
		return err
	}

	found := &rolloutsv1alpha1.Rollout{}
	err := r.Get(ctx, types.NamespacedName{Name: rollout.Name, Namespace: rollout.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Rollout", "name", rollout.Name, "namespace", rollout.Namespace)
		return r.Create(ctx, rollout)
	} else if err != nil {
		return err
	}

	desired := rollout.Spec.DeepCopy()
	if desired.Replicas == nil {
		desired.Replicas = found.Spec.Replicas
	}

	if equality.Semantic.DeepEqual(found.Spec, *desired) && equality.Semantic.DeepEqual(found.Labels, rollout.Labels) {
		return nil
	}

	found.Spec = *desired
	found.Labels = rollout.Labels
	return r.Update(ctx, found)
}

// deleteDeploymentIfExists deletes the Deployment for a component if it exists.
// Called when switching to Argo Rollout mode to clean up the previous Deployment.
func (r *BindplaneReconciler) deleteDeploymentIfExists(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string, log logr.Logger) error {
	dep := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}, dep)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	log.Info("Deleting Deployment", "name", dep.Name)
	return r.Delete(ctx, dep)
}

// deleteRolloutIfExists deletes the Argo Rollout for a component if it exists.
// When the Argo Rollouts CRD is not installed, the error is treated as a no-op.
func (r *BindplaneReconciler) deleteRolloutIfExists(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string, log logr.Logger) error {
	rollout := &rolloutsv1alpha1.Rollout{}
	err := r.Get(ctx, types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}, rollout)
	if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	log.Info("Deleting Rollout", "name", rollout.Name)
	return r.Delete(ctx, rollout)
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

	// Skip the API call when the mutable fields haven't changed.
	if equality.Semantic.DeepEqual(found.Spec.Replicas, statefulSet.Spec.Replicas) &&
		equality.Semantic.DeepEqual(found.Spec.Template, statefulSet.Spec.Template) &&
		equality.Semantic.DeepEqual(found.Labels, statefulSet.Labels) {
		return nil
	}

	found.Spec.Replicas = statefulSet.Spec.Replicas
	found.Spec.Template = statefulSet.Spec.Template
	found.Labels = statefulSet.Labels
	return r.Update(ctx, found)
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

	// Preserve immutable / API-server-managed fields so Kubernetes does not reject the Update.
	service.Spec.ClusterIP = found.Spec.ClusterIP
	service.Spec.ClusterIPs = found.Spec.ClusterIPs
	service.Spec.IPFamilies = found.Spec.IPFamilies
	service.Spec.IPFamilyPolicy = found.Spec.IPFamilyPolicy

	// If the desired Service was built with WithPreserveSelectorKey, copy those keys
	// from the live Service into the desired selector so the operator does not remove
	// them (e.g. rollouts-pod-template-hash written by Argo Rollouts during a BlueGreen
	// cutover).
	if preserveAnnotation, ok := service.Annotations[annotationPreserveSelectorKeys]; ok {
		for _, key := range strings.Split(preserveAnnotation, ",") {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if val, exists := found.Spec.Selector[key]; exists {
				if service.Spec.Selector == nil {
					service.Spec.Selector = make(map[string]string)
				}
				service.Spec.Selector[key] = val
			}
		}
	}

	// Skip the API call when nothing actually changed.
	if equality.Semantic.DeepEqual(found.Spec.Ports, service.Spec.Ports) &&
		equality.Semantic.DeepEqual(found.Spec.Selector, service.Spec.Selector) &&
		equality.Semantic.DeepEqual(found.Labels, service.Labels) {
		return nil
	}

	found.Spec.Ports = service.Spec.Ports
	found.Spec.Selector = service.Spec.Selector
	found.Spec.ClusterIP = service.Spec.ClusterIP
	found.Spec.ClusterIPs = service.Spec.ClusterIPs
	found.Spec.IPFamilies = service.Spec.IPFamilies
	found.Spec.IPFamilyPolicy = service.Spec.IPFamilyPolicy
	found.Labels = service.Labels
	return r.Update(ctx, found)
}

// handleDeletion performs cleanup when a Bindplane CR is being deleted.
// Sets Phase=Deleting in status before removing the finalizer so observers can
// detect the in-progress deletion. All namespaced owned resources are garbage-
// collected automatically via ownerReference GC; this function is a hook for
// any resources that cannot carry an ownerReference (e.g., cluster-scoped).
func (r *BindplaneReconciler) handleDeletion(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
	log.Info("handling deletion, running cleanup before removing finalizer")

	// Mark Phase=Deleting so status reflects the in-progress deletion.
	bindplane.Status.Phase = "Deleting"
	if statusErr := r.Status().Update(ctx, bindplane); statusErr != nil {
		log.Error(statusErr, "failed to update Bindplane status to Deleting")
		// Non-fatal: proceed with finalizer removal even if status update fails.
	}

	// Future: add cleanup of any resources not covered by ownerReference GC here.
	controllerutil.RemoveFinalizer(bindplane, bindplaneFinalizer)
	return r.Update(ctx, bindplane)
}

// deletePodDisruptionBudgetIfExists deletes the PDB for a component if it exists.
// Called when disablePodDisruptionBudget is set to true to clean up a previously-created PDB.
func (r *BindplaneReconciler) deletePodDisruptionBudgetIfExists(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, component string, log logr.Logger) error {
	pdb := &policyv1.PodDisruptionBudget{}
	err := r.Get(ctx, types.NamespacedName{Name: getResourceName(bindplane, component), Namespace: bindplane.Namespace}, pdb)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	log.Info("Deleting PodDisruptionBudget", "name", pdb.Name)
	return r.Delete(ctx, pdb)
}

// reconcilePodDisruptionBudget reconciles a PodDisruptionBudget resource.
func (r *BindplaneReconciler) reconcilePodDisruptionBudget(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, pdb *policyv1.PodDisruptionBudget, log logr.Logger) error {
	if err := controllerutil.SetControllerReference(bindplane, pdb, r.Scheme); err != nil {
		return err
	}

	found := &policyv1.PodDisruptionBudget{}
	err := r.Get(ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating PodDisruptionBudget", "name", pdb.Name, "namespace", pdb.Namespace)
		return r.Create(ctx, pdb)
	} else if err != nil {
		return err
	}

	// Skip the API call when nothing has changed.
	if equality.Semantic.DeepEqual(found.Spec, pdb.Spec) && equality.Semantic.DeepEqual(found.Labels, pdb.Labels) {
		return nil
	}
	found.Spec = pdb.Spec
	found.Labels = pdb.Labels
	return r.Update(ctx, found)
}

// newPodDisruptionBudget creates a PodDisruptionBudget for a component with minAvailable: 1.
func newPodDisruptionBudget(bindplane *bindplanev1alpha1.Bindplane, component string) *policyv1.PodDisruptionBudget {
	minAvailable := intstr.FromInt32(1)
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getResourceName(bindplane, component),
			Namespace: bindplane.Namespace,
			Labels:    getLabels(bindplane, component),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &minAvailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: getSelectorLabels(bindplane, component),
			},
		},
	}
}

// getKubernetesEnvVars returns the common Kubernetes environment variables
// that should be present in all pods deployed by this operator
// combineEnvVars combines multiple slices of environment variables into a single slice
func combineEnvVars(envVarSlices ...[]corev1.EnvVar) []corev1.EnvVar {
	total := 0
	for _, s := range envVarSlices {
		total += len(s)
	}
	result := make([]corev1.EnvVar, 0, total)
	for _, envVars := range envVarSlices {
		result = append(result, envVars...)
	}
	return result
}

// prependExtraEnv places user-supplied extraEnv entries first, followed by the
// operator-managed env var slices. Because Kubernetes honours the LAST occurrence
// of a duplicate Name, putting operator vars last means operator values always win
// over any colliding name the user might set in extraEnv.
func prependExtraEnv(extraEnv []corev1.EnvVar, operatorEnv ...[]corev1.EnvVar) []corev1.EnvVar {
	total := len(extraEnv)
	for _, s := range operatorEnv {
		total += len(s)
	}
	out := make([]corev1.EnvVar, 0, total)
	out = append(out, extraEnv...)
	for _, s := range operatorEnv {
		out = append(out, s...)
	}
	return out
}

// appendExtraVolumes appends user-supplied extra volumes after the operator-managed volumes.
// Nil/empty extras return the operator slice unchanged.
func appendExtraVolumes(operatorVols []corev1.Volume, extra []corev1.Volume) []corev1.Volume {
	if len(extra) == 0 {
		return operatorVols
	}
	return append(operatorVols, extra...)
}

// appendExtraVolumeMounts appends user-supplied extra volume mounts after the operator-managed mounts.
// Nil/empty extras return the operator slice unchanged.
func appendExtraVolumeMounts(operatorMounts []corev1.VolumeMount, extra []corev1.VolumeMount) []corev1.VolumeMount {
	if len(extra) == 0 {
		return operatorMounts
	}
	return append(operatorMounts, extra...)
}

// getNodeExtraVolumes returns the user-supplied extra volumes for Bindplane Node.
func getNodeExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	return bindplane.Spec.Bindplane.ExtraVolumes
}

// getNodeExtraVolumeMounts returns the user-supplied extra volume mounts for Bindplane Node.
func getNodeExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	return bindplane.Spec.Bindplane.ExtraVolumeMounts
}

// getOpAMPExtraVolumes returns the user-supplied extra volumes for the OpAMP deployment, or nil.
func getOpAMPExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.OpAMP != nil {
		return bindplane.Spec.OpAMP.ExtraVolumes
	}
	return nil
}

// getOpAMPExtraVolumeMounts returns the user-supplied extra volume mounts for the OpAMP deployment, or nil.
func getOpAMPExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.OpAMP != nil {
		return bindplane.Spec.OpAMP.ExtraVolumeMounts
	}
	return nil
}

// getBindplaneJobsExtraVolumes returns the user-supplied extra volumes for Bindplane Jobs, or nil.
func getBindplaneJobsExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.BindplaneJobs != nil {
		return bindplane.Spec.BindplaneJobs.ExtraVolumes
	}
	return nil
}

// getBindplaneJobsExtraVolumeMounts returns the user-supplied extra volume mounts for Bindplane Jobs, or nil.
func getBindplaneJobsExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.BindplaneJobs != nil {
		return bindplane.Spec.BindplaneJobs.ExtraVolumeMounts
	}
	return nil
}

// getBindplaneJobsMigrateExtraVolumes returns the user-supplied extra volumes for Bindplane Jobs Migrate, or nil.
func getBindplaneJobsMigrateExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		return bindplane.Spec.BindplaneJobsMigrate.ExtraVolumes
	}
	return nil
}

// getBindplaneJobsMigrateExtraVolumeMounts returns the user-supplied extra volume mounts for Bindplane Jobs Migrate, or nil.
func getBindplaneJobsMigrateExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.BindplaneJobsMigrate != nil {
		return bindplane.Spec.BindplaneJobsMigrate.ExtraVolumeMounts
	}
	return nil
}

// getNatsExtraVolumes returns the user-supplied extra volumes for NATS, or nil.
func getNatsExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.Nats != nil {
		return bindplane.Spec.Nats.ExtraVolumes
	}
	return nil
}

// getNatsExtraVolumeMounts returns the user-supplied extra volume mounts for NATS, or nil.
func getNatsExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.Nats != nil {
		return bindplane.Spec.Nats.ExtraVolumeMounts
	}
	return nil
}

// getTransformAgentExtraVolumes returns the user-supplied extra volumes for Transform Agent, or nil.
func getTransformAgentExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.TransformAgent != nil {
		return bindplane.Spec.TransformAgent.ExtraVolumes
	}
	return nil
}

// getTransformAgentExtraVolumeMounts returns the user-supplied extra volume mounts for Transform Agent, or nil.
func getTransformAgentExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.TransformAgent != nil {
		return bindplane.Spec.TransformAgent.ExtraVolumeMounts
	}
	return nil
}

// getTSDBExtraVolumes returns the user-supplied extra volumes for TSDB, or nil.
func getTSDBExtraVolumes(bindplane *bindplanev1alpha1.Bindplane) []corev1.Volume {
	if bindplane.Spec.TSDB != nil {
		return bindplane.Spec.TSDB.ExtraVolumes
	}
	return nil
}

// getTSDBExtraVolumeMounts returns the user-supplied extra volume mounts for TSDB, or nil.
func getTSDBExtraVolumeMounts(bindplane *bindplanev1alpha1.Bindplane) []corev1.VolumeMount {
	if bindplane.Spec.TSDB != nil {
		return bindplane.Spec.TSDB.ExtraVolumeMounts
	}
	return nil
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

// newPodSecurityContext creates the default pod security context for Bindplane workloads.
func newPodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		FSGroup:    new(defaultRunAsGroup),
		RunAsGroup: new(defaultRunAsGroup),
		RunAsUser:  new(defaultRunAsUser),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
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
func newServiceAccount(bindplane *bindplanev1alpha1.Bindplane, component string, annotations map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        getResourceName(bindplane, component),
			Namespace:   bindplane.Namespace,
			Labels:      getLabels(bindplane, component),
			Annotations: annotations,
		},
	}
}

// serviceOptions holds configuration options for creating a Service
type serviceOptions struct {
	ports                []corev1.ServicePort
	preserveSelectorKeys []string
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

// WithPreserveSelectorKey marks selector keys that should be copied from the live
// Service into the desired Service before the equality comparison and update.
// This prevents the operator from removing keys written externally (e.g., the
// rollouts-pod-template-hash selector that Argo Rollouts writes during a BlueGreen
// cutover).
func WithPreserveSelectorKey(keys ...string) serviceOption {
	return func(opts *serviceOptions) {
		opts.preserveSelectorKeys = append(opts.preserveSelectorKeys, keys...)
	}
}

// newService creates a ClusterIP Service for a component
// It accepts variadic serviceOption functions to configure ports and selector preservation.
func newService(bindplane *bindplanev1alpha1.Bindplane, component string, opts ...serviceOption) *corev1.Service {
	labels := getLabels(bindplane, component)
	selectorLabels := getSelectorLabels(bindplane, component)

	options := &serviceOptions{
		ports: []corev1.ServicePort{},
	}

	for _, opt := range opts {
		opt(options)
	}

	svc := &corev1.Service{
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

	// Store preserve-selector keys as an annotation so reconcileService can read them.
	if len(options.preserveSelectorKeys) > 0 {
		if svc.Annotations == nil {
			svc.Annotations = make(map[string]string)
		}
		svc.Annotations[annotationPreserveSelectorKeys] = strings.Join(options.preserveSelectorKeys, ",")
	}

	return svc
}

// defaultPodAntiAffinity returns a preferred pod anti-affinity rule that spreads pods
// across nodes using the component selector labels. The preference is soft (weight 100)
// so it does not block scheduling when cluster capacity is constrained.
func defaultPodAntiAffinity(bindplane *bindplanev1alpha1.Bindplane, component string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: getSelectorLabels(bindplane, component),
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
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
// and internal TLS (cert-manager: Prometheus remote write client cert, NATS TLS, Transform Agent TLS).
// Used by Node, Jobs, and NATS so they receive all config TLS secrets when configured.
func getConfigTLSVolumesAndMounts(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	ldapVols, ldapMounts := getLDAPTLSVolumeAndMount(bindplane)
	netVols, netMounts := getNetworkTLSVolumeAndMount(bindplane)
	pgVols, pgMounts := getPostgresTLSVolumeAndMount(bindplane)
	internalVols, internalMounts := getInternalTLSVolumesAndMounts(bindplane)
	natsVols, natsMounts := getNatsTLSVolumesAndMounts(bindplane)
	transformAgentVols, transformAgentMounts := getTransformAgentTLSVolumesAndMounts(bindplane)
	vols := append(append(append(append(append(ldapVols, netVols...), pgVols...), internalVols...), natsVols...), transformAgentVols...)
	mounts := append(append(append(append(append(ldapMounts, netMounts...), pgMounts...), internalMounts...), natsMounts...), transformAgentMounts...)
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

// getTransformAgentTLSVolumesAndMounts returns volumes and mounts for Transform Agent TLS
// when cert-manager is used (spec.transformAgent.tls.certManager).
func getTransformAgentTLSVolumesAndMounts(bindplane *bindplanev1alpha1.Bindplane) ([]corev1.Volume, []corev1.VolumeMount) {
	if !isTransformAgentCertManagerTLSEnabled(bindplane) {
		return nil, nil
	}
	vol := corev1.Volume{
		Name: internalTLSTransformAgentVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: getResourceName(bindplane, transformAgentTLSCertSuffix)},
		},
	}
	mount := corev1.VolumeMount{
		Name:      internalTLSTransformAgentVolumeName,
		MountPath: internalTLSTransformAgentMountPath,
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
