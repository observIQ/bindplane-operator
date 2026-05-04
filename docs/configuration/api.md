# API Reference

## Packages
- [k8s.bindplane.com/v1alpha1](#k8sbindplanecomv1alpha1)

## k8s.bindplane.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the bindplane v1alpha1 API group.

### Resource Types
- [Bindplane](#bindplane)

#### AdvancedAgentConfig

AdvancedAgentConfig contains advanced agent configuration options.

_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `telemetryPort` _integer_ | TelemetryPort is the port used for agent telemetry collection. Defaults to 8888. |  | Maximum: 65535 <br />Minimum: 1 <br />Optional: \{\} <br /> |

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
| `agent` _[AdvancedAgentConfig](#advancedagentconfig)_ | Agent contains advanced agent telemetry configuration options. |  | Optional: \{\} <br /> |
| `cache` _[AdvancedCacheConfig](#advancedcacheconfig)_ | Cache contains advanced cache configuration options. |  | Optional: \{\} <br /> |
| `rollout` _[AdvancedRolloutConfig](#advancedrolloutconfig)_ | Rollout contains advanced rollout configuration options. |  | Optional: \{\} <br /> |

#### AdvancedRolloutConfig

AdvancedRolloutConfig contains advanced rollout configuration options.

_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `disableUpdater` _boolean_ | DisableUpdater disables the background rollout updater. |  | Optional: \{\} <br /> |
| `retryInterval` _string_ | RetryInterval is the duration to retry configuring pending agents (e.g., "30s"). Defaults to 30s. |  | Optional: \{\} <br /> |
| `updateWorkerCount` _integer_ | UpdateWorkerCount is the number of workers used for updating rollouts. Defaults to 10. |  | Minimum: 1 <br />Optional: \{\} <br /> |

#### AdvancedServerConfig

AdvancedServerConfig contains advanced HTTP/OpAMP server options.

_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxRequestBytes` _integer_ | MaxRequestBytes is the maximum request body size the server accepts, excluding offline<br />agent uploads. Defaults to 10485760 (10 MiB). |  | Optional: \{\} <br /> |
| `shutdownGracePeriod` _string_ | ShutdownGracePeriod is the total time the server waits for in-flight requests and<br />connections to finish before forceful shutdown (e.g. "5m"). Defaults to 2m. |  | Optional: \{\} <br /> |
| `opampShutdownGracePeriodTarget` _string_ | OpAMPShutdownGracePeriodTarget is the fraction (0.1–1.0) of ShutdownGracePeriod<br />that the OpAMP server is given to drain agents (e.g. "0.25"). Defaults to 0.25. |  | Optional: \{\} <br /> |

#### AdvancedStoreConfig

AdvancedStoreConfig contains advanced store configuration options.

_Appears in:_
- [AdvancedConfig](#advancedconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stats` _[AdvancedStoreStatsConfig](#advancedstorestatsconfig)_ | Stats configures advanced measurement storage tuning. |  | Optional: \{\} <br /> |

#### AdvancedStoreStatsConfig

AdvancedStoreStatsConfig tunes measurement pipeline performance.

_Appears in:_
- [AdvancedStoreConfig](#advancedstoreconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `batchFlushInterval` _string_ | BatchFlushInterval is the interval at which to flush measurement batches (e.g. "1s"). |  | Optional: \{\} <br /> |
| `workerCount` _integer_ | WorkerCount is the number of workers saving measurements to the backend. |  | Optional: \{\} <br /> |
| `enableSorting` _boolean_ | EnableSorting enables sorting of metrics by timestamp before saving. |  | Optional: \{\} <br /> |
| `metricChannelSize` _integer_ | MetricChannelSize is the buffer size for the incoming metrics channel. |  | Optional: \{\} <br /> |
| `batchChannelSize` _integer_ | BatchChannelSize is the buffer size for the batch channel between accept and save workers. |  | Optional: \{\} <br /> |

#### AgentDuplicationPreventionConfig

AgentDuplicationPreventionConfig configures duplicate agent connection prevention.

_Appears in:_
- [AgentsConfig](#agentsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enableMiddleware` _boolean_ | EnableMiddleware enables the duplication prevention middleware. |  | Optional: \{\} <br /> |
| `reassignID` _boolean_ | ReassignID reassigns the agent ID when a duplicate is detected. |  | Optional: \{\} <br /> |
| `detectionStrategy` _string_ | DetectionStrategy is the strategy used to detect duplicate connections.<br />Valid values: grace_period. Defaults to grace_period. | grace_period | Enum: [grace_period] <br />Optional: \{\} <br /> |
| `detectionGracePeriod` _string_ | DetectionGracePeriod is how long after the first claim failure to wait before treating<br />subsequent failures as true duplicates (e.g. "3m"). Used with the grace_period strategy.<br />Must be >= connectionRegistryStaleDuration. Defaults to 3m. | 3m | Optional: \{\} <br /> |
| `minGracePeriodFailures` _integer_ | MinGracePeriodFailures is the minimum number of claim failures required before the<br />grace_period strategy confirms a duplicate. Both the time (grace period) and count<br />must be satisfied. Defaults to 3. | 3 | Minimum: 1 <br />Optional: \{\} <br /> |
| `retryAfter` _string_ | RetryAfter is the Retry-After duration sent to the agent when rejecting connections<br />during the grace period (e.g. "30s"). Defaults to 30s. | 30s | Optional: \{\} <br /> |
| `maxReassignmentAttempts` _integer_ | MaxReassignmentAttempts is the maximum number of times an agent can reconnect with<br />the same ID after being assigned a new ID before being permanently rejected (1–10).<br />Defaults to 3. | 3 | Maximum: 10 <br />Minimum: 1 <br />Optional: \{\} <br /> |
| `reassignmentCacheTTL` _string_ | ReassignmentCacheTTL is how long to remember reassignment attempts (e.g. "24h").<br />Maximum 7d. Defaults to 24h. | 24h | Optional: \{\} <br /> |
| `reassignmentRetryAfter` _string_ | ReassignmentRetryAfter is the Retry-After duration sent to an agent when it is<br />permanently rejected after exceeding maxReassignmentAttempts (e.g. "5m").<br />Maximum 1h. Defaults to 5m. | 5m | Optional: \{\} <br /> |
| `enableDuplicateNotifications` _boolean_ | EnableDuplicateNotifications enables sending notifications when duplicate connections are detected. |  | Optional: \{\} <br /> |
| `enablePerOrgEnforcement` _boolean_ | EnablePerOrgEnforcement enables per-organization enforcement of duplicate prevention. |  | Optional: \{\} <br /> |

#### AgentVersionsConfig

AgentVersionsConfig configures how Bindplane syncs agent versions.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `syncInterval` _string_ | SyncInterval is the interval at which to sync agent versions (e.g. "2h").<br />Must be at least 1h. Defaults to 1h. | 1h | Optional: \{\} <br /> |
| `clients` _string array_ | Clients is a deprecated list of version client types (e.g. ["bdot", "github"]).<br />Version clients are now configured per-agent-type via AgentType resources. |  | Optional: \{\} <br /> |

#### AgentsAuthConfig

AgentsAuthConfig configures authentication for agent connections.

_Appears in:_
- [AgentsConfig](#agentsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the authentication method(s) for agent connections.<br />Can be a single method or a comma-separated list (e.g. "oauth,secretKey").<br />Valid values: secretKey, oauth. Defaults to secretKey. | secretKey | Optional: \{\} <br /> |
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
| `cacheTTL` _string_ | CacheTTL is the duration a valid OAuth token is cached (e.g. "1h"). Defaults to 1h. | 1h | Optional: \{\} <br /> |

#### AgentsAuthSecretKeyConfig

AgentsAuthSecretKeyConfig configures secret key authentication for agent connections.

_Appears in:_
- [AgentsAuthConfig](#agentsauthconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `headers` _string array_ | Headers is the list of HTTP headers to read the secret key from.<br />Defaults to ["X-Bindplane-Authorization", "Authorization"]. |  | Optional: \{\} <br /> |

#### AgentsConfig

AgentsConfig configures how Bindplane communicates with agents.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auth` _[AgentsAuthConfig](#agentsauthconfig)_ | Auth configures authentication for agent connections. |  | Optional: \{\} <br /> |
| `heartbeatInterval` _string_ | HeartbeatInterval is the interval on which to perform a heartbeat over agent connections (e.g. "30s").<br />Defaults to 30s. | 30s | Optional: \{\} <br /> |
| `heartbeatTTL` _string_ | HeartbeatTTL is the amount of time between agent-initiated heartbeat messages before an agent<br />connection expires (e.g. "1m"). Must be greater than HeartbeatInterval. Defaults to 1m. | 1m | Optional: \{\} <br /> |
| `heartbeatExpiryInterval` _string_ | HeartbeatExpiryInterval is the interval between reaping expired agents (e.g. "30s").<br />Defaults to 30s. | 30s | Optional: \{\} <br /> |
| `rebalanceInterval` _string_ | RebalanceInterval is the interval between rebalancing agents (e.g. "1h").<br />Defaults to 1h. | 1h | Optional: \{\} <br /> |
| `rebalancePercentage` _integer_ | RebalancePercentage is the percentage of agents to rebalance (0–100).<br />0 disables percentage-based rebalancing. Defaults to 0 (disabled). |  | Maximum: 100 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `rebalanceJitter` _integer_ | RebalanceJitter is the maximum percentage jitter to add to the rebalance interval (0–100).<br />Defaults to 0 (no jitter). |  | Maximum: 100 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `maxSimultaneousConnections` _integer_ | MaxSimultaneousConnections is the maximum number of goroutines that will service<br />OpAMP connections concurrently. Generally set to the same value as<br />spec.config.maxConcurrency. Do not modify unless directed by Bindplane support. | 10 | Optional: \{\} <br /> |
| `enableConnectionRegistryMiddleware` _boolean_ | EnableConnectionRegistryMiddleware enables the connection registry middleware. |  | Optional: \{\} <br /> |
| `connectionRegistryHeartbeatInterval` _string_ | ConnectionRegistryHeartbeatInterval is the interval at which agents send heartbeats to<br />the connection registry (e.g. "15s"). Defaults to 15s. | 15s | Optional: \{\} <br /> |
| `connectionRegistryStaleDuration` _string_ | ConnectionRegistryStaleDuration is the duration after which an agent connection is<br />considered stale if no heartbeat is received (e.g. "45s"). Must be greater than<br />connectionRegistryHeartbeatInterval. Defaults to 45s. | 45s | Optional: \{\} <br /> |
| `connectionRegistryLockTimeout` _string_ | ConnectionRegistryLockTimeout is the PostgreSQL lock_timeout for connection registry<br />operations (e.g. "2s"). Defaults to 2s. | 2s | Optional: \{\} <br /> |
| `connectionClaimTimeout` _string_ | ConnectionClaimTimeout is the overall timeout for ClaimConnection operations (e.g. "3s").<br />Must be greater than connectionRegistryLockTimeout. Defaults to 3s. | 3s | Optional: \{\} <br /> |
| `duplicationPrevention` _[AgentDuplicationPreventionConfig](#agentduplicationpreventionconfig)_ | DuplicationPrevention configures duplicate agent connection prevention. |  | Optional: \{\} <br /> |

#### AnalyticsConfig

AnalyticsConfig configures Bindplane analytics reporting.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `disabled` _boolean_ | Disabled turns off analytics reporting. When false or omitted, analytics are enabled.<br />Free licenses do not support disabling analytics; this option is ignored for that license type. | false | Optional: \{\} <br /> |
| `segmentWriteKey` _string_ | SegmentWriteKey overrides the default Segment write key used for analytics.<br />Do not set unless directed by Bindplane support. |  | Optional: \{\} <br /> |

#### AnthropicConfig

AnthropicConfig configures Anthropic integration.

_Appears in:_
- [LLMConfig](#llmconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _string_ | APIKey is the Anthropic API key (plain value; prefer apiKeySecretRef in production). |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | APIKeySecretRef references a Kubernetes Secret containing the Anthropic API key. |  | Optional: \{\} <br /> |

#### AuditTrailConfig

AuditTrailConfig defines audit trail configuration

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `retentionDays` _integer_ | RetentionDays is the number of days to retain audit trail events. | 365 | Optional: \{\} <br /> |

#### Auth0Config

Auth0Config configures Auth0 as the Bindplane authentication provider.

_Appears in:_
- [AuthConfig](#authconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientID` _string_ | ClientID is the Auth0 application client ID. |  | Optional: \{\} <br /> |
| `domain` _string_ | Domain is the Auth0 tenant domain (e.g. "example.auth0.com"). |  | Optional: \{\} <br /> |
| `audience` _string_ | Audience is the Auth0 API audience identifier. |  | Optional: \{\} <br /> |
| `managementDomain` _string_ | ManagementDomain is the Auth0 management API domain. |  | Optional: \{\} <br /> |
| `managementClientID` _string_ | ManagementClientID is the client ID for the Auth0 management API application. |  | Optional: \{\} <br /> |
| `managementClientSecret` _string_ | ManagementClientSecret is a plain-text client secret for the Auth0 management API. |  | Optional: \{\} <br /> |
| `managementClientSecretSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | ManagementClientSecretSecretRef references a Secret containing the management API client secret.<br />Takes precedence over ManagementClientSecret when both are set. |  | Optional: \{\} <br /> |
| `sso` _[Auth0SSOConfig](#auth0ssoconfig)_ | SSO configures Auth0 single sign-on settings. |  | Optional: \{\} <br /> |
| `wif` _[Auth0WIFConfig](#auth0wifconfig)_ | WIF configures Workload Identity Federation for Auth0. |  | Optional: \{\} <br /> |

#### Auth0SSOConfig

Auth0SSOConfig configures Auth0 SSO settings.

_Appears in:_
- [Auth0Config](#auth0config)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled toggles Auth0 SSO support. |  | Optional: \{\} <br /> |
| `selfServiceProfileID` _string_ | SelfServiceProfileID is the Auth0 self-service SSO profile identifier. |  | Optional: \{\} <br /> |

#### Auth0WIFConfig

Auth0WIFConfig configures Workload Identity Federation for Auth0.

_Appears in:_
- [Auth0Config](#auth0config)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientID` _string_ | ClientID is the WIF client ID. |  | Optional: \{\} <br /> |
| `clientSecret` _string_ | ClientSecret is a plain-text WIF client secret. |  | Optional: \{\} <br /> |
| `clientSecretSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | ClientSecretSecretRef references a Secret containing the WIF client secret.<br />Takes precedence over ClientSecret when both are set. |  | Optional: \{\} <br /> |
| `audience` _string_ | Audience is the WIF audience identifier. |  | Optional: \{\} <br /> |

#### AuthConfig

AuthConfig defines authentication configuration

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the authentication type. |  | Enum: [system ldap active-directory oidc auth0] <br />Optional: \{\} <br /> |
| `username` _string_ | Username for authentication |  | Optional: \{\} <br /> |
| `usernameSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | UsernameSecretRef references a Kubernetes Secret containing the auth username.<br />Takes precedence over Username if both are set. |  | Optional: \{\} <br /> |
| `password` _string_ | Password for authentication |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the auth password.<br />Takes precedence over Password if both are set. |  | Optional: \{\} <br /> |
| `sessionSecret` _string_ | SessionSecret is a plain-text secret used to sign session cookies. |  | Optional: \{\} <br /> |
| `sessionSecretSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | SessionSecretSecretRef references a Secret containing the session secret.<br />Takes precedence over SessionSecret when both are set. |  | Optional: \{\} <br /> |
| `apiKey` _string_ | APIKey is a plain-text API key for programmatic access. |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | APIKeySecretRef references a Secret containing the API key.<br />Takes precedence over APIKey when both are set. |  | Optional: \{\} <br /> |
| `sessionsStrictMode` _boolean_ | SessionsStrictMode enables strict mode for session cookies. |  | Optional: \{\} <br /> |
| `ldap` _[LDAPConfig](#ldapconfig)_ | LDAP is the configuration for ldap or active-directory auth types. |  | Optional: \{\} <br /> |
| `oidc` _[OIDCConfig](#oidcconfig)_ | OIDC is the configuration for the oidc auth type. |  | Optional: \{\} <br /> |
| `auth0` _[Auth0Config](#auth0config)_ | Auth0 is the configuration for the auth0 auth type. |  | Optional: \{\} <br /> |

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
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the Bindplane Node primary container.<br />If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Bindplane Node |  | Type: object <br />Optional: \{\} <br /> |
| `disablePodDisruptionBudget` _boolean_ | DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.<br />When false (default), the operator creates a PDB with minAvailable: 1. | false | Optional: \{\} <br /> |
| `minReadySeconds` _integer_ | MinReadySeconds is the minimum number of seconds a newly created Node pod must be<br />ready (passing its readiness probe) before it is considered available. During a<br />rolling update the next pod is not replaced until this window elapses. When omitted,<br />the operator defaults this to the pod's termination grace period, giving agents<br />that were connected to the outgoing pod enough time to reconnect to healthy nodes<br />(including the new pod) before another pod is taken out of service. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `strategy` _[DeploymentStrategy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#deploymentstrategy-v1-apps)_ | Strategy defines the rollout strategy for the Bindplane Node Deployment.<br />When omitted, defaults to RollingUpdate with maxSurge=1 and maxUnavailable=0,<br />ensuring a replacement pod is running before the old pod is removed. |  | Optional: \{\} <br /> |
| `autoscaling` _[NodeAutoscalingSpec](#nodeautoscalingspec)_ | Autoscaling configures optional horizontal pod autoscaling for Bindplane Node.<br />When autoscaling is enabled, spec.bindplane.replicas is ignored and the<br />HorizontalPodAutoscaler controls the replica count. |  | Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

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
| `maxConcurrency` _integer_ | MaxConcurrency is the maximum number of concurrent OpAMP operations.<br />Generally set to the same value as spec.config.agents.maxSimultaneousConnections.<br />Do not modify unless directed by Bindplane support. |  | Optional: \{\} <br /> |
| `auditTrail` _[AuditTrailConfig](#audittrailconfig)_ | AuditTrail configures audit trail retention. When omitted, retentionDays defaults to 365. |  | Optional: \{\} <br /> |
| `tsdb` _[TSDBConfig](#tsdbconfig)_ | TSDB configures TLS and remote settings for Bindplane's TSDB integration. |  | Optional: \{\} <br /> |
| `nats` _[NatsConfig](#natsconfig)_ | Nats configures TLS for the NATS event bus (client and server). Cert-manager only. |  | Optional: \{\} <br /> |
| `profiling` _[ProfilingConfig](#profilingconfig)_ | Profiling configures Google Cloud Profiler for Bindplane. When omitted or disabled, profiling is off. |  | Optional: \{\} <br /> |
| `pprof` _[PprofConfig](#pprofconfig)_ | Pprof configures the pprof HTTP server for Bindplane. When omitted or disabled, pprof is off. |  | Optional: \{\} <br /> |
| `eventBus` _[EventBusConfig](#eventbusconfig)_ | EventBus configures the event bus (NATS) integration, including health checks. |  | Optional: \{\} <br /> |
| `analytics` _[AnalyticsConfig](#analyticsconfig)_ | Analytics configures Bindplane analytics reporting. |  | Optional: \{\} <br /> |
| `logging` _[LoggingConfig](#loggingconfig)_ | Logging configures the Bindplane log level and output destination. |  | Optional: \{\} <br /> |
| `advanced` _[AdvancedConfig](#advancedconfig)_ | Advanced configures advanced Bindplane options. These are typically used to<br />fine-tune behavior at scale and are not required for basic operation. |  | Optional: \{\} <br /> |
| `agents` _[AgentsConfig](#agentsconfig)_ | Agents configures Bindplane agent connection, heartbeat, rebalance, and authentication options. |  | Optional: \{\} <br /> |
| `agentVersions` _[AgentVersionsConfig](#agentversionsconfig)_ | AgentVersions configures agent version sync behavior. |  | Optional: \{\} <br /> |
| `saas` _[SaaSConfig](#saasconfig)_ | SaaS configures Bindplane SaaS-specific functionality including the license server and Stripe billing. |  | Optional: \{\} <br /> |
| `encryptionProvider` _[EncryptionProviderConfig](#encryptionproviderconfig)_ | EncryptionProvider configures the encryption provider for at-rest encryption of sensitive store data.<br />When omitted, Bindplane uses its built-in encryption. |  | Optional: \{\} <br /> |
| `features` _[FeaturesConfig](#featuresconfig)_ | Features configures the feature flag backend and feature overrides. |  | Optional: \{\} <br /> |
| `errors` _[ErrorsConfig](#errorsconfig)_ | Errors configures error tracking (e.g., BetterStack, Sentry).<br />When omitted, error tracking is disabled. |  | Optional: \{\} <br /> |
| `llm` _[LLMConfig](#llmconfig)_ | LLM configures large language model integrations.<br />When omitted, LLM features are disabled. |  | Optional: \{\} <br /> |
| `quotas` _[QuotasConfig](#quotasconfig)_ | Quotas configures usage quota enforcement. |  | Optional: \{\} <br /> |

#### BindplaneJobsComponentSpec

BindplaneJobsComponentSpec defines the Bindplane Jobs component pod specification

_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the Bindplane Jobs primary container.<br />If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Bindplane Jobs<br />Note: Jobs are restricted to 1 replica and cannot be scaled |  | Type: object <br />Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

#### BindplaneJobsMigrateComponentSpec

BindplaneJobsMigrateComponentSpec defines the Bindplane Jobs Migrate component pod specification.
Jobs Migrate runs as a Kubernetes batch/v1 Job that performs database migrations at install time
and whenever the Bindplane image version changes.

_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the Bindplane Jobs Migrate primary container.<br />If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for the Bindplane Jobs Migrate batch/v1 Job |  | Type: object <br />Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

#### BindplaneSpec

BindplaneSpec defines the desired state of Bindplane.

_Appears in:_
- [Bindplane](#bindplane)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `version` _string_ | Version specifies the Bindplane release version used for all component container images.<br />Changing this value triggers a rolling update of all Bindplane workloads and a new<br />database migration Job before downstream workloads are updated. | 1.99.1 | Optional: \{\} <br /> |
| `config` _[BindplaneConfigSpec](#bindplaneconfigspec)_ | Config contains Bindplane's configuration (license, auth, network, store, eventBus)<br />This config is shared by Node, Jobs, and Jobs Migrate |  |  |
| `bindplane` _[BindplaneComponentSpec](#bindplanecomponentspec)_ | Bindplane configuration and pod specification | \{  \} | Optional: \{\} <br /> |
| `bindplaneJobs` _[BindplaneJobsComponentSpec](#bindplanejobscomponentspec)_ | Bindplane Jobs pod specification |  | Optional: \{\} <br /> |
| `bindplaneJobsMigrate` _[BindplaneJobsMigrateComponentSpec](#bindplanejobsmigratecomponentspec)_ | Bindplane Jobs Migrate pod specification |  | Optional: \{\} <br /> |
| `transformAgent` _[TransformAgentComponentSpec](#transformagentcomponentspec)_ | Transform Agent pod specification | \{  \} | Optional: \{\} <br /> |
| `tsdb` _[TSDBComponentSpec](#tsdbcomponentspec)_ | TSDB pod specification |  | Optional: \{\} <br /> |
| `nats` _[NatsComponentSpec](#natscomponentspec)_ | NATS pod specification | \{  \} | Optional: \{\} <br /> |
| `opamp` _[OpAMPComponentSpec](#opampcomponentspec)_ | OpAMP, when enabled, runs a dedicated Deployment for OpAMP/agent traffic<br />alongside the primary Node deployment. When nil or disabled (the default),<br />the primary Node deployment serves both frontend and OpAMP traffic. |  | Optional: \{\} <br /> |

#### CertManagerTLSIssuerRef

CertManagerTLSIssuerRef references a cert-manager Issuer or ClusterIssuer.
See https://cert-manager.io/docs/concepts/issuer/

_Appears in:_
- [NatsTLSConfig](#natstlsconfig)
- [TSDBTLSConfig](#tsdbtlsconfig)
- [TransformAgentTLSConfig](#transformagenttlsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Issuer or ClusterIssuer resource. |  |  |
| `kind` _string_ | Kind is the type of issuer. Either "Issuer" (namespaced) or "ClusterIssuer" (cluster-scoped). | Issuer | Enum: [Issuer ClusterIssuer] <br />Optional: \{\} <br /> |
| `group` _string_ | Group is the API group of the issuer. Defaults to cert-manager.io. | cert-manager.io | Optional: \{\} <br /> |

#### EncryptionProviderCacheConfig

EncryptionProviderCacheConfig configures the in-memory cache for encryption keys.

_Appears in:_
- [EncryptionProviderConfig](#encryptionproviderconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `capacity` _integer_ | Capacity is the maximum number of entries in the encryption key cache. Defaults to 2000. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `cacheTimeout` _string_ | CacheTimeout is the TTL for cached encryption keys (e.g., "2m"). Defaults to 2m. |  | Optional: \{\} <br /> |

#### EncryptionProviderConfig

EncryptionProviderConfig configures the encryption provider for Bindplane.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the encryption provider type. |  | Enum: [googleKMS] <br />Optional: \{\} <br /> |
| `googleKMS` _[GoogleKMSConfig](#googlekmsconfig)_ | GoogleKMS configures Google Cloud KMS as the encryption provider. |  | Optional: \{\} <br /> |
| `cache` _[EncryptionProviderCacheConfig](#encryptionprovidercacheconfig)_ | Cache configures the in-memory cache for encryption keys. |  | Optional: \{\} <br /> |

#### ErrorsConfig

ErrorsConfig configures error tracking for Bindplane.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables error tracking. |  | Optional: \{\} <br /> |
| `backendDSN` _string_ | BackendDSN is the DSN (Data Source Name) for the backend error tracking service. |  | Optional: \{\} <br /> |
| `frontendDSN` _string_ | FrontendDSN is the DSN for the frontend error tracking service. |  | Optional: \{\} <br /> |
| `environment` _string_ | Environment is the deployment environment name reported to the error tracking service (e.g., "production", "staging"). |  | Optional: \{\} <br /> |
| `release` _string_ | Release is the release version reported to the error tracking service (e.g., "1.2.3"). |  | Optional: \{\} <br /> |
| `tracesSampleRate` _string_ | TracesSampleRate is the fraction of transactions to send for performance monitoring (0.0 to 1.0). |  | Optional: \{\} <br /> |
| `debug` _boolean_ | Debug enables debug mode for the error tracking SDK. |  | Optional: \{\} <br /> |

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
| `requiredHosts` _integer_ | RequiredHosts is the minimum number of pods that must respond to the health check<br />event for the event bus to be considered healthy. When omitted, defaults to<br />floor(total / 2) + 1, where total is the sum of node, NATS, and jobs replicas.<br />Jobs Migrate is a batch/v1 Job (not a long-running pod) and is excluded from this total. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `interval` _string_ | Interval is how often the event bus health check is performed (e.g. 15s, 1m).<br />When omitted, the Bindplane server default is used. |  | Optional: \{\} <br /> |

#### FeatureOverridesConfig

FeatureOverridesConfig forces specific features on or off regardless of the flag backend.

_Appears in:_
- [FeaturesConfig](#featuresconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `growthLicense` _boolean_ | GrowthLicense forces the growth license feature on. |  | Optional: \{\} <br /> |
| `secopsTheme` _boolean_ | SecopsTheme forces the SecOps theme on. |  | Optional: \{\} <br /> |
| `secopsIntegration` _boolean_ | SecopsIntegration forces the SecOps integration feature on. |  | Optional: \{\} <br /> |
| `secopsGcsIntegration` _boolean_ | SecopsGcsIntegration forces the SecOps GCS integration feature on. |  | Optional: \{\} <br /> |
| `llmFeatures` _boolean_ | LLMFeatures forces LLM features on. |  | Optional: \{\} <br /> |
| `pipelineIntelligence` _boolean_ | PipelineIntelligence forces the pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `snapshotPipelineIntelligence` _boolean_ | SnapshotPipelineIntelligence forces the snapshot pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceSnapshotLogTypes` _boolean_ | PipelineIntelligenceSnapshotLogTypes forces the pipeline intelligence snapshot log types feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceOtelConfigImport` _boolean_ | PipelineIntelligenceOtelConfigImport forces the OpenTelemetry config import pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceChronicleForwarderConfigImport` _boolean_ | PipelineIntelligenceChronicleForwarderConfigImport forces the Chronicle Forwarder config import feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceSplunkConfigImport` _boolean_ | PipelineIntelligenceSplunkConfigImport forces the Splunk config import pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceParseField` _boolean_ | PipelineIntelligenceParseField forces the parse field pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `pipelineIntelligenceGenerateProcessors` _boolean_ | PipelineIntelligenceGenerateProcessors forces the generate processors pipeline intelligence feature on. |  | Optional: \{\} <br /> |
| `rawConfigLegacy` _boolean_ | RawConfigLegacy forces the raw config legacy feature on. |  | Optional: \{\} <br /> |
| `rawLogMetricViews` _boolean_ | RawLogMetricViews is deprecated and ignored; kept for backwards compatibility. |  | Optional: \{\} <br /> |
| `notifications` _boolean_ | Notifications forces the notifications feature on. |  | Optional: \{\} <br /> |
| `vault` _boolean_ | Vault forces the vault feature on. |  | Optional: \{\} <br /> |
| `auth0SSO` _boolean_ | Auth0SSO forces the Auth0 SSO feature on. |  | Optional: \{\} <br /> |
| `aixPlatform` _boolean_ | AixPlatform forces the AIX platform feature on. |  | Optional: \{\} <br /> |
| `advancedPipelineEditor` _boolean_ | AdvancedPipelineEditor forces the advanced pipeline editor feature on. |  | Optional: \{\} <br /> |
| `identityTablesDualWrite` _boolean_ | IdentityTablesDualWrite forces the identity tables dual write feature on. |  | Optional: \{\} <br /> |
| `identityTablesCutover` _boolean_ | IdentityTablesCutover forces the identity tables cutover feature on. |  | Optional: \{\} <br /> |
| `v2Configuration` _boolean_ | V2Configuration is deprecated and kept for backwards compatibility. |  | Optional: \{\} <br /> |
| `v2Connectors` _boolean_ | V2Connectors is deprecated and kept for backwards compatibility. |  | Optional: \{\} <br /> |
| `bindplaneBlueprints` _boolean_ | BindplaneBlueprints is deprecated and kept for backwards compatibility. |  | Optional: \{\} <br /> |
| `fleets` _boolean_ | Fleets is deprecated and kept for backwards compatibility. |  | Optional: \{\} <br /> |

#### FeaturesConfig

FeaturesConfig configures the feature flag backend and feature overrides.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the feature flag backend type. |  | Enum: [posthog] <br />Optional: \{\} <br /> |
| `postHog` _[PostHogConfig](#posthogconfig)_ | PostHog configures PostHog as the feature flag backend. |  | Optional: \{\} <br /> |
| `overrides` _[FeatureOverridesConfig](#featureoverridesconfig)_ | Overrides configures feature flag overrides that force specific features on or off. |  | Optional: \{\} <br /> |

#### GeminiConfig

GeminiConfig configures Google Gemini as an LLM backend.

_Appears in:_
- [LLMConfig](#llmconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectID` _string_ | ProjectID is the Google Cloud project ID. |  | Optional: \{\} <br /> |
| `location` _string_ | Location is the Google Cloud region for the Gemini API (e.g., "us-central1"). |  | Optional: \{\} <br /> |
| `credentialsFile` _string_ | CredentialsFile is the path to the Google Cloud credentials file.<br />When running on GKE with Workload Identity, this is typically not needed. |  | Optional: \{\} <br /> |
| `maxTokens` _integer_ | MaxTokens is the maximum number of tokens for responses from the Gemini API.<br />When omitted or 0, the Gemini API default is used. |  | Optional: \{\} <br /> |
| `vectorSearchRedis` _[VectorSearchRedisConfig](#vectorsearchredisconfig)_ | VectorSearchRedis configures Redis for Gemini vector search. |  | Optional: \{\} <br /> |

#### GoogleKMSConfig

GoogleKMSConfig configures Google Cloud KMS as the encryption provider.

_Appears in:_
- [EncryptionProviderConfig](#encryptionproviderconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectID` _string_ | ProjectID is the Google Cloud project ID that contains the KMS key ring. |  | Optional: \{\} <br /> |
| `location` _string_ | Location is the Google Cloud region or zone for the KMS key ring (e.g., "us-central1"). |  | Optional: \{\} <br /> |
| `keyRotationPeriod` _string_ | KeyRotationPeriod is the rotation period for KMS keys (e.g., "30d"). |  | Optional: \{\} <br /> |
| `keyDeletionJob` _boolean_ | KeyDeletionJob enables the KMS key deletion job on the Jobs Migrate workload only.<br />When true, BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_KEY_DELETION_JOB is set on Jobs Migrate. |  | Optional: \{\} <br /> |

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

#### LLMConfig

LLMConfig configures large language model integrations for Bindplane.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `gemini` _[GeminiConfig](#geminiconfig)_ | Gemini configures Google Gemini integration. |  | Optional: \{\} <br /> |
| `langsmith` _[LangsmithConfig](#langsmithconfig)_ | Langsmith configures LangSmith tracing for LLM calls. |  | Optional: \{\} <br /> |
| `openai` _[OpenAIConfig](#openaiconfig)_ | OpenAI configures OpenAI integration. |  | Optional: \{\} <br /> |
| `anthropic` _[AnthropicConfig](#anthropicconfig)_ | Anthropic configures Anthropic integration. |  | Optional: \{\} <br /> |

#### LangsmithConfig

LangsmithConfig configures LangSmith LLM call tracing.

_Appears in:_
- [LLMConfig](#llmconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables LangSmith tracing. |  | Optional: \{\} <br /> |
| `apiKey` _string_ | APIKey is the LangSmith API key (plain value; prefer apiKeySecretRef in production). |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | APIKeySecretRef references a Kubernetes Secret containing the LangSmith API key. |  | Optional: \{\} <br /> |
| `projectName` _string_ | ProjectName is the LangSmith project name for tracing. |  | Optional: \{\} <br /> |
| `url` _string_ | URL is the LangSmith API URL (e.g., "https://api.smith.langchain.com/api/v1"). |  | Optional: \{\} <br /> |
| `sanitizeContent` _boolean_ | SanitizeContent controls whether input/output content is sanitized before sending to LangSmith<br />for privacy compliance. Defaults to true when omitted. |  | Optional: \{\} <br /> |
| `tags` _string array_ | Tags is the list of default tags to apply to all LLM runs in LangSmith. |  | Optional: \{\} <br /> |

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
| `interval` _string_ | Interval is the interval at which to export logs (e.g. "1s"). Defaults to 1s. |  | Optional: \{\} <br /> |

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
| `replicas` _integer_ | Replicas specifies the number of replicas for NATS StatefulSet | 2 | Optional: \{\} <br /> |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the NATS primary container.<br />If podTemplate.spec.containers[server].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for NATS |  | Type: object <br />Optional: \{\} <br /> |
| `disablePodDisruptionBudget` _boolean_ | DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.<br />When false (default), the operator creates a PDB with minAvailable: 1. |  | Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

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
| `skipVerify` _boolean_ | SkipVerify disables TLS certificate verification for NATS connections.<br />Not recommended for production use. |  | Optional: \{\} <br /> |

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
| `rateLimits` _[NetworkRateLimitsConfig](#networkratelimitsconfig)_ | RateLimits configures per-endpoint rate limiting for the REST API and GraphQL API. |  | Optional: \{\} <br /> |

#### NetworkRateLimitsConfig

NetworkRateLimitsConfig configures rate limiting for the Bindplane REST and GraphQL APIs.

_Appears in:_
- [NetworkConfig](#networkconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiRate` _string_ | APIRate is the maximum number of REST API requests per second (e.g. "10").<br />Bindplane reads this as a float64. |  | Optional: \{\} <br /> |
| `apiBurst` _integer_ | APIBurst is the maximum burst size for REST API requests. |  | Optional: \{\} <br /> |
| `graphqlRate` _string_ | GraphQLRate is the maximum number of GraphQL API requests per second (e.g. "10").<br />Bindplane reads this as a float64. |  | Optional: \{\} <br /> |
| `graphqlBurst` _integer_ | GraphQLBurst is the maximum burst size for GraphQL API requests. |  | Optional: \{\} <br /> |

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

#### NodeAutoscalingSpec

NodeAutoscalingSpec configures horizontal pod autoscaling for Bindplane Node.
When enabled, the operator creates a HorizontalPodAutoscaler and the
spec.bindplane.replicas field is ignored — the HPA controls replica count.

All fields are optional. Omitted fields use defaults tuned for Bindplane Node's
stateful WebSocket (OpAMP) workload.

_Appears in:_
- [BindplaneComponentSpec](#bindplanecomponentspec)
- [OpAMPComponentSpec](#opampcomponentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables the HorizontalPodAutoscaler for Bindplane Node.<br />When false (the default), static replica counts from spec.bindplane.replicas<br />are used and no HPA is created. | false | Optional: \{\} <br /> |
| `minReplicas` _integer_ | MinReplicas is the lower replica bound for the autoscaler. Default: 2. | 2 | Minimum: 1 <br />Optional: \{\} <br /> |
| `maxReplicas` _integer_ | MaxReplicas is the upper replica bound for the autoscaler. Default: 10. | 10 | Minimum: 1 <br />Optional: \{\} <br /> |
| `metrics` _[MetricSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#metricspec-v2-autoscaling) array_ | Metrics contains the specifications for which metrics to use when calculating<br />the desired replica count. When omitted, defaults to CPU at 50% target utilization. |  | Optional: \{\} <br /> |
| `behavior` _[HorizontalPodAutoscalerBehavior](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#horizontalpodautoscalerbehavior-v2-autoscaling)_ | Behavior configures the scaling behavior in both Up and Down directions.<br />When omitted, the default scaleDown policy enforces slow scale-down<br />(1 pod per 5 minutes) to prevent agent reconnection storms. |  | Optional: \{\} <br /> |

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
| `disableInvitations` _boolean_ | DisableInvitations disables the invitation flow for OIDC-authenticated users.<br />When true, users cannot be invited via email and must log in via OIDC directly. |  | Optional: \{\} <br /> |

#### OpAMPComponentSpec

OpAMPComponentSpec defines an optional dedicated Bindplane Deployment that
serves OpAMP/agent traffic. When enabled, the operator provisions a second
Deployment running BINDPLANE_MODE=node alongside the primary Node deployment.
Both Deployments share the same Bindplane configuration (license, store, auth,
event bus). They differ in resources, replicas, autoscaling, PDB, and
OpAMP-specific tuning environment variables.

Use this when you want to scale agent-handling capacity independently from
the frontend (UI/API), for example when you have a large fleet of agents but
modest UI traffic.

_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables the dedicated OpAMP deployment. When false (the default),<br />the primary Node deployment serves both frontend and OpAMP traffic. | false | Optional: \{\} <br /> |
| `replicas` _integer_ | Replicas specifies the number of replicas for the OpAMP deployment.<br />Ignored when Autoscaling.Enabled is true. | 3 | Optional: \{\} <br /> |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the OpAMP<br />primary container. If podTemplate.spec.containers[server].resources is<br />also set, the podTemplate value takes precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for the OpAMP deployment.<br />Merged on top of operator-managed defaults using the same merge rules as<br />other component podTemplates. |  | Type: object <br />Optional: \{\} <br /> |
| `disablePodDisruptionBudget` _boolean_ | DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget<br />for the OpAMP deployment. When false (the default), the operator creates<br />a PDB with minAvailable: 1. | false | Optional: \{\} <br /> |
| `minReadySeconds` _integer_ | MinReadySeconds is the minimum number of seconds a newly created OpAMP pod<br />must be ready before it is considered available. When omitted, the operator<br />defaults this to the pod's termination grace period. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `strategy` _[DeploymentStrategy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#deploymentstrategy-v1-apps)_ | Strategy defines the rollout strategy for the OpAMP Deployment. When<br />omitted, defaults to RollingUpdate with maxSurge=1 and maxUnavailable=0. |  | Optional: \{\} <br /> |
| `autoscaling` _[NodeAutoscalingSpec](#nodeautoscalingspec)_ | Autoscaling configures optional horizontal pod autoscaling for OpAMP.<br />When enabled, spec.bindplane.opamp.replicas is ignored. |  | Optional: \{\} <br /> |
| `maxSimultaneousConnections` _integer_ | MaxSimultaneousConnections sets BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS<br />for the OpAMP deployment only. When unset, falls back to<br />spec.config.agents.maxSimultaneousConnections which is shared<br />across all node-mode Deployments. Useful when you want OpAMP pods to handle<br />a higher concurrency than the frontend pods. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `shutdownGracePeriodTarget` _string_ | ShutdownGracePeriodTarget sets BINDPLANE_ADVANCED_SERVER_OPAMP_SHUTDOWN_GRACE_PERIOD_TARGET<br />for the OpAMP deployment. This is a 0-1 fraction (e.g. "0.6") of the OpAMP<br />shutdown grace period after which the server stops accepting new OpAMP<br />connections. Only applied when set. |  | Optional: \{\} <br /> |

#### OpenAIConfig

OpenAIConfig configures OpenAI integration.

_Appears in:_
- [LLMConfig](#llmconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _string_ | APIKey is the OpenAI API key (plain value; prefer apiKeySecretRef in production). |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | APIKeySecretRef references a Kubernetes Secret containing the OpenAI API key. |  | Optional: \{\} <br /> |

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
- [OpAMPComponentSpec](#opampcomponentspec)
- [TSDBComponentSpec](#tsdbcomponentspec)
- [TransformAgentComponentSpec](#transformagentcomponentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[PodSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#podspec-v1-core)_ | Specification of the desired behavior of the pod.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |  | Optional: \{\} <br /> |

#### PostHogConfig

PostHogConfig configures PostHog as the feature flag backend.

_Appears in:_
- [FeaturesConfig](#featuresconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectAPIKey` _string_ | ProjectAPIKey is the PostHog project API key (plain value; prefer projectAPIKeySecretRef in production). |  | Optional: \{\} <br /> |
| `projectAPIKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | ProjectAPIKeySecretRef references a Kubernetes Secret containing the PostHog project API key. |  | Optional: \{\} <br /> |
| `personalAPIKey` _string_ | PersonalAPIKey is the PostHog personal API key (plain value; prefer personalAPIKeySecretRef in production). |  | Optional: \{\} <br /> |
| `personalAPIKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PersonalAPIKeySecretRef references a Kubernetes Secret containing the PostHog personal API key. |  | Optional: \{\} <br /> |
| `endpoint` _string_ | Endpoint is the PostHog API endpoint URL. |  | Optional: \{\} <br /> |
| `featureFlagRequestTimeout` _string_ | FeatureFlagRequestTimeout is the timeout for PostHog feature flag requests (e.g., "30s"). Defaults to 30s. |  | Optional: \{\} <br /> |
| `defaultFeatureFlagsPollingInterval` _string_ | DefaultFeatureFlagsPollingInterval is the polling interval for feature flags (e.g., "5m"). Defaults to 5m. |  | Optional: \{\} <br /> |

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

#### QuotasConfig

QuotasConfig configures usage quota enforcement.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables quota enforcement globally. |  | Optional: \{\} <br /> |
| `enforced` _boolean_ | Enforced causes quota violations to be enforced (rejected) rather than only logged. |  | Optional: \{\} <br /> |
| `organizations` _[QuotasTierConfig](#quotastierconfig)_ | Organizations configures per-organization quota enforcement. |  | Optional: \{\} <br /> |
| `projects` _[QuotasTierConfig](#quotastierconfig)_ | Projects configures per-project quota enforcement. |  | Optional: \{\} <br /> |

#### QuotasTierConfig

QuotasTierConfig configures quota enforcement for a specific tier (organizations or projects).

_Appears in:_
- [QuotasConfig](#quotasconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled enables quota enforcement for this tier. |  | Optional: \{\} <br /> |
| `enforced` _boolean_ | Enforced causes quota violations for this tier to be enforced (rejected) rather than only logged. |  | Optional: \{\} <br /> |
| `default` _[QuotasTierDefaultConfig](#quotastierdefaultconfig)_ | Default configures default quota limits for this tier. |  | Optional: \{\} <br /> |

#### QuotasTierDefaultConfig

QuotasTierDefaultConfig configures default quota limits for a quota tier.

_Appears in:_
- [QuotasTierConfig](#quotastierconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `maxAgents` _integer_ | MaxAgents is the default maximum number of agents allowed for this tier.<br />When omitted or 0, no default limit is applied. |  | Minimum: 0 <br />Optional: \{\} <br /> |

#### SaaSConfig

SaaSConfig configures Bindplane SaaS-specific functionality.

_Appears in:_
- [BindplaneConfigSpec](#bindplaneconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled toggles SaaS mode. |  | Optional: \{\} <br /> |
| `licenseServerAddress` _string_ | LicenseServerAddress is the URL of the SaaS license server. |  | Optional: \{\} <br /> |
| `licenseServerAPIKey` _string_ | LicenseServerAPIKey is a plain-text API key for the license server. |  | Optional: \{\} <br /> |
| `licenseServerAPIKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | LicenseServerAPIKeySecretRef references a Secret containing the license server API key.<br />Takes precedence over LicenseServerAPIKey when both are set. |  | Optional: \{\} <br /> |
| `useStagePublicRSAKey` _boolean_ | UseStagePublicRSAKey enables use of the staging RSA public key for token validation. |  | Optional: \{\} <br /> |
| `stripe` _[SaaSStripeConfig](#saasstripeconfig)_ | Stripe configures Stripe billing integration. |  | Optional: \{\} <br /> |

#### SaaSStripeConfig

SaaSStripeConfig configures Stripe billing for Bindplane SaaS.

_Appears in:_
- [SaaSConfig](#saasconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled toggles Stripe integration. |  | Optional: \{\} <br /> |
| `secretKey` _string_ | SecretKey is a plain-text Stripe secret key. |  | Optional: \{\} <br /> |
| `secretKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | SecretKeySecretRef references a Secret containing the Stripe secret key.<br />Takes precedence over SecretKey when both are set. |  | Optional: \{\} <br /> |
| `publishableKey` _string_ | PublishableKey is a plain-text Stripe publishable key. |  | Optional: \{\} <br /> |
| `publishableKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PublishableKeySecretRef references a Secret containing the Stripe publishable key.<br />Takes precedence over PublishableKey when both are set. |  | Optional: \{\} <br /> |
| `webhookSecret` _string_ | WebhookSecret is a plain-text Stripe webhook signing secret. |  | Optional: \{\} <br /> |
| `webhookSecretSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | WebhookSecretSecretRef references a Secret containing the Stripe webhook secret.<br />Takes precedence over WebhookSecret when both are set. |  | Optional: \{\} <br /> |
| `growthPlanIDs` _[SaaSStripeGrowthPlanIDsConfig](#saasstripegrowthplanidsconfig)_ | GrowthPlanIDs configures Stripe plan identifiers for the growth tier. |  | Optional: \{\} <br /> |
| `growthPlanMeterNames` _[SaaSStripeGrowthPlanMeterNamesConfig](#saasstripegrowthplanmeternamesconfig)_ | GrowthPlanMeterNames configures Stripe meter names for the growth tier. |  | Optional: \{\} <br /> |
| `meterReportInterval` _string_ | MeterReportInterval is the interval at which usage metrics are reported to Stripe (e.g., "6h"). Defaults to 6h. |  | Optional: \{\} <br /> |

#### SaaSStripeGrowthPlanIDsConfig

SaaSStripeGrowthPlanIDsConfig configures Stripe plan IDs for the growth tier.

_Appears in:_
- [SaaSStripeConfig](#saasstripeconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `baseRate` _string_ | BaseRate is the Stripe price ID for the base rate plan. |  | Optional: \{\} <br /> |
| `usageRates` _string_ | UsageRates is a comma-separated list of Stripe price IDs for usage-based rates. |  | Optional: \{\} <br /> |

#### SaaSStripeGrowthPlanMeterNamesConfig

SaaSStripeGrowthPlanMeterNamesConfig configures Stripe meter names for usage billing.

_Appears in:_
- [SaaSStripeConfig](#saasstripeconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `logs` _string_ | Logs is the Stripe meter name for log volume. |  | Optional: \{\} <br /> |
| `metrics` _string_ | Metrics is the Stripe meter name for metric volume. |  | Optional: \{\} <br /> |
| `traces` _string_ | Traces is the Stripe meter name for trace volume. |  | Optional: \{\} <br /> |
| `collectors` _string_ | Collectors is the Stripe meter name for collector count. |  | Optional: \{\} <br /> |

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
| `maxEvents` _integer_ | MaxEvents is the maximum number of events to merge into a single event. Defaults to 100. | 100 | Optional: \{\} <br /> |
| `eventMergeWindow` _string_ | EventMergeWindow is the window during which events are merged (e.g. "100ms"). Defaults to 100ms. | 100ms | Optional: \{\} <br /> |
| `summaryRollupRetentionDays` _integer_ | SummaryRollupRetentionDays is the number of days to retain daily rollup data.<br />0 means indefinite retention (rollups are never deleted). Defaults to 365. | 365 | Optional: \{\} <br /> |

#### TSDBComponentSpec

TSDBComponentSpec defines the TSDB component pod specification.
By default, this deploys a Prometheus StatefulSet managed by the operator.

_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the TSDB primary container.<br />If podTemplate.spec.containers[tsdb].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for the TSDB component |  | Type: object <br />Optional: \{\} <br /> |
| `storage` _[StorageSpec](#storagespec)_ | Storage defines the persistent storage configuration for the TSDB component |  | Optional: \{\} <br /> |
| `tls` _[TSDBTLSConfig](#tsdbtlsconfig)_ | TLS configures TLS for the TSDB server (StatefulSet). Use either secretName (user-defined Secret)<br />or certManager (cert-manager Issuer/ClusterIssuer), not both. When set, the TSDB serves remote write over TLS. |  | Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

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
| `type` _string_ | Type specifies the tracing type. One of: otlp, google, honeycomb. When empty, tracing is disabled. |  | Enum: [otlp google honeycomb] <br />Optional: \{\} <br /> |
| `otlp` _[TracingOTLPConfig](#tracingotlpconfig)_ | OTLP configures OTLP tracing when Type is otlp. |  | Optional: \{\} <br /> |
| `honeycomb` _[TracingHoneycombConfig](#tracinghoneycombconfig)_ | Honeycomb configures Honeycomb tracing when Type is honeycomb. |  | Optional: \{\} <br /> |
| `samplingRate` _string_ | SamplingRate is the ratio between 0 and 1 of traces to keep. Omit or 0 to disable sampling. |  | Optional: \{\} <br /> |

#### TracingHoneycombConfig

TracingHoneycombConfig defines Honeycomb tracing configuration.

_Appears in:_
- [TracingConfig](#tracingconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiKey` _string_ | APIKey is the Honeycomb API key (plain text). Use APIKeySecretRef instead for sensitive values. |  | Optional: \{\} <br /> |
| `apiKeySecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | APIKeySecretRef references a Kubernetes Secret containing the Honeycomb API key.<br />Takes precedence over APIKey when both are set. |  | Optional: \{\} <br /> |

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
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#resourcerequirements-v1-core)_ | Resources defines compute resource requests and limits for the Transform Agent primary container.<br />If podTemplate.spec.containers[transform-agent].resources is also set, the podTemplate value takes<br />precedence because it is more specific. |  | Optional: \{\} <br /> |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Transform Agent |  | Type: object <br />Optional: \{\} <br /> |
| `tls` _[TransformAgentTLSConfig](#transformagenttlsconfig)_ | TLS configures mutual TLS for the Transform Agent via cert-manager. When set, a single certificate<br />is used for the Transform Agent server and Bindplane clients. |  | Optional: \{\} <br /> |
| `disablePodDisruptionBudget` _boolean_ | DisablePodDisruptionBudget disables the operator-managed PodDisruptionBudget for this component.<br />When false (default), the operator creates a PDB with minAvailable: 1. |  | Optional: \{\} <br /> |
| `extraEnv` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#envvar-v1-core) array_ | ExtraEnv is a list of additional environment variables to inject into the<br />primary container of this component. These are prepended BEFORE the<br />operator-managed environment variables, so a duplicate Name set here will<br />be ignored — Kubernetes uses the LAST entry for a given Name and the<br />operator will not let user entries override its own values.<br />This is the supported way to add custom environment variables. Setting<br />env on podTemplate.spec.containers[<name>] is intentionally ignored.<br />Environment variable names starting with BINDPLANE_ are rejected by the<br />validating webhook unless the operator is started with --allow-bindplane-extra-env=true. |  | Optional: \{\} <br /> |

#### TransformAgentTLSConfig

TransformAgentTLSConfig defines TLS for the Transform Agent. Only cert-manager is supported; no secretName.

_Appears in:_
- [TransformAgentComponentSpec](#transformagentcomponentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certManager` _[CertManagerTLSIssuerRef](#certmanagertlsissuerref)_ | CertManager references a cert-manager Issuer or ClusterIssuer to issue the Transform Agent certificate<br />used by both the Transform Agent server and Bindplane clients. |  | Optional: \{\} <br /> |

#### VectorSearchRedisConfig

VectorSearchRedisConfig configures Redis for vector search.

_Appears in:_
- [GeminiConfig](#geminiconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `address` _string_ | Address is the Redis address (host:port). |  | Optional: \{\} <br /> |
| `enableTLS` _boolean_ | EnableTLS enables TLS for the Redis connection. |  | Optional: \{\} <br /> |

#### VolumeClaimTemplate

VolumeClaimTemplate defines a template for creating PersistentVolumeClaims

_Appears in:_
- [StorageSpec](#storagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[PersistentVolumeClaimSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#persistentvolumeclaimspec-v1-core)_ | Spec defines the PersistentVolumeClaim specification |  |  |

