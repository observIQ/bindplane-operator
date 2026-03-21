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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BindplaneSpec defines the desired state of Bindplane.
type BindplaneSpec struct {
	// Config contains Bindplane's configuration (license, auth, network, store, eventBus)
	// This config is shared by Node, Jobs, and Jobs Migrate
	Config BindplaneConfigSpec `json:"config"`

	// Bindplane configuration and pod specification
	Bindplane BindplaneComponentSpec `json:"bindplane"`

	// Bindplane Jobs pod specification
	// +optional
	BindplaneJobs *BindplaneJobsComponentSpec `json:"bindplaneJobs,omitempty"`

	// Bindplane Jobs Migrate pod specification
	// +optional
	BindplaneJobsMigrate *BindplaneJobsMigrateComponentSpec `json:"bindplaneJobsMigrate,omitempty"`

	// Transform Agent pod specification
	// +optional
	// +kubebuilder:default={}
	TransformAgent *TransformAgentComponentSpec `json:"transformAgent,omitempty"`

	// TSDB pod specification
	// +optional
	TSDB *TSDBComponentSpec `json:"tsdb,omitempty"`

	// NATS pod specification
	// +optional
	// +kubebuilder:default={}
	Nats *NatsComponentSpec `json:"nats,omitempty"`
}

