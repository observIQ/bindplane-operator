# API Reference

## Packages
- [k8s.bindplane.com/v1alpha1](#k8sbindplanecomv1alpha1)


## k8s.bindplane.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the bindplane v1alpha1 API group.

### Resource Types
- [Bindplane](#bindplane)



#### AdvancedCacheConfig



AdvancedCacheConfig configures the distributed cache.



_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the cache backend to use. Currently only "redis" is supported. |  | Enum: [redis] <br />Optional: \{\} <br /> |
| `redis` _[AdvancedCacheRedisConfig](#advancedcacheredisconfig)_ | Redis configures the Redis cache connection. |  | Optional: \{\} <br /> |


#### AdvancedCacheRedisConfig



AdvancedCacheRedisConfig configures a Redis cache backend.



_Appears in:_
- [AdvancedCacheConfig](#advancedcacheconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `address` _string_ | Address is the Redis server address in host:port form (e.g. "redis.default.svc:6379"). |  |  |
| `password` _string_ | Password is the Redis password (plain text). Use PasswordSecretRef instead for sensitive values. |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the Redis password.<br />Takes precedence over Password when both are set. |  | Optional: \{\} <br /> |
| `db` _integer_ | DB is the Redis database index. When omitted, defaults to 0. |  | Optional: \{\} <br /> |
| `readTimeout` _string_ | ReadTimeout is the read timeout for Redis commands (e.g. "5s"). When omitted, the Redis client default is used. |  | Optional: \{\} <br /> |
| `writeTimeout` _string_ | WriteTimeout is the write timeout for Redis commands (e.g. "5s"). When omitted, the Redis client default is used. |  | Optional: \{\} <br /> |
| `enableTLS` _boolean_ | EnableTLS enables TLS for the Redis connection. |  | Optional: \{\} <br /> |
| `tls` _[AdvancedCacheRedisTLSConfig](#advancedcacheredistlsconfig)_ | TLS configures TLS for the Redis connection. Only relevant when EnableTLS is true. |  | Optional: \{\} <br /> |


#### AdvancedCacheRedisTLSConfig



AdvancedCacheRedisTLSConfig configures TLS for Redis via a Kubernetes Secret.
The operator mounts the Secret and sets file-path env vars automatically.



_Appears in:_
- [AdvancedCacheRedisConfig](#advancedcacheredisconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | SecretName is the name of the Secret containing TLS assets. |  | Optional: \{\} <br /> |
| `certKey` _string_ | CertKey is the key within SecretName for the TLS certificate file. |  | Optional: \{\} <br /> |
| `keyKey` _string_ | KeyKey is the key within SecretName for the TLS private key file. |  | Optional: \{\} <br /> |
| `caKey` _string_ | CAKey is the key within SecretName for the CA certificate file. |  | Optional: \{\} <br /> |
| `skipVerify` _boolean_ | SkipVerify disables TLS certificate verification. |  | Optional: \{\} <br /> |
| `minTLSVersion` _string_ | MinTLSVersion is the minimum TLS version. One of: 1.2, 1.3. |  | Enum: [1.2 1.3] <br />Optional: \{\} <br /> |


#### AdvancedConfig



AdvancedConfig defines advanced Bindplane configuration options.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `store` _[AdvancedStoreConfig](#advancedstoreconfig)_ | Store contains advanced store configuration options. |  | Optional: \{\} <br /> |
| `server` _[AdvancedServerConfig](#advancedserverconfig)_ | Server contains advanced server configuration options. |  | Optional: \{\} <br /> |
| `cache` _[AdvancedCacheConfig](#advancedcacheconfig)_ | Cache contains advanced cache configuration options. |  | Optional: \{\} <br /> |


#### AdvancedServerConfig



AdvancedServerConfig contains advanced HTTP/OpAMP server options.
All fields are optional; when omitted Bindplane uses its own defaults.



_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxRequestBytes` _integer_ | MaxRequestBytes is the maximum request body size the server accepts, excluding offline<br />agent uploads. When omitted, Bindplane defaults to 10485760 (10 MiB). |  | Optional: \{\} <br /> |
| `opampShutdownGracePeriod` _string_ | OpAMPShutdownGracePeriod is how long the OpAMP server waits for agents to disconnect<br />during shutdown (e.g. "30s"). When omitted, Bindplane defaults to 30s. |  | Optional: \{\} <br /> |


#### AdvancedStoreConfig



AdvancedStoreConfig contains advanced store configuration options.



_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stats` _[AdvancedStoreStatsConfig](#advancedstorestatsconfig)_ | Stats configures advanced measurement storage tuning. |  | Optional: \{\} <br /> |


#### AdvancedStoreStatsConfig



AdvancedStoreStatsConfig tunes measurement pipeline performance.
All fields are optional; when omitted Bindplane uses its own defaults.



_Appears in:_
- [AdvancedStoreConfig](#advancedstoreconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `batchFlushInterval` _string_ | BatchFlushInterval is the interval at which to flush measurement batches (e.g. "1s"). |  | Optional: \{\} <br /> |
| `workerCount` _integer_ | WorkerCount is the number of workers saving measurements to the backend. |  | Optional: \{\} <br /> |
| `enableSorting` _boolean_ | EnableSorting enables sorting of metrics by timestamp before saving. |  | Optional: \{\} <br /> |
| `metricChannelSize` _integer_ | MetricChannelSize is the buffer size for the incoming metrics channel. |  | Optional: \{\} <br /> |
| `batchChannelSize` _integer_ | BatchChannelSize is the buffer size for the batch channel between accept and save workers. |  | Optional: \{\} <br /> |


#### AgentVersionsConfig



AgentVersionsConfig configures how Bindplane syncs agent versions.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `syncInterval` _string_ | SyncInterval is the interval at which to sync agent versions (e.g. "2h").<br />Must be at least 1h. When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |
| `clients` _string array_ | Clients is a deprecated list of version client types (e.g. ["bdot", "github"]).<br />Version clients are now configured per-agent-type via AgentType resources. |  | Optional: \{\} <br /> |


#### AgentsAuthConfig



AgentsAuthConfig configures authentication for agent connections.



_Appears in:_
- [AgentsConfig](#agentsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the authentication method(s) for agent connections.<br />Can be a single method or a comma-separated list (e.g. "oauth,secretKey").<br />Valid values: secretKey, oauth. When omitted, Bindplane defaults to secretKey. |  | Optional: \{\} <br /> |
| `secretKey` _[AgentsAuthSecretKeyConfig](#agentsauthsecretkeyconfig)_ | SecretKey configures the secret key authentication method. |  | Optional: \{\} <br /> |
| `oauth` _[AgentsAuthOAuthConfig](#agentsauthoauthconfig)_ | OAuth configures the OAuth authentication method. |  | Optional: \{\} <br /> |


#### AgentsAuthOAuthConfig



AgentsAuthOAuthConfig configures OAuth authentication for agent connections.



_Appears in:_
- [AgentsAuthConfig](#agentsauthconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `issuer` _string_ | Issuer is the URL of the OAuth provider used to validate the token's iss claim. |  | Optional: \{\} <br /> |
| `audiences` _string array_ | Audiences is the list of valid audience values. The token's aud claim must match<br />at least one of these values. |  | Optional: \{\} <br /> |
| `requiredClaims` _string array_ | RequiredClaims is the list of claim names that must be present in the token. |  | Optional: \{\} <br /> |
| `requiredScopes` _string array_ | RequiredScopes is the list of scopes that must all be present in the token. |  | Optional: \{\} <br /> |
| `cacheTTL` _string_ | CacheTTL is the duration a valid OAuth token is cached (e.g. "1h").<br />When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |


#### AgentsAuthSecretKeyConfig



AgentsAuthSecretKeyConfig configures secret key authentication for agent connections.



_Appears in:_
- [AgentsAuthConfig](#agentsauthconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `headers` _string array_ | Headers is the list of HTTP headers to read the secret key from.<br />When omitted, Bindplane defaults to ["X-Bindplane-Authorization", "Authorization"]. |  | Optional: \{\} <br /> |


#### AgentsConfig



AgentsConfig configures how Bindplane communicates with agents.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[AgentsAuthConfig](#agentsauthconfig)_ | Auth configures authentication for agent connections. |  | Optional: \{\} <br /> |
| `heartbeatInterval` _string_ | HeartbeatInterval is the interval on which to perform a heartbeat over agent connections (e.g. "30s").<br />When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |
| `heartbeatTTL` _string_ | HeartbeatTTL is the amount of time between agent-initiated heartbeat messages before an agent<br />connection expires (e.g. "1m"). When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |
| `heartbeatExpiryInterval` _string_ | HeartbeatExpiryInterval is the interval between reaping expired agents (e.g. "30s").<br />When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |
| `rebalanceInterval` _string_ | RebalanceInterval is the interval between rebalancing agents (e.g. "1h").<br />When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |
| `rebalancePercentage` _integer_ | RebalancePercentage is the percentage of agents to rebalance (0–100).<br />0 disables percentage-based rebalancing. When omitted, Bindplane uses its own default. |  | Maximum: 100 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `rebalanceJitter` _integer_ | RebalanceJitter is the maximum percentage jitter to add to the rebalance interval (0–100).<br />When omitted, Bindplane uses its own default. |  | Maximum: 100 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `maxSimultaneousConnections` _integer_ | MaxSimultaneousConnections is the maximum number of goroutines that will service<br />OpAMP connections concurrently. Generally set to the same value as<br />spec.config.maxConcurrency. Do not modify unless directed by Bindplane support. | 10 | Optional: \{\} <br /> |


#### AnalyticsConfig



AnalyticsConfig configures Bindplane analytics reporting.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `disabled` _boolean_ | Disabled turns off analytics reporting. When false or omitted, analytics are enabled.<br />Free licenses do not support disabling analytics; this option is ignored for that license type. | false | Optional: \{\} <br /> |
| `segmentWriteKey` _string_ | SegmentWriteKey overrides the default Segment write key used for analytics.<br />Do not set unless directed by Bindplane support. |  | Optional: \{\} <br /> |


#### AuditTrailConfig



AuditTrailConfig defines audit trail configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `retentionDays` _integer_ | RetentionDays is the number of days to retain audit trail events. | 365 | Optional: \{\} <br /> |


#### AuthConfig



AuthConfig defines authentication configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the authentication type. |  | Enum: [system ldap active-directory oidc] <br />Optional: \{\} <br /> |
| `username` _string_ | Username for authentication |  | Optional: \{\} <br /> |
| `usernameSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | UsernameSecretRef references a Kubernetes Secret containing the auth username.<br />Takes precedence over Username if both are set. |  | Optional: \{\} <br /> |
| `password` _string_ | Password for authentication |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the auth password.<br />Takes precedence over Password if both are set. |  | Optional: \{\} <br /> |
| `sessionsStrictMode` _boolean_ | SessionsStrictMode enables strict mode for session cookies. |  | Optional: \{\} <br /> |
| `ldap` _[LDAPConfig](#ldapconfig)_ | LDAP is the configuration for ldap or active-directory auth types. |  | Optional: \{\} <br /> |
| `oidc` _[OIDCConfig](#oidcconfig)_ | OIDC is the configuration for the oidc auth type. |  | Optional: \{\} <br /> |


#### Bindplane



Bindplane is the Schema for the bindplanes API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `k8s.bindplane.com/v1alpha1` | | |
| `kind` _string_ | `Bindplane` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BindplaneSpec](#bindplanespec)_ |  |  |  |


#### BindplaneComponentSpec



BindplaneComponentSpec defines the Bindplane component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | Replicas specifies the number of replicas for Bindplane Node deployment | 3 | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Bindplane Node |  | Type: object <br />Optional: \{\} <br /> |


#### BindplaneConfigSpec



BindplaneConfigSpec defines Bindplane's configuration



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `license` _string_ | License is the Bindplane license key |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `licenseSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | LicenseSecretRef references a Kubernetes Secret containing the Bindplane license key.<br />Takes precedence over License if both are set. |  | Optional: \{\} <br /> |
| `auth` _[AuthConfig](#authconfig)_ | Auth configuration for Bindplane |  | Optional: \{\} <br /> |
| `network` _[NetworkConfig](#networkconfig)_ | Network configuration for Bindplane |  | Optional: \{\} <br /> |
| `store` _[StoreConfig](#storeconfig)_ | Store configuration for Bindplane |  |  |
| `tracing` _[TracingConfig](#tracingconfig)_ | Tracing configuration for Bindplane. When omitted or type empty, tracing is disabled. |  | Optional: \{\} <br /> |
| `metrics` _[MetricsConfig](#metricsconfig)_ | Metrics configuration for Bindplane. When omitted, defaults to prometheus type with interval 60s and endpoint /metrics. |  | Optional: \{\} <br /> |
| `maxConcurrency` _integer_ | MaxConcurrency is the maximum number of concurrent OpAMP operations.<br />Generally set to the same value as spec.config.agents.maxSimultaneousConnections.<br />Do not modify unless directed by Bindplane support. | 10 | Optional: \{\} <br /> |
| `auditTrail` _[AuditTrailConfig](#audittrailconfig)_ | AuditTrail configures audit trail retention. When omitted, retentionDays defaults to 365. |  | Optional: \{\} <br /> |
| `tsdb` _[TSDBConfig](#tsdbconfig)_ | TSDB configures TLS and remote settings for Bindplane's TSDB integration. |  | Optional: \{\} <br /> |
| `nats` _[NatsConfig](#natsconfig)_ | Nats configures TLS for the NATS event bus (client and server). Cert-manager only. |  | Optional: \{\} <br /> |
| `profiling` _[ProfilingConfig](#profilingconfig)_ | Profiling configures Google Cloud Profiler for Bindplane. When omitted or disabled, profiling is off. |  | Optional: \{\} <br /> |
| `pprof` _[PprofConfig](#pprofconfig)_ | Pprof configures the pprof HTTP server for Bindplane. When omitted or disabled, pprof is off. |  | Optional: \{\} <br /> |
| `eventBus` _[EventBusConfig](#eventbusconfig)_ | EventBus configures the event bus (NATS) integration, including health checks. |  | Optional: \{\} <br /> |
| `analytics` _[AnalyticsConfig](#analyticsconfig)_ | Analytics configures Bindplane analytics reporting. |  | Optional: \{\} <br /> |
| `logging` _[LoggingConfig](#loggingconfig)_ | Logging configures the Bindplane log level and output destination. |  | Optional: \{\} <br /> |
| `advanced` _[AdvancedConfig](#advancedconfig)_ | Advanced configures advanced Bindplane options. These are typically used to<br />fine-tune behavior at scale and are not required for basic operation. |  | Optional: \{\} <br /> |
| `agents` _[AgentsConfig](#agentsconfig)_ | Agents configures Bindplane agent connection, heartbeat, rebalance, and authentication options.<br />When omitted, Bindplane uses its own defaults. |  | Optional: \{\} <br /> |
| `agentVersions` _[AgentVersionsConfig](#agentversionsconfig)_ | AgentVersions configures agent version sync behavior.<br />When omitted, Bindplane uses its own defaults. |  | Optional: \{\} <br /> |


#### BindplaneJobsComponentSpec



BindplaneJobsComponentSpec defines the Bindplane Jobs component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Bindplane Jobs<br />Note: Jobs are restricted to 1 replica and cannot be scaled |  | Type: object <br />Optional: \{\} <br /> |


#### BindplaneJobsMigrateComponentSpec



BindplaneJobsMigrateComponentSpec defines the Bindplane Jobs Migrate component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Bindplane Jobs Migrate<br />Note: Jobs Migrate are restricted to 1 replica and cannot be scaled |  | Type: object <br />Optional: \{\} <br /> |


#### BindplaneSpec



BindplaneSpec defines the desired state of Bindplane.



_Appears in:_
- [Bindplane](#bindplane)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `config` _[BindplaneConfigSpec](#bindplaneconfigspec)_ | Config contains Bindplane's configuration (license, auth, network, store, eventBus)<br />This config is shared by Node, Jobs, and Jobs Migrate |  |  |
| `bindplane` _[BindplaneComponentSpec](#bindplanecomponentspec)_ | Bindplane configuration and pod specification |  |  |
| `bindplaneJobs` _[BindplaneJobsComponentSpec](#bindplanejobscomponentspec)_ | Bindplane Jobs pod specification |  | Optional: \{\} <br /> |
| `bindplaneJobsMigrate` _[BindplaneJobsMigrateComponentSpec](#bindplanejobsmigratecomponentspec)_ | Bindplane Jobs Migrate pod specification |  | Optional: \{\} <br /> |
| `transformAgent` _[TransformAgentComponentSpec](#transformagentcomponentspec)_ | Transform Agent pod specification | \{  \} | Optional: \{\} <br /> |
| `tsdb` _[TSDBComponentSpec](#tsdbcomponentspec)_ | TSDB pod specification |  | Optional: \{\} <br /> |
| `nats` _[NatsComponentSpec](#natscomponentspec)_ | NATS pod specification | \{  \} | Optional: \{\} <br /> |




#### CertManagerTLSIssuerRef



CertManagerTLSIssuerRef references a cert-manager Issuer or ClusterIssuer.
See https://cert-manager.io/docs/concepts/issuer/



_Appears in:_
- [NatsTLSConfig](#natstlsconfig)
- [TSDBTLSConfig](#tsdbtlsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Issuer or ClusterIssuer resource. |  |  |
| `kind` _string_ | Kind is the type of issuer. Either "Issuer" (namespaced) or "ClusterIssuer" (cluster-scoped). | Issuer | Enum: [Issuer ClusterIssuer] <br />Optional: \{\} <br /> |
| `group` _string_ | Group is the API group of the issuer. Defaults to cert-manager.io. | cert-manager.io | Optional: \{\} <br /> |


#### EventBusConfig



EventBusConfig configures the Bindplane event bus (NATS) integration.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `health` _[EventBusHealthConfig](#eventbushealthconfig)_ | Health configures the event bus health check endpoints. |  | Optional: \{\} <br /> |


#### EventBusHealthConfig



EventBusHealthConfig configures the Bindplane event bus health check.
The health check sends an event over NATS and waits for responses from other pods.
Health check failures affect only the status page in the Bindplane web interface;
they do not cause pod shutdown or failure.



_Appears in:_
- [EventBusConfig](#eventbusconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `requiredHosts` _integer_ | RequiredHosts is the minimum number of pods that must respond to the health check<br />event for the event bus to be considered healthy. When omitted, defaults to<br />floor(total / 2) + 1, where total is the sum of node, NATS, jobs, and<br />jobs-migrate replicas. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `interval` _string_ | Interval is how often the event bus health check is performed (e.g. 15s, 1m).<br />When omitted, the Bindplane server default is used. |  | Optional: \{\} <br /> |


#### LDAPConfig



LDAPConfig defines LDAP and Active Directory authentication configuration



_Appears in:_
- [AuthConfig](#authconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `protocol` _string_ | Protocol to use when connecting to the LDAP server. One of: ldap\|ldaps |  | Optional: \{\} <br /> |
| `server` _string_ | Server is the LDAP server hostname |  | Optional: \{\} <br /> |
| `port` _string_ | Port is the LDAP server port |  | Optional: \{\} <br /> |
| `baseDN` _string_ | BaseDN is the base distinguished name for user searches |  | Optional: \{\} <br /> |
| `bindUser` _string_ | BindUser is the username used to bind to the LDAP server |  | Optional: \{\} <br /> |
| `bindUserSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | BindUserSecretRef references a Kubernetes Secret containing the LDAP bind username.<br />Takes precedence over BindUser if both are set. |  | Optional: \{\} <br /> |
| `bindPassword` _string_ | BindPassword is the password used to bind to the LDAP server |  | Optional: \{\} <br /> |
| `bindPasswordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | BindPasswordSecretRef references a Kubernetes Secret containing the LDAP bind password.<br />Takes precedence over BindPassword if both are set. |  | Optional: \{\} <br /> |
| `searchFilter` _string_ | SearchFilter is the LDAP search filter used to locate users |  | Optional: \{\} <br /> |
| `tls` _[LDAPTLSConfig](#ldaptlsconfig)_ | TLS configures TLS for LDAP using a Secret. The operator mounts the Secret and sets<br />BINDPLANE_LDAP_TLS_CERT, BINDPLANE_LDAP_TLS_KEY, and BINDPLANE_LDAP_TLS_CA to the<br />mounted file paths. Omit TLS to disable mutual TLS / custom CA. |  | Optional: \{\} <br /> |
| `tlsSkipVerify` _boolean_ | TLSSkipVerify disables TLS certificate verification |  | Optional: \{\} <br /> |


#### LDAPTLSConfig



LDAPTLSConfig defines TLS for LDAP by referencing a Secret. The Secret is mounted
at a fixed path; the operator sets the TLS env vars to the mounted file paths.
Users specify only the secret name and key names, not mount paths.



_Appears in:_
- [LDAPConfig](#ldapconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA. |  |  |
| `certKey` _string_ | CertKey is the key in the Secret for the TLS certificate (for mutual TLS). |  | Optional: \{\} <br /> |
| `keyKey` _string_ | KeyKey is the key in the Secret for the TLS private key (for mutual TLS). |  | Optional: \{\} <br /> |
| `caKey` _string_ | CAKey is the key in the Secret for the CA certificate. Omit to use system CAs. |  | Optional: \{\} <br /> |


#### LoggingConfig



LoggingConfig defines logging configuration.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `level` _string_ | Level specifies the log level. One of: debug, info, warn, error. | info | Enum: [debug info warn error] <br />Optional: \{\} <br /> |
| `type` _string_ | Type specifies the logging output destination.<br />Use "stdout" to write logs to standard output, "otlp" to export via OTLP,<br />or "stdout,otlp" to write to both simultaneously. | stdout | Pattern: `^(stdout\|otlp)(,(stdout\|otlp))?$` <br />Optional: \{\} <br /> |
| `otlp` _[LoggingOTLPConfig](#loggingotlpconfig)_ | OTLP configures OTLP log export when Type includes otlp. |  | Optional: \{\} <br /> |


#### LoggingOTLPConfig



LoggingOTLPConfig defines OTLP logging configuration.



_Appears in:_
- [LoggingConfig](#loggingconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the gRPC endpoint to send logs to (e.g. localhost:4317). |  | Optional: \{\} <br /> |
| `insecure` _boolean_ | Insecure disables TLS verification for the OTLP connection. |  | Optional: \{\} <br /> |
| `interval` _string_ | Interval is the interval at which to export logs (e.g. 60s).<br />When omitted, Bindplane uses its own default. |  | Optional: \{\} <br /> |


#### MetricsConfig



MetricsConfig defines metrics configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the metrics type. One of: otlp, prometheus. | prometheus | Enum: [otlp prometheus] <br />Optional: \{\} <br /> |
| `interval` _string_ | Interval is the interval at which to export metrics (e.g. 60s). Used when Type is otlp. | 60s | Optional: \{\} <br /> |
| `prometheus` _[MetricsPrometheusConfig](#metricsprometheusconfig)_ | Prometheus configures Prometheus metrics when Type is prometheus. |  | Optional: \{\} <br /> |
| `otlp` _[MetricsOTLPConfig](#metricsotlpconfig)_ | OTLP configures OTLP metrics when Type is otlp. |  | Optional: \{\} <br /> |


#### MetricsOTLPConfig



MetricsOTLPConfig defines OTLP metrics configuration



_Appears in:_
- [MetricsConfig](#metricsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the gRPC endpoint to send metrics to (e.g. localhost:4317). |  | Optional: \{\} <br /> |
| `insecure` _boolean_ | Insecure disables TLS verification for the OTLP connection. |  | Optional: \{\} <br /> |


#### MetricsPrometheusConfig



MetricsPrometheusConfig defines Prometheus metrics configuration



_Appears in:_
- [MetricsConfig](#metricsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the HTTP path to serve metrics on (e.g. /metrics). | /metrics | Optional: \{\} <br /> |
| `username` _string_ | Username is the basic auth username for the metrics endpoint, if any. |  | Optional: \{\} <br /> |
| `password` _string_ | Password is the basic auth password for the metrics endpoint. |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the metrics endpoint password.<br />Takes precedence over Password if both are set. |  | Optional: \{\} <br /> |


#### NatsComponentSpec



NatsComponentSpec defines the NATS component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | Replicas specifies the number of replicas for NATS StatefulSet | 1 | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for NATS |  | Type: object <br />Optional: \{\} <br /> |


#### NatsConfig



NatsConfig configures the NATS event bus (client and server use the same TLS config).



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tls` _[NatsTLSConfig](#natstlsconfig)_ | TLS configures mutual TLS for NATS via cert-manager. When set, a single certificate is used for client, cluster, and HTTP ports. |  | Optional: \{\} <br /> |


#### NatsTLSConfig



NatsTLSConfig defines TLS for NATS. Only cert-manager is supported; no secretName.



_Appears in:_
- [NatsConfig](#natsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certManager` _[CertManagerTLSIssuerRef](#certmanagertlsissuerref)_ | CertManager references a cert-manager Issuer or ClusterIssuer to issue the NATS certificate (used for client, cluster, and HTTP). |  | Optional: \{\} <br /> |


#### NetworkConfig



NetworkConfig defines network configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | Host specifies the bind address |  | Optional: \{\} <br /> |
| `port` _string_ | Port specifies the port to listen on |  | Optional: \{\} <br /> |
| `remoteURL` _string_ | RemoteURL specifies the remote URL for Bindplane.<br />Defaults to http://<bindplane-name>-node:3001 (the internal node service URL).<br />Override this when using ingress, e.g. https://bindplane.my-corp.net |  | Optional: \{\} <br /> |
| `webURL` _string_ | WebURL is the URL used by the client for the web interface. Defaults to RemoteURL when not set. Only set when explicitly configuring. |  | Optional: \{\} <br /> |
| `corsAllowedOrigins` _string_ | CorsAllowedOrigins is the allowed origin for CORS requests. Only set when explicitly configuring. |  | Optional: \{\} <br /> |
| `tls` _[NetworkTLSConfig](#networktlsconfig)_ | TLS configures TLS for the Bindplane server using a Secret. The operator mounts the Secret and sets<br />BINDPLANE_TLS_CERT, BINDPLANE_TLS_KEY, and optionally BINDPLANE_TLS_CA to the mounted file paths.<br />Omit or omit secretName/certKey/keyKey to disable server TLS (e.g. when using Ingress to terminate TLS). |  | Optional: \{\} <br /> |


#### NetworkTLSConfig



NetworkTLSConfig defines TLS for the Bindplane server by referencing a Secret. The Secret is mounted
at a fixed path; the operator sets the TLS env vars to the mounted file paths.
Users specify only the secret name and key names, not mount paths.
Server-side TLS: set secretName, certKey, and keyKey. Mutual TLS: also set caKey.



_Appears in:_
- [NetworkConfig](#networkconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `minVersion` _string_ | MinVersion is the minimum TLS version. One of: 1.2, 1.3. Omit to use server default. |  | Enum: [1.2 1.3] <br />Optional: \{\} <br /> |
| `secretName` _string_ | SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA. |  |  |
| `certKey` _string_ | CertKey is the key in the Secret for the TLS certificate (server or mutual TLS). |  | Optional: \{\} <br /> |
| `keyKey` _string_ | KeyKey is the key in the Secret for the TLS private key (server or mutual TLS). |  | Optional: \{\} <br /> |
| `caKey` _string_ | CAKey is the key in the Secret for the CA certificate. Set for mutual TLS (client cert verification); generally not used. |  | Optional: \{\} <br /> |
| `skipVerify` _boolean_ | SkipVerify disables TLS certificate verification. Only set for testing. |  | Optional: \{\} <br /> |


#### OIDCConfig



OIDCConfig defines OpenID Connect authentication configuration



_Appears in:_
- [AuthConfig](#authconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientID` _string_ | ClientID is the OIDC OAuth2 client ID |  | Optional: \{\} <br /> |
| `clientIDSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | ClientIDSecretRef references a Kubernetes Secret containing the OIDC client ID.<br />Takes precedence over ClientID if both are set. |  | Optional: \{\} <br /> |
| `clientSecret` _string_ | ClientSecret is the OIDC OAuth2 client secret |  | Optional: \{\} <br /> |
| `clientSecretSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | ClientSecretSecretRef references a Kubernetes Secret containing the OIDC client secret.<br />Takes precedence over ClientSecret if both are set. |  | Optional: \{\} <br /> |
| `issuer` _string_ | Issuer is the URL of the OIDC provider |  | Optional: \{\} <br /> |
| `scopes` _string array_ | Scopes is the list of OAuth2 scopes to request |  | Optional: \{\} <br /> |


#### PodTemplateSpec



PodTemplateSpec defines pod template specification.
This embeds corev1.PodTemplateSpec to allow arbitrary pod spec fields.
Note: The operator will merge this with operator-managed fields, ensuring
operator-managed fields (like ServiceAccountName, containers, etc.) take precedence.



_Appears in:_
- [BindplaneComponentSpec](#bindplanecomponentspec)
- [BindplaneJobsComponentSpec](#bindplanejobscomponentspec)
- [BindplaneJobsMigrateComponentSpec](#bindplanejobsmigratecomponentspec)
- [NatsComponentSpec](#natscomponentspec)
- [TSDBComponentSpec](#tsdbcomponentspec)
- [TransformAgentComponentSpec](#transformagentcomponentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[PodSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#podspec-v1-core)_ | Specification of the desired behavior of the pod.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  | Optional: \{\} <br /> |


#### PostgresConfig



PostgresConfig defines PostgreSQL store configuration



_Appears in:_
- [StoreConfig](#storeconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | Host specifies the PostgreSQL host |  |  |
| `port` _string_ | Port specifies the PostgreSQL port |  | Optional: \{\} <br /> |
| `connectTimeout` _string_ | ConnectTimeout specifies the connection timeout |  | Optional: \{\} <br /> |
| `statementTimeout` _string_ | StatementTimeout specifies the statement timeout |  | Optional: \{\} <br /> |
| `database` _string_ | Database specifies the database name |  | Optional: \{\} <br /> |
| `sslmode` _string_ | SSLMode specifies the PostgreSQL SSL mode. One of: disable, require, verify-ca, verify-full. | disable | Enum: [disable require verify-ca verify-full] <br />Optional: \{\} <br /> |
| `tls` _[PostgresTLSConfig](#postgrestlsconfig)_ | TLS configures TLS for PostgreSQL using a Secret. The operator mounts the Secret and sets<br />BINDPLANE_POSTGRES_SSL_ROOT_CERT, BINDPLANE_POSTGRES_SSL_CERT, and BINDPLANE_POSTGRES_SSL_KEY to the<br />mounted file paths. Server-side TLS: set secretName and caKey. Mutual TLS: also set certKey and keyKey. |  | Optional: \{\} <br /> |
| `username` _string_ | Username specifies the PostgreSQL username |  | Optional: \{\} <br /> |
| `usernameSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | UsernameSecretRef references a Kubernetes Secret containing the PostgreSQL username.<br />Takes precedence over Username if both are set. |  | Optional: \{\} <br /> |
| `password` _string_ | Password specifies the PostgreSQL password |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the PostgreSQL password.<br />Takes precedence over Password if both are set. |  | Optional: \{\} <br /> |
| `maxConnections` _integer_ | MaxConnections specifies the maximum number of connections |  | Optional: \{\} <br /> |
| `maxIdleConnections` _integer_ | MaxIdleConnections specifies the maximum number of idle connections. Optional; no default. |  | Optional: \{\} <br /> |
| `maxLifetime` _string_ | MaxLifetime specifies the maximum connection lifetime |  | Optional: \{\} <br /> |
| `maxIdleTime` _string_ | MaxIdleTime specifies the maximum time a connection may remain idle (e.g. 20s, 1m). Optional; no default. |  | Optional: \{\} <br /> |
| `schema` _string_ | Schema specifies the database schema |  | Optional: \{\} <br /> |


#### PostgresTLSConfig



PostgresTLSConfig defines TLS for PostgreSQL by referencing a Secret. The Secret is mounted
at a fixed path; the operator sets the TLS env vars (sslRootCert, sslCert, sslKey) to the mounted file paths.
Users specify only the secret name and key names, not mount paths.
Server-side TLS: set secretName and caKey. Mutual TLS: set secretName, caKey, certKey, and keyKey.



_Appears in:_
- [PostgresConfig](#postgresconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | SecretName is the name of the Secret containing the CA and optionally client cert and key. |  |  |
| `caKey` _string_ | CAKey is the key in the Secret for the root CA (maps to sslRootCert). Required for TLS; enables server-side TLS. |  | Optional: \{\} <br /> |
| `certKey` _string_ | CertKey is the key in the Secret for the client certificate (maps to sslCert). Set with KeyKey for mutual TLS. |  | Optional: \{\} <br /> |
| `keyKey` _string_ | KeyKey is the key in the Secret for the client private key (maps to sslKey). Set with CertKey for mutual TLS. |  | Optional: \{\} <br /> |


#### PprofConfig



PprofConfig configures the pprof HTTP server for Bindplane.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled turns on the pprof server. When false or omitted, pprof is disabled. | false | Optional: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the host:port the pprof server listens on. When unset, defaults to 127.0.0.1:6060. |  | Optional: \{\} <br /> |


#### ProfilingConfig



ProfilingConfig configures Google Cloud Profiler for Bindplane.



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled turns on Google Cloud Profiler. When false or omitted, profiling is disabled. | false | Optional: \{\} <br /> |
| `projectID` _string_ | ProjectID is the GCP project ID. Required when enabled is true. |  | Optional: \{\} <br /> |
| `noCPU` _boolean_ | NoCPU disables CPU profiling. | false | Optional: \{\} <br /> |
| `noAlloc` _boolean_ | NoAlloc disables allocation profiling. | false | Optional: \{\} <br /> |
| `noHeap` _boolean_ | NoHeap disables heap profiling. | false | Optional: \{\} <br /> |
| `noGoroutine` _boolean_ | NoGoroutine disables goroutine profiling. | false | Optional: \{\} <br /> |
| `mutex` _boolean_ | Mutex enables mutex profiling (disabled by default in Bindplane). | false | Optional: \{\} <br /> |




#### StorageSpec



StorageSpec defines persistent storage configuration



_Appears in:_
- [TSDBComponentSpec](#tsdbcomponentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `volumeClaimTemplate` _[VolumeClaimTemplate](#volumeclaimtemplate)_ | VolumeClaimTemplate defines the template for creating PersistentVolumeClaims<br />This follows the same structure as StatefulSet volumeClaimTemplates |  |  |


#### StoreConfig



StoreConfig defines store configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `postgres` _[PostgresConfig](#postgresconfig)_ | Postgres configuration |  |  |
| `maxEvents` _integer_ | MaxEvents is the maximum number of events to merge into a single event.<br />When omitted, Bindplane defaults to 100. |  | Optional: \{\} <br /> |
| `eventMergeWindow` _string_ | EventMergeWindow is the window during which events are merged (e.g. "100ms").<br />When omitted, Bindplane defaults to 100ms. |  | Optional: \{\} <br /> |
| `summaryRollupRetentionDays` _integer_ | SummaryRollupRetentionDays is the number of days to retain daily rollup data.<br />0 means indefinite retention (rollups are never deleted).<br />When omitted, Bindplane defaults to 365. |  | Optional: \{\} <br /> |


#### TSDBComponentSpec



TSDBComponentSpec defines the TSDB component pod specification.
By default, this deploys a Prometheus StatefulSet managed by the operator.



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for the TSDB component |  | Type: object <br />Optional: \{\} <br /> |
| `storage` _[StorageSpec](#storagespec)_ | Storage defines the persistent storage configuration for the TSDB component |  | Optional: \{\} <br /> |
| `tls` _[TSDBTLSConfig](#tsdbtlsconfig)_ | TLS configures TLS for the TSDB server (StatefulSet). Use either secretName (user-defined Secret)<br />or certManager (cert-manager Issuer/ClusterIssuer), not both. When set, the TSDB serves remote write over TLS. |  | Optional: \{\} <br /> |


#### TSDBConfig



TSDBConfig configures Bindplane's TSDB component (default implementation: Prometheus).



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `remote` _[TSDBRemoteConfig](#tsdbremoteconfig)_ | Remote configures Bindplane to use an externally managed TSDB-compatible backend<br />(for example, Prometheus, Mimir, or VictoriaMetrics) instead of the operator-managed TSDB StatefulSet. |  | Optional: \{\} <br /> |
| `tls` _[TSDBTLSConfig](#tsdbtlsconfig)_ | TLS configures TLS for TSDB remote write. |  | Optional: \{\} <br /> |


#### TSDBRemoteConfig



TSDBRemoteConfig defines how Bindplane connects to an externally managed TSDB-compatible backend.



_Appears in:_
- [TSDBConfig](#tsdbconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enable` _boolean_ | Enable controls whether Bindplane should connect to an external TSDB-compatible backend.<br />When false, all other fields in this object must be omitted. |  | Optional: \{\} <br /> |
| `host` _string_ | Host is the hostname or IP of the external TSDB-compatible backend.<br />Required when enable is true. |  | Optional: \{\} <br /> |
| `port` _integer_ | Port is the TCP port of the external TSDB-compatible backend.<br />Required when enable is true. | 9090 | Optional: \{\} <br /> |
| `queryPathPrefix` _string_ | QueryPathPrefix is an optional prefix path for PromQL APIs (for example, /prometheus). |  | Optional: \{\} <br /> |
| `remoteWrite` _[TSDBRemoteWriteConfig](#tsdbremotewriteconfig)_ | RemoteWrite optionally overrides where Bindplane sends TSDB remote write traffic. |  | Optional: \{\} <br /> |


#### TSDBRemoteWriteConfig



TSDBRemoteWriteConfig defines optional remote write endpoint overrides.



_Appears in:_
- [TSDBRemoteConfig](#tsdbremoteconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ | Host is the remote write hostname or IP. Must be set together with port. |  | Optional: \{\} <br /> |
| `port` _integer_ | Port is the remote write TCP port. Must be set together with host. |  | Optional: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the remote write HTTP path. | /api/v1/write | Optional: \{\} <br /> |


#### TSDBTLSConfig



TSDBTLSConfig defines TLS for TSDB remote write.
Exactly one of secretName (user-defined Secret) or certManager (cert-manager Issuer/ClusterIssuer) should be set.



_Appears in:_
- [TSDBComponentSpec](#tsdbcomponentspec)
- [TSDBConfig](#tsdbconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretName` _string_ | SecretName is the name of the Secret containing the TLS certificate, key, and optionally CA (user-defined TLS).<br />Omit when using certManager. |  | Optional: \{\} <br /> |
| `certKey` _string_ | CertKey is the key in the Secret for the TLS certificate. |  | Optional: \{\} <br /> |
| `keyKey` _string_ | KeyKey is the key in the Secret for the TLS private key. |  | Optional: \{\} <br /> |
| `caKey` _string_ | CAKey is the key in the Secret for the CA certificate. |  | Optional: \{\} <br /> |
| `certManager` _[CertManagerTLSIssuerRef](#certmanagertlsissuerref)_ | CertManager references a cert-manager Issuer or ClusterIssuer to issue server and client certs (mTLS).<br />Mutually exclusive with secretName. |  | Optional: \{\} <br /> |
| `skipVerify` _boolean_ | SkipVerify disables TLS certificate verification for the TSDB remote write client. Only set for testing. |  | Optional: \{\} <br /> |


#### TracingConfig



TracingConfig defines tracing configuration



_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the tracing type. One of: otlp, google. When empty, tracing is disabled. |  | Enum: [otlp google] <br />Optional: \{\} <br /> |
| `otlp` _[TracingOTLPConfig](#tracingotlpconfig)_ | OTLP configures OTLP tracing when Type is otlp. |  | Optional: \{\} <br /> |
| `samplingRate` _string_ | SamplingRate is the ratio between 0 and 1 of traces to keep. Omit or 0 to disable sampling. |  | Optional: \{\} <br /> |


#### TracingOTLPConfig



TracingOTLPConfig defines OTLP tracing configuration



_Appears in:_
- [TracingConfig](#tracingconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the OTLP endpoint to send traces to (e.g. http://localhost:4317). |  | Optional: \{\} <br /> |
| `insecure` _boolean_ | Insecure disables TLS verification for the OTLP connection. |  | Optional: \{\} <br /> |


#### TransformAgentComponentSpec



TransformAgentComponentSpec defines the Transform Agent component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `replicas` _integer_ | Replicas specifies the number of replicas for Transform Agent deployment | 2 | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Transform Agent |  | Type: object <br />Optional: \{\} <br /> |


#### VolumeClaimTemplate



VolumeClaimTemplate defines a template for creating PersistentVolumeClaims



_Appears in:_
- [StorageSpec](#storagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#persistentvolumeclaimspec-v1-core)_ | Spec defines the PersistentVolumeClaim specification |  |  |


