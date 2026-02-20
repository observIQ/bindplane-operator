# API Reference

## Packages
- [k8s.bindplane.com/v1alpha1](#k8sbindplanecomv1alpha1)


## k8s.bindplane.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the bindplane v1alpha1 API group.

### Resource Types
- [Bindplane](#bindplane)



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
| `license` _string_ | License is the Bindplane license key |  |  |
| `licenseSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | LicenseSecretRef references a Kubernetes Secret containing the Bindplane license key.<br />Takes precedence over License if both are set. |  | Optional: \{\} <br /> |
| `auth` _[AuthConfig](#authconfig)_ | Auth configuration for Bindplane |  | Optional: \{\} <br /> |
| `network` _[NetworkConfig](#networkconfig)_ | Network configuration for Bindplane |  | Optional: \{\} <br /> |
| `store` _[StoreConfig](#storeconfig)_ | Store configuration for Bindplane |  |  |
| `tracing` _[TracingConfig](#tracingconfig)_ | Tracing configuration for Bindplane. When omitted or type empty, tracing is disabled. |  | Optional: \{\} <br /> |
| `metrics` _[MetricsConfig](#metricsconfig)_ | Metrics configuration for Bindplane. When omitted, defaults to prometheus type with interval 60s and endpoint /metrics. |  | Optional: \{\} <br /> |
| `offline` _boolean_ | Offline enables offline mode for the server. Omit or leave unset to leave offline mode disabled; set only when directed. |  | Optional: \{\} <br /> |
| `maxConcurrency` _integer_ | MaxConcurrency is the maximum number of concurrent OpAMP operations. Do not modify unless directed by Bindplane support. | 10 | Optional: \{\} <br /> |
| `auditTrail` _[AuditTrailConfig](#audittrailconfig)_ | AuditTrail configures audit trail retention. When omitted, retentionDays defaults to 365. |  | Optional: \{\} <br /> |


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
| `prometheus` _[PrometheusComponentSpec](#prometheuscomponentspec)_ | Prometheus pod specification |  | Optional: \{\} <br /> |
| `nats` _[NatsComponentSpec](#natscomponentspec)_ | NATS pod specification | \{  \} | Optional: \{\} <br /> |




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
- [PrometheusComponentSpec](#prometheuscomponentspec)
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
| `sslmode` _string_ | SSLMode specifies the SSL mode |  | Optional: \{\} <br /> |
| `username` _string_ | Username specifies the PostgreSQL username |  | Optional: \{\} <br /> |
| `usernameSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | UsernameSecretRef references a Kubernetes Secret containing the PostgreSQL username.<br />Takes precedence over Username if both are set. |  | Optional: \{\} <br /> |
| `password` _string_ | Password specifies the PostgreSQL password |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#secretkeyselector-v1-core)_ | PasswordSecretRef references a Kubernetes Secret containing the PostgreSQL password.<br />Takes precedence over Password if both are set. |  | Optional: \{\} <br /> |
| `maxConnections` _integer_ | MaxConnections specifies the maximum number of connections |  | Optional: \{\} <br /> |
| `maxLifetime` _string_ | MaxLifetime specifies the maximum connection lifetime |  | Optional: \{\} <br /> |
| `schema` _string_ | Schema specifies the database schema |  | Optional: \{\} <br /> |


#### PrometheusComponentSpec



PrometheusComponentSpec defines the Prometheus component pod specification



_Appears in:_
- [BindplaneSpec](#bindplanespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `podTemplate` _[PodTemplateSpec](#podtemplatespec)_ | PodTemplate defines pod template specification for Prometheus |  | Type: object <br />Optional: \{\} <br /> |
| `storage` _[StorageSpec](#storagespec)_ | Storage defines the persistent storage configuration for Prometheus |  | Optional: \{\} <br /> |


#### StorageSpec



StorageSpec defines persistent storage configuration



_Appears in:_
- [PrometheusComponentSpec](#prometheuscomponentspec)

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