// BindplaneComponentSpec defines the Bindplane component pod specification
type BindplaneComponentSpec struct {
	// Replicas specifies the number of replicas for Bindplane Node deployment
	// +optional
	// +kubebuilder:default=3
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for Bindplane Node
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneJobsComponentSpec defines the Bindplane Jobs component pod specification
type BindplaneJobsComponentSpec struct {
	// PodTemplate defines pod template specification for Bindplane Jobs
	// Note: Jobs are restricted to 1 replica and cannot be scaled
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneJobsMigrateComponentSpec defines the Bindplane Jobs Migrate component pod specification
type BindplaneJobsMigrateComponentSpec struct {
	// PodTemplate defines pod template specification for Bindplane Jobs Migrate
	// Note: Jobs Migrate are restricted to 1 replica and cannot be scaled
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// BindplaneConfigSpec defines Bindplane's configuration
// +kubebuilder:validation:XValidation:rule="has(self.license) != has(self.licenseSecretRef)",message="exactly one of license or licenseSecretRef must be set"
type BindplaneConfigSpec struct {
	// License is the Bindplane license key
	// +optional
	// +kubebuilder:validation:MinLength=1
	License string `json:"license,omitempty"`

	// LicenseSecretRef references a Kubernetes Secret containing the Bindplane license key.
	// Takes precedence over License if both are set.
	// +optional
	LicenseSecretRef *corev1.SecretKeySelector `json:"licenseSecretRef,omitempty"`

	// Auth configuration for Bindplane
	// +optional
	Auth *AuthConfig `json:"auth,omitempty"`

	// Network configuration for Bindplane
	// +optional
	Network *NetworkConfig `json:"network,omitempty"`

	// Store configuration for Bindplane
	Store StoreConfig `json:"store"`

	// Tracing configuration for Bindplane. When omitted or type empty, tracing is disabled.
	// +optional
	Tracing *TracingConfig `json:"tracing,omitempty"`

	// Metrics configuration for Bindplane. When omitted, defaults to prometheus type with interval 60s and endpoint /metrics.
	// +optional
	Metrics *MetricsConfig `json:"metrics,omitempty"`

	// MaxConcurrency is the maximum number of concurrent OpAMP operations.
	// Generally set to the same value as spec.config.agents.maxSimultaneousConnections.
	// Do not modify unless directed by Bindplane support.
	// +optional
	// +kubebuilder:default=10
	MaxConcurrency int `json:"maxConcurrency,omitempty"`

	// AuditTrail configures audit trail retention. When omitted, retentionDays defaults to 365.
	// +optional
	AuditTrail *AuditTrailConfig `json:"auditTrail,omitempty"`

	// TSDB configures TLS and remote settings for Bindplane's TSDB integration.
	// +optional
	TSDB *TSDBConfig `json:"tsdb,omitempty"`

	// Nats configures TLS for the NATS event bus (client and server). Cert-manager only.
	// +optional
	Nats *NatsConfig `json:"nats,omitempty"`

	// Profiling configures Google Cloud Profiler for Bindplane. When omitted or disabled, profiling is off.
	// +optional
	Profiling *ProfilingConfig `json:"profiling,omitempty"`

	// Pprof configures the pprof HTTP server for Bindplane. When omitted or disabled, pprof is off.
	// +optional
	Pprof *PprofConfig `json:"pprof,omitempty"`

	// Status configures the Bindplane status check endpoints.
	// +optional
	Status *StatusConfig `json:"status,omitempty"`

	// EventBus configures the event bus (NATS) integration, including health checks.
	// +optional
	EventBus *EventBusConfig `json:"eventBus,omitempty"`

	// Analytics configures Bindplane analytics reporting.
	// +optional
	Analytics *AnalyticsConfig `json:"analytics,omitempty"`

	// Logging configures the Bindplane log level and output destination.
	// +optional
	Logging *LoggingConfig `json:"logging,omitempty"`

	// Advanced configures advanced Bindplane options. These are typically used to
	// fine-tune behavior at scale and are not required for basic operation.
	// +optional
	Advanced *AdvancedConfig `json:"advanced,omitempty"`

	// Agents configures Bindplane agent connection, heartbeat, rebalance, and authentication options.
	// When omitted, Bindplane uses its own defaults.
	// +optional
	Agents *AgentsConfig `json:"agents,omitempty"`
}

// AgentsConfig configures how Bindplane communicates with agents.
type AgentsConfig struct {
	// Auth configures authentication for agent connections.
	// +optional
	Auth *AgentsAuthConfig `json:"auth,omitempty"`

	// HeartbeatInterval is the interval on which to perform a heartbeat over agent connections (e.g. "30s").
	// When omitted, Bindplane uses its own default.
	// +optional
	HeartbeatInterval string `json:"heartbeatInterval,omitempty"`

	// HeartbeatTTL is the amount of time between agent-initiated heartbeat messages before an agent
	// connection expires (e.g. "1m"). When omitted, Bindplane uses its own default.
	// +optional
	HeartbeatTTL string `json:"heartbeatTTL,omitempty"`

	// HeartbeatExpiryInterval is the interval between reaping expired agents (e.g. "30s").
	// When omitted, Bindplane uses its own default.
	// +optional
	HeartbeatExpiryInterval string `json:"heartbeatExpiryInterval,omitempty"`

	// RebalanceInterval is the interval between rebalancing agents (e.g. "1h").
	// When omitted, Bindplane uses its own default.
	// +optional
	RebalanceInterval string `json:"rebalanceInterval,omitempty"`

	// RebalancePercentage is the percentage of agents to rebalance (0–100).
	// 0 disables percentage-based rebalancing. When omitted, Bindplane uses its own default.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	RebalancePercentage *int `json:"rebalancePercentage,omitempty"`

	// RebalanceJitter is the maximum percentage jitter to add to the rebalance interval (0–100).
	// When omitted, Bindplane uses its own default.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	RebalanceJitter *int `json:"rebalanceJitter,omitempty"`

	// MaxSimultaneousConnections is the maximum number of goroutines that will service
	// OpAMP connections concurrently. Generally set to the same value as
	// spec.config.maxConcurrency. Do not modify unless directed by Bindplane support.
	// +optional
	// +kubebuilder:default=10
	MaxSimultaneousConnections int `json:"maxSimultaneousConnections,omitempty"`
}

// AgentsAuthConfig configures authentication for agent connections.
type AgentsAuthConfig struct {
	// Type specifies the authentication method(s) for agent connections.
	// Can be a single method or a comma-separated list (e.g. "oauth,secretKey").
	// Valid values: secretKey, oauth. When omitted, Bindplane defaults to secretKey.
	// +optional
	Type string `json:"type,omitempty"`

	// SecretKey configures the secret key authentication method.
	// +optional
	SecretKey *AgentsAuthSecretKeyConfig `json:"secretKey,omitempty"`

	// OAuth configures the OAuth authentication method.
	// +optional
	OAuth *AgentsAuthOAuthConfig `json:"oauth,omitempty"`
}

// AgentsAuthSecretKeyConfig configures secret key authentication for agent connections.
type AgentsAuthSecretKeyConfig struct {
	// Headers is the list of HTTP headers to read the secret key from.
	// When omitted, Bindplane defaults to ["X-Bindplane-Authorization", "Authorization"].
	// +optional
	Headers []string `json:"headers,omitempty"`
}

// AgentsAuthOAuthConfig configures OAuth authentication for agent connections.
type AgentsAuthOAuthConfig struct {
	// Issuer is the URL of the OAuth provider used to validate the token's iss claim.
	// +optional
	Issuer string `json:"issuer,omitempty"`

	// Audiences is the list of valid audience values. The token's aud claim must match
	// at least one of these values.
	// +optional
	Audiences []string `json:"audiences,omitempty"`

	// RequiredClaims is the list of claim names that must be present in the token.
	// +optional
	RequiredClaims []string `json:"requiredClaims,omitempty"`

	// RequiredScopes is the list of scopes that must all be present in the token.
	// +optional
	RequiredScopes []string `json:"requiredScopes,omitempty"`

	// CacheTTL is the duration a valid OAuth token is cached (e.g. "1h").
	// When omitted, Bindplane uses its own default.
	// +optional
	CacheTTL string `json:"cacheTTL,omitempty"`
}

// AdvancedConfig defines advanced Bindplane configuration options.
type AdvancedConfig struct {
	// Store contains advanced store configuration options.
	// +optional
	Store *AdvancedStoreConfig `json:"store,omitempty"`
	// Server contains advanced server configuration options.
	// +optional
	Server *AdvancedServerConfig `json:"server,omitempty"`
	// Cache contains advanced cache configuration options.
	// +optional
	Cache *AdvancedCacheConfig `json:"cache,omitempty"`
}

// AdvancedStoreConfig contains advanced store configuration options.
type AdvancedStoreConfig struct {
	// Stats configures advanced measurement storage tuning.
	// +optional
	Stats *AdvancedStoreStatsConfig `json:"stats,omitempty"`
}

// AdvancedStoreStatsConfig tunes measurement pipeline performance.
// All fields are optional; when omitted Bindplane uses its own defaults.
type AdvancedStoreStatsConfig struct {
	// BatchFlushInterval is the interval at which to flush measurement batches (e.g. "1s").
	// +optional
	BatchFlushInterval string `json:"batchFlushInterval,omitempty"`
	// WorkerCount is the number of workers saving measurements to the backend.
	// +optional
	WorkerCount int `json:"workerCount,omitempty"`
	// EnableSorting enables sorting of metrics by timestamp before saving.
	// +optional
	EnableSorting bool `json:"enableSorting,omitempty"`
	// MetricChannelSize is the buffer size for the incoming metrics channel.
	// +optional
	MetricChannelSize int `json:"metricChannelSize,omitempty"`
	// BatchChannelSize is the buffer size for the batch channel between accept and save workers.
	// +optional
	BatchChannelSize int `json:"batchChannelSize,omitempty"`
}

// AdvancedServerConfig contains advanced HTTP/OpAMP server options.
// All fields are optional; when omitted Bindplane uses its own defaults.
type AdvancedServerConfig struct {
	// MaxRequestBytes is the maximum request body size the server accepts, excluding offline
	// agent uploads. When omitted, Bindplane defaults to 10485760 (10 MiB).
	// +optional
	MaxRequestBytes int64 `json:"maxRequestBytes,omitempty"`
	// OpAMPShutdownGracePeriod is how long the OpAMP server waits for agents to disconnect
	// during shutdown (e.g. "30s"). When omitted, Bindplane defaults to 30s.
	// +optional
	OpAMPShutdownGracePeriod string `json:"opampShutdownGracePeriod,omitempty"`
}

// AdvancedCacheConfig configures the distributed cache.
type AdvancedCacheConfig struct {
	// Type is the cache backend to use. Currently only "redis" is supported.
	// +optional
	// +kubebuilder:validation:Enum=redis
	Type string `json:"type,omitempty"`
	// Redis configures the Redis cache connection.
	// +optional
	Redis *AdvancedCacheRedisConfig `json:"redis,omitempty"`
}

// AdvancedCacheRedisConfig configures a Redis cache backend.
type AdvancedCacheRedisConfig struct {
	// Address is the Redis server address in host:port form (e.g. "redis.default.svc:6379").
	Address string `json:"address"`
	// Password is the Redis password (plain text). Use PasswordSecretRef instead for sensitive values.
	// +optional
	Password string `json:"password,omitempty"`
	// PasswordSecretRef references a Kubernetes Secret containing the Redis password.
	// Takes precedence over Password when both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
	// DB is the Redis database index. When omitted, defaults to 0.
	// +optional
	DB int `json:"db,omitempty"`
	// ReadTimeout is the read timeout for Redis commands (e.g. "5s"). When omitted, the Redis client default is used.
	// +optional
	ReadTimeout string `json:"readTimeout,omitempty"`
	// WriteTimeout is the write timeout for Redis commands (e.g. "5s"). When omitted, the Redis client default is used.
	// +optional
	WriteTimeout string `json:"writeTimeout,omitempty"`
	// EnableTLS enables TLS for the Redis connection.
	// +optional
	EnableTLS bool `json:"enableTLS,omitempty"`
	// TLS configures TLS for the Redis connection. Only relevant when EnableTLS is true.
	// +optional
	TLS *AdvancedCacheRedisTLSConfig `json:"tls,omitempty"`
}

// AdvancedCacheRedisTLSConfig configures TLS for Redis via a Kubernetes Secret.
// The operator mounts the Secret and sets file-path env vars automatically.
type AdvancedCacheRedisTLSConfig struct {
	// SecretName is the name of the Secret containing TLS assets.
	// +optional
	SecretName string `json:"secretName,omitempty"`
	// CertKey is the key within SecretName for the TLS certificate file.
	// +optional
	CertKey string `json:"certKey,omitempty"`
	// KeyKey is the key within SecretName for the TLS private key file.
	// +optional
	KeyKey string `json:"keyKey,omitempty"`
	// CAKey is the key within SecretName for the CA certificate file.
	// +optional
	CAKey string `json:"caKey,omitempty"`
	// SkipVerify disables TLS certificate verification.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
	// MinTLSVersion is the minimum TLS version. One of: 1.2, 1.3.
	// +optional
	// +kubebuilder:validation:Enum="1.2";"1.3"
	MinTLSVersion string `json:"minTLSVersion,omitempty"`
}

// AuditTrailConfig defines audit trail configuration
type AuditTrailConfig struct {
	// RetentionDays is the number of days to retain audit trail events.
	// +optional
	// +kubebuilder:default=365
	RetentionDays int `json:"retentionDays,omitempty"`
}

// ProfilingConfig configures Google Cloud Profiler for Bindplane.
// +kubebuilder:validation:XValidation:rule="!self.enabled || (size(self.projectID) > 0)",message="projectID is required when profiling is enabled"
type ProfilingConfig struct {
	// Enabled turns on Google Cloud Profiler. When false or omitted, profiling is disabled.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ProjectID is the GCP project ID. Required when enabled is true.
	// +optional
	ProjectID string `json:"projectID,omitempty"`

	// NoCPU disables CPU profiling.
	// +optional
	// +kubebuilder:default=false
	NoCPU bool `json:"noCPU,omitempty"`

	// NoAlloc disables allocation profiling.
	// +optional
	// +kubebuilder:default=false
	NoAlloc bool `json:"noAlloc,omitempty"`

	// NoHeap disables heap profiling.
	// +optional
	// +kubebuilder:default=false
	NoHeap bool `json:"noHeap,omitempty"`

	// NoGoroutine disables goroutine profiling.
	// +optional
	// +kubebuilder:default=false
	NoGoroutine bool `json:"noGoroutine,omitempty"`

	// Mutex enables mutex profiling (disabled by default in Bindplane).
	// +optional
	// +kubebuilder:default=false
	Mutex bool `json:"mutex,omitempty"`
}

// StatusConfig configures the Bindplane status check endpoints.
// +kubebuilder:validation:XValidation:rule="!self.enabled || (size(self.keys) > 0 || has(self.keysSecretRef))",message="at least one key must be configured when status is enabled"
type StatusConfig struct {
	// Enabled controls whether the status check endpoints are enabled.
	// Defaults to true.
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Keys are UUIDs used to authenticate requests to the status check endpoints.
	// Supports multiple keys to allow rotation. At least one is required when enabled is true.
	// +optional
	Keys []string `json:"keys,omitempty"`

	// KeysSecretRef references a Kubernetes Secret containing status check keys.
	// The secret value should be comma-delimited UUIDs to support rotation.
	// Takes precedence over Keys if both are set.
	// +optional
	KeysSecretRef *corev1.SecretKeySelector `json:"keysSecretRef,omitempty"`
}

// AnalyticsConfig configures Bindplane analytics reporting.
type AnalyticsConfig struct {
	// Disabled turns off analytics reporting. When false or omitted, analytics are enabled.
	// Free licenses do not support disabling analytics; this option is ignored for that license type.
	// +optional
	// +kubebuilder:default=false
	Disabled bool `json:"disabled,omitempty"`

	// SegmentWriteKey overrides the default Segment write key used for analytics.
	// Do not set unless directed by Bindplane support.
	// +optional
	SegmentWriteKey string `json:"segmentWriteKey,omitempty"`
}

// EventBusHealthConfig configures the Bindplane event bus health check.
// The health check sends an event over NATS and waits for responses from other pods.
// Health check failures affect only the status page in the Bindplane web interface;
// they do not cause pod shutdown or failure.
type EventBusHealthConfig struct {
	// RequiredHosts is the minimum number of pods that must respond to the health check
	// event for the event bus to be considered healthy. When omitted, defaults to
	// floor(total / 2) + 1, where total is the sum of node, NATS, jobs, and
	// jobs-migrate replicas.
	// +optional
	// +kubebuilder:validation:Minimum=1
	RequiredHosts *int32 `json:"requiredHosts,omitempty"`

	// Interval is how often the event bus health check is performed (e.g. 15s, 1m).
	// When omitted, the Bindplane server default is used.
	// +optional
	Interval string `json:"interval,omitempty"`
}

// EventBusConfig configures the Bindplane event bus (NATS) integration.
type EventBusConfig struct {
	// Health configures the event bus health check endpoints.
	// +optional
	Health *EventBusHealthConfig `json:"health,omitempty"`
}

// PprofConfig configures the pprof HTTP server for Bindplane.
type PprofConfig struct {
	// Enabled turns on the pprof server. When false or omitted, pprof is disabled.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Endpoint is the host:port the pprof server listens on. When unset, defaults to 127.0.0.1:6060.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}

// TSDBConfig configures Bindplane's TSDB component (default implementation: Prometheus).
type TSDBConfig struct {
	// Remote configures Bindplane to use an externally managed TSDB-compatible backend
	// (for example, Prometheus, Mimir, or VictoriaMetrics) instead of the operator-managed TSDB StatefulSet.
	// +optional
	Remote *TSDBRemoteConfig `json:"remote,omitempty"`

	// TLS configures TLS for TSDB remote write.
	// +optional
	TLS *TSDBTLSConfig `json:"tls,omitempty"`
}

// TSDBRemoteConfig defines how Bindplane connects to an externally managed TSDB-compatible backend.
// +kubebuilder:validation:XValidation:rule="self.enable || (!has(self.host) && !has(self.queryPathPrefix) && !has(self.remoteWrite) && !has(self.port))",message="host, port, queryPathPrefix, and remoteWrite must be unset when enable is false"
// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.host)",message="host is required when enable is true"
// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.port)",message="port is required when enable is true"
type TSDBRemoteConfig struct {
	// Enable controls whether Bindplane should connect to an external TSDB-compatible backend.
	// When false, all other fields in this object must be omitted.
	// +optional
	Enable bool `json:"enable,omitempty"`

	// Host is the hostname or IP of the external TSDB-compatible backend.
	// Required when enable is true.
	// +optional
	Host string `json:"host,omitempty"`

	// Port is the TCP port of the external TSDB-compatible backend.
	// Required when enable is true.
	// +optional
	// +kubebuilder:default=9090
	Port int32 `json:"port,omitempty"`

	// QueryPathPrefix is an optional prefix path for PromQL APIs (for example, /prometheus).
	// +optional
	QueryPathPrefix string `json:"queryPathPrefix,omitempty"`

	// RemoteWrite optionally overrides where Bindplane sends TSDB remote write traffic.
	// +optional
	RemoteWrite *TSDBRemoteWriteConfig `json:"remoteWrite,omitempty"`
}

// TSDBRemoteWriteConfig defines optional remote write endpoint overrides.
// +kubebuilder:validation:XValidation:rule="(has(self.host) && has(self.port)) || (!has(self.host) && !has(self.port))",message="host and port must be set together"
type TSDBRemoteWriteConfig struct {
	// Host is the remote write hostname or IP. Must be set together with port.
	// +optional
	Host string `json:"host,omitempty"`

	// Port is the remote write TCP port. Must be set together with host.
	// +optional
	Port int32 `json:"port,omitempty"`

	// Endpoint is the remote write HTTP path.
	// +optional
	// +kubebuilder:default="/api/v1/write"
	Endpoint string `json:"endpoint,omitempty"`
}

// TSDBTLSConfig defines TLS for TSDB remote write.
// Exactly one of secretName (user-defined Secret) or certManager (cert-manager Issuer/ClusterIssuer) should be set.
type TSDBTLSConfig struct {
	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA (user-defined TLS).
	// Omit when using certManager.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// CertKey is the key in the Secret for the TLS certificate.
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key.
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// CertManager references a cert-manager Issuer or ClusterIssuer to issue server and client certs (mTLS).
	// Mutually exclusive with secretName.
	// +optional
	CertManager *CertManagerTLSIssuerRef `json:"certManager,omitempty"`

	// SkipVerify disables TLS certificate verification for the TSDB remote write client. Only set for testing.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
}

// NatsConfig configures the NATS event bus (client and server use the same TLS config).
type NatsConfig struct {
	// TLS configures mutual TLS for NATS via cert-manager. When set, a single certificate is used for client, cluster, and HTTP ports.
	// +optional
	TLS *NatsTLSConfig `json:"tls,omitempty"`
}

// NatsTLSConfig defines TLS for NATS. Only cert-manager is supported; no secretName.
type NatsTLSConfig struct {
	// CertManager references a cert-manager Issuer or ClusterIssuer to issue the NATS certificate (used for client, cluster, and HTTP).
	// +optional
	CertManager *CertManagerTLSIssuerRef `json:"certManager,omitempty"`
}

// CertManagerTLSIssuerRef references a cert-manager Issuer or ClusterIssuer.
// See https://cert-manager.io/docs/concepts/issuer/
type CertManagerTLSIssuerRef struct {
	// Name is the name of the Issuer or ClusterIssuer resource.
	Name string `json:"name"`

	// Kind is the type of issuer. Either "Issuer" (namespaced) or "ClusterIssuer" (cluster-scoped).
	// +optional
	// +kubebuilder:validation:Enum=Issuer;ClusterIssuer
	// +kubebuilder:default=Issuer
	Kind string `json:"kind,omitempty"`

	// Group is the API group of the issuer. Defaults to cert-manager.io.
	// +optional
	// +kubebuilder:default=cert-manager.io
	Group string `json:"group,omitempty"`
}

// TracingConfig defines tracing configuration
type TracingConfig struct {
	// Type specifies the tracing type. One of: otlp, google. When empty, tracing is disabled.
	// +optional
	// +kubebuilder:validation:Enum=otlp;google
	Type string `json:"type,omitempty"`

	// OTLP configures OTLP tracing when Type is otlp.
	// +optional
	OTLP *TracingOTLPConfig `json:"otlp,omitempty"`

	// SamplingRate is the ratio between 0 and 1 of traces to keep. Omit or 0 to disable sampling.
	// +optional
	SamplingRate string `json:"samplingRate,omitempty"`
}

// TracingOTLPConfig defines OTLP tracing configuration
type TracingOTLPConfig struct {
	// Endpoint is the OTLP endpoint to send traces to (e.g. http://localhost:4317).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Insecure disables TLS verification for the OTLP connection.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	// Type specifies the metrics type. One of: otlp, prometheus.
	// +optional
	// +kubebuilder:validation:Enum=otlp;prometheus
	// +kubebuilder:default=prometheus
	Type string `json:"type,omitempty"`

	// Interval is the interval at which to export metrics (e.g. 60s). Used when Type is otlp.
	// +optional
	// +kubebuilder:default="60s"
	Interval string `json:"interval,omitempty"`

	// Prometheus configures Prometheus metrics when Type is prometheus.
	// +optional
	Prometheus *MetricsPrometheusConfig `json:"prometheus,omitempty"`

	// OTLP configures OTLP metrics when Type is otlp.
	// +optional
	OTLP *MetricsOTLPConfig `json:"otlp,omitempty"`
}

// MetricsPrometheusConfig defines Prometheus metrics configuration
type MetricsPrometheusConfig struct {
	// Endpoint is the HTTP path to serve metrics on (e.g. /metrics).
	// +optional
	// +kubebuilder:default="/metrics"
	Endpoint string `json:"endpoint,omitempty"`

	// Username is the basic auth username for the metrics endpoint, if any.
	// +optional
	Username string `json:"username,omitempty"`

	// Password is the basic auth password for the metrics endpoint.
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the metrics endpoint password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// LoggingOTLPConfig defines OTLP logging configuration.
type LoggingOTLPConfig struct {
	// Endpoint is the gRPC endpoint to send logs to (e.g. localhost:4317).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Insecure disables TLS verification for the OTLP connection.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// Interval is the interval at which to export logs (e.g. 60s).
	// When omitted, Bindplane uses its own default.
	// +optional
	Interval string `json:"interval,omitempty"`
}

// LoggingConfig defines logging configuration.
type LoggingConfig struct {
	// Level specifies the log level. One of: debug, info, warn, error.
	// +optional
	// +kubebuilder:validation:Enum=debug;info;warn;error
	// +kubebuilder:default=info
	Level string `json:"level,omitempty"`

	// Type specifies the logging output destination.
	// Use "stdout" to write logs to standard output, "otlp" to export via OTLP,
	// or "stdout,otlp" to write to both simultaneously.
	// +optional
	// +kubebuilder:default=stdout
	// +kubebuilder:validation:Pattern=`^(stdout|otlp)(,(stdout|otlp))?$`
	Type string `json:"type,omitempty"`

	// OTLP configures OTLP log export when Type includes otlp.
	// +optional
	OTLP *LoggingOTLPConfig `json:"otlp,omitempty"`
}

// MetricsOTLPConfig defines OTLP metrics configuration
type MetricsOTLPConfig struct {
	// Endpoint is the gRPC endpoint to send metrics to (e.g. localhost:4317).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Insecure disables TLS verification for the OTLP connection.
	// +optional
	Insecure bool `json:"insecure,omitempty"`
}

// TransformAgentComponentSpec defines the Transform Agent component pod specification
type TransformAgentComponentSpec struct {
	// Replicas specifies the number of replicas for Transform Agent deployment
	// +optional
	// +kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for Transform Agent
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// TSDBComponentSpec defines the TSDB component pod specification.
// By default, this deploys a Prometheus StatefulSet managed by the operator.
type TSDBComponentSpec struct {
	// PodTemplate defines pod template specification for the TSDB component
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`

	// Storage defines the persistent storage configuration for the TSDB component
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// TLS configures TLS for the TSDB server (StatefulSet). Use either secretName (user-defined Secret)
	// or certManager (cert-manager Issuer/ClusterIssuer), not both. When set, the TSDB serves remote write over TLS.
	// +optional
	TLS *TSDBTLSConfig `json:"tls,omitempty"`
}

// NatsComponentSpec defines the NATS component pod specification
type NatsComponentSpec struct {
	// Replicas specifies the number of replicas for NATS StatefulSet
	// +optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// PodTemplate defines pod template specification for NATS
	// +optional
	// +kubebuilder:validation:Type=object
	// +kubebuilder:pruning:PreserveUnknownFields
	PodTemplate *PodTemplateSpec `json:"podTemplate,omitempty"`
}

// StorageSpec defines persistent storage configuration
type StorageSpec struct {
	// VolumeClaimTemplate defines the template for creating PersistentVolumeClaims
	// This follows the same structure as StatefulSet volumeClaimTemplates
	VolumeClaimTemplate *VolumeClaimTemplate `json:"volumeClaimTemplate,omitempty"`
}

// VolumeClaimTemplate defines a template for creating PersistentVolumeClaims
type VolumeClaimTemplate struct {
	// Metadata for the PersistentVolumeClaim
	// +optional
	Metadata *metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the PersistentVolumeClaim specification
	Spec corev1.PersistentVolumeClaimSpec `json:"spec"`
}

// PodTemplateSpec defines pod template specification.
// This embeds corev1.PodTemplateSpec to allow arbitrary pod spec fields.
// Note: The operator will merge this with operator-managed fields, ensuring
// operator-managed fields (like ServiceAccountName, containers, etc.) take precedence.
// +kubebuilder:pruning:PreserveUnknownFields
type PodTemplateSpec struct {
	// Embedded PodTemplateSpec allows users to specify arbitrary pod spec fields
	// such as affinity, tolerations, nodeSelector, securityContext, etc.
	// Operator-managed fields (ServiceAccountName, containers, etc.) will be preserved.
	corev1.PodTemplateSpec `json:",inline"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	// Type specifies the authentication type.
	// +optional
	// +kubebuilder:validation:Enum=system;ldap;active-directory;oidc
	Type string `json:"type,omitempty"`

	// Username for authentication
	// +optional
	Username string `json:"username,omitempty"`

	// UsernameSecretRef references a Kubernetes Secret containing the auth username.
	// Takes precedence over Username if both are set.
	// +optional
	UsernameSecretRef *corev1.SecretKeySelector `json:"usernameSecretRef,omitempty"`

	// Password for authentication
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the auth password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// SessionsStrictMode enables strict mode for session cookies.
	// +optional
	SessionsStrictMode bool `json:"sessionsStrictMode,omitempty"`

	// LDAP is the configuration for ldap or active-directory auth types.
	// +optional
	LDAP *LDAPConfig `json:"ldap,omitempty"`

	// OIDC is the configuration for the oidc auth type.
	// +optional
	OIDC *OIDCConfig `json:"oidc,omitempty"`

	// Note: sessionSecret is not exposed - it will be dynamically generated and stored as a Kubernetes secret
}

// LDAPConfig defines LDAP and Active Directory authentication configuration
type LDAPConfig struct {
	// Protocol to use when connecting to the LDAP server. One of: ldap|ldaps
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Server is the LDAP server hostname
	// +optional
	Server string `json:"server,omitempty"`

	// Port is the LDAP server port
	// +optional
	Port string `json:"port,omitempty"`

	// BaseDN is the base distinguished name for user searches
	// +optional
	BaseDN string `json:"baseDN,omitempty"`

	// BindUser is the username used to bind to the LDAP server
	// +optional
	BindUser string `json:"bindUser,omitempty"`

	// BindUserSecretRef references a Kubernetes Secret containing the LDAP bind username.
	// Takes precedence over BindUser if both are set.
	// +optional
	BindUserSecretRef *corev1.SecretKeySelector `json:"bindUserSecretRef,omitempty"`

	// BindPassword is the password used to bind to the LDAP server
	// +optional
	BindPassword string `json:"bindPassword,omitempty"`

	// BindPasswordSecretRef references a Kubernetes Secret containing the LDAP bind password.
	// Takes precedence over BindPassword if both are set.
	// +optional
	BindPasswordSecretRef *corev1.SecretKeySelector `json:"bindPasswordSecretRef,omitempty"`

	// SearchFilter is the LDAP search filter used to locate users
	// +optional
	SearchFilter string `json:"searchFilter,omitempty"`

	// TLS configures TLS for LDAP using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_LDAP_TLS_CERT, BINDPLANE_LDAP_TLS_KEY, and BINDPLANE_LDAP_TLS_CA to the
	// mounted file paths. Omit TLS to disable mutual TLS / custom CA.
	// +optional
	TLS *LDAPTLSConfig `json:"tls,omitempty"`

	// TLSSkipVerify disables TLS certificate verification
	// +optional
	TLSSkipVerify bool `json:"tlsSkipVerify,omitempty"`
}

// LDAPTLSConfig defines TLS for LDAP by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
type LDAPTLSConfig struct {
	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA.
	SecretName string `json:"secretName"`

	// CertKey is the key in the Secret for the TLS certificate (for mutual TLS).
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key (for mutual TLS).
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate. Omit to use system CAs.
	// +optional
	CAKey string `json:"caKey,omitempty"`
}

// NetworkTLSConfig defines TLS for the Bindplane server by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
// Server-side TLS: set secretName, certKey, and keyKey. Mutual TLS: also set caKey.
type NetworkTLSConfig struct {
	// MinVersion is the minimum TLS version. One of: 1.2, 1.3. Omit to use server default.
	// +optional
	// +kubebuilder:validation:Enum=1.2;1.3
	MinVersion string `json:"minVersion,omitempty"`

	// SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA.
	SecretName string `json:"secretName"`

	// CertKey is the key in the Secret for the TLS certificate (server or mutual TLS).
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the TLS private key (server or mutual TLS).
	// +optional
	KeyKey string `json:"keyKey,omitempty"`

	// CAKey is the key in the Secret for the CA certificate. Set for mutual TLS (client cert verification); generally not used.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// SkipVerify disables TLS certificate verification. Only set for testing.
	// +optional
	SkipVerify bool `json:"skipVerify,omitempty"`
}

// PostgresTLSConfig defines TLS for PostgreSQL by referencing a Secret. The Secret is mounted
// at a fixed path; the operator sets the TLS env vars (sslRootCert, sslCert, sslKey) to the mounted file paths.
// Users specify only the secret name and key names, not mount paths.
// Server-side TLS: set secretName and caKey. Mutual TLS: set secretName, caKey, certKey, and keyKey.
type PostgresTLSConfig struct {
	// SecretName is the name of the Secret containing the CA and optionally client cert and key.
	SecretName string `json:"secretName"`

	// CAKey is the key in the Secret for the root CA (maps to sslRootCert). Required for TLS; enables server-side TLS.
	// +optional
	CAKey string `json:"caKey,omitempty"`

	// CertKey is the key in the Secret for the client certificate (maps to sslCert). Set with KeyKey for mutual TLS.
	// +optional
	CertKey string `json:"certKey,omitempty"`

	// KeyKey is the key in the Secret for the client private key (maps to sslKey). Set with CertKey for mutual TLS.
	// +optional
	KeyKey string `json:"keyKey,omitempty"`
}

// OIDCConfig defines OpenID Connect authentication configuration
type OIDCConfig struct {
	// ClientID is the OIDC OAuth2 client ID
	// +optional
	ClientID string `json:"clientID,omitempty"`

	// ClientIDSecretRef references a Kubernetes Secret containing the OIDC client ID.
	// Takes precedence over ClientID if both are set.
	// +optional
	ClientIDSecretRef *corev1.SecretKeySelector `json:"clientIDSecretRef,omitempty"`

	// ClientSecret is the OIDC OAuth2 client secret
	// +optional
	ClientSecret string `json:"clientSecret,omitempty"`

	// ClientSecretSecretRef references a Kubernetes Secret containing the OIDC client secret.
	// Takes precedence over ClientSecret if both are set.
	// +optional
	ClientSecretSecretRef *corev1.SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// Issuer is the URL of the OIDC provider
	// +optional
	Issuer string `json:"issuer,omitempty"`

	// Scopes is the list of OAuth2 scopes to request
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

// NetworkConfig defines network configuration
type NetworkConfig struct {
	// Host specifies the bind address
	// +optional
	Host string `json:"host,omitempty"`

	// Port specifies the port to listen on
	// +optional
	Port string `json:"port,omitempty"`

	// RemoteURL specifies the remote URL for Bindplane.
	// Defaults to http://<bindplane-name>-node:3001 (the internal node service URL).
	// Override this when using ingress, e.g. https://bindplane.my-corp.net
	// +optional
	RemoteURL string `json:"remoteURL,omitempty"`

	// WebURL is the URL used by the client for the web interface. Defaults to RemoteURL when not set. Only set when explicitly configuring.
	// +optional
	WebURL string `json:"webURL,omitempty"`

	// CorsAllowedOrigins is the allowed origin for CORS requests. Only set when explicitly configuring.
	// +optional
	CorsAllowedOrigins string `json:"corsAllowedOrigins,omitempty"`

	// TLS configures TLS for the Bindplane server using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_TLS_CERT, BINDPLANE_TLS_KEY, and optionally BINDPLANE_TLS_CA to the mounted file paths.
	// Omit or omit secretName/certKey/keyKey to disable server TLS (e.g. when using Ingress to terminate TLS).
	// +optional
	TLS *NetworkTLSConfig `json:"tls,omitempty"`
}

// StoreConfig defines store configuration
type StoreConfig struct {
	// Postgres configuration
	Postgres *PostgresConfig `json:"postgres"`

	// MaxEvents is the maximum number of events to merge into a single event.
	// When omitted, Bindplane defaults to 100.
	// +optional
	MaxEvents int `json:"maxEvents,omitempty"`

	// EventMergeWindow is the window during which events are merged (e.g. "100ms").
	// When omitted, Bindplane defaults to 100ms.
	// +optional
	EventMergeWindow string `json:"eventMergeWindow,omitempty"`

	// SummaryRollupRetentionDays is the number of days to retain daily rollup data.
	// 0 means indefinite retention (rollups are never deleted).
	// When omitted, Bindplane defaults to 365.
	// +optional
	SummaryRollupRetentionDays *int `json:"summaryRollupRetentionDays,omitempty"`
}

// PostgresConfig defines PostgreSQL store configuration
type PostgresConfig struct {
	// Host specifies the PostgreSQL host
	Host string `json:"host"`

	// Port specifies the PostgreSQL port
	// +optional
	Port string `json:"port,omitempty"`

	// ConnectTimeout specifies the connection timeout
	// +optional
	ConnectTimeout string `json:"connectTimeout,omitempty"`

	// StatementTimeout specifies the statement timeout
	// +optional
	StatementTimeout string `json:"statementTimeout,omitempty"`

	// Database specifies the database name
	// +optional
	Database string `json:"database,omitempty"`

	// SSLMode specifies the PostgreSQL SSL mode. One of: disable, require, verify-ca, verify-full.
	// +optional
	// +kubebuilder:default=disable
	// +kubebuilder:validation:Enum=disable;require;verify-ca;verify-full
	SSLMode string `json:"sslmode,omitempty"`

	// TLS configures TLS for PostgreSQL using a Secret. The operator mounts the Secret and sets
	// BINDPLANE_POSTGRES_SSL_ROOT_CERT, BINDPLANE_POSTGRES_SSL_CERT, and BINDPLANE_POSTGRES_SSL_KEY to the
	// mounted file paths. Server-side TLS: set secretName and caKey. Mutual TLS: also set certKey and keyKey.
	// +optional
	TLS *PostgresTLSConfig `json:"tls,omitempty"`

	// Username specifies the PostgreSQL username
	// +optional
	Username string `json:"username,omitempty"`

	// UsernameSecretRef references a Kubernetes Secret containing the PostgreSQL username.
	// Takes precedence over Username if both are set.
	// +optional
	UsernameSecretRef *corev1.SecretKeySelector `json:"usernameSecretRef,omitempty"`

	// Password specifies the PostgreSQL password
	// +optional
	Password string `json:"password,omitempty"`

	// PasswordSecretRef references a Kubernetes Secret containing the PostgreSQL password.
	// Takes precedence over Password if both are set.
	// +optional
	PasswordSecretRef *corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// MaxConnections specifies the maximum number of connections
	// +optional
	MaxConnections int `json:"maxConnections,omitempty"`

	// MaxIdleConnections specifies the maximum number of idle connections. Optional; no default.
	// +optional
	MaxIdleConnections *int `json:"maxIdleConnections,omitempty"`

	// MaxLifetime specifies the maximum connection lifetime
	// +optional
	MaxLifetime string `json:"maxLifetime,omitempty"`

	// MaxIdleTime specifies the maximum time a connection may remain idle (e.g. 20s, 1m). Optional; no default.
	// +optional
	MaxIdleTime string `json:"maxIdleTime,omitempty"`

	// Schema specifies the database schema
	// +optional
	Schema string `json:"schema,omitempty"`
}

// BindplaneStatus defines the observed state of Bindplane.
type BindplaneStatus struct {
	// Conditions represent the latest available observations of the Bindplane's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=bindplanes,singular=bindplane,scope=Namespaced

// Bindplane is the Schema for the bindplanes API.
type Bindplane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BindplaneSpec   `json:"spec,omitempty"`
	Status BindplaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BindplaneList contains a list of Bindplane.
type BindplaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bindplane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bindplane{}, &BindplaneList{})
}
