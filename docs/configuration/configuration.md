# Bindplane Configuration

This document describes **Bindplane configuration**—the `spec.config` field and related Bindplane server settings (license, auth, network, store, tracing, metrics). For the full list of custom resource options (all CRD fields, including component specs and pod templates), see the [API Reference](api.md). The API reference is generated from the CRD; run `make generate-api-docs` to regenerate it. For an overview of TLS and Secret usage (including operator-generated secrets), see [Security: TLS and Secrets](security.md).

Configuration is provided via the `spec.config` field of the `Bindplane` custom resource. The operator translates these fields into environment variables on the Node, NATS, Jobs, and Jobs Migrate workloads.

## Table of contents

- [License](#license)
- [Authentication](#authentication)
  - [System auth](#system-auth)
  - [LDAP and Active Directory](#ldap-and-active-directory)
  - [OIDC](#oidc)
- [Network](#network)
- [Store](#store)
  - [PostgreSQL](#postgresql)
- [Tracing](#tracing)
- [Metrics](#metrics)
- [TSDB](#tsdb)
- [Max concurrency](#max-concurrency)
- [Audit trail](#audit-trail)
- [Event bus](#event-bus)
- [Logging](#logging)
- [Status checks](#status-checks)
- [Agents](#agents)
  - [Authentication](#agents-authentication)
  - [Heartbeat](#heartbeat)
  - [Rebalance](#rebalance)
- [Agent versions](#agent-versions)
- [Extra environment variables](#extra-environment-variables)
  - [Reserved env names](#reserved-env-names)
  - [Pod template vs extraEnv](#pod-template-vs-extraenv)
- [Extra volumes and volume mounts](#extra-volumes-and-volume-mounts)
  - [Allowed volume sources](#allowed-volume-sources)
  - [Reserved volume names and mount paths](#reserved-volume-names-and-mount-paths)
  - [Pod template vs extraVolumeMounts](#pod-template-vs-extravolumemounts)
- [Argo Rollouts (primary Bindplane component)](#argo-rollouts-primary-bindplane-component)
- [OpAMP deployment split](#opamp-deployment-split)
- [Container images](#container-images)
- [Service account annotations](#service-account-annotations)
- [Scope](#scope)
- [Lifecycle](#lifecycle)
  - [Pause annotation](#pause-annotation)
  - [Finalizer and garbage collection](#finalizer-and-garbage-collection)
  - [Conditions and phases](#conditions-and-phases)
  - [Migration contract](#migration-contract)
- [Examples](#examples)
  - [Minimal configuration](#minimal-configuration)

## License

The license key can be set as a direct value or via a Secret reference. Use `licenseSecretRef` with `name` (Secret name) and `key` (key within the Secret). The Secret reference takes precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.license` | `BINDPLANE_LICENSE` | — | Yes |
| `spec.config.licenseSecretRef` | `BINDPLANE_LICENSE` | — | Yes |

Example (direct value):

```yaml
spec:
  config:
    license: "my-license-key"
```

Example (Secret reference):

```yaml
spec:
  config:
    licenseSecretRef:
      name: bindplane-secrets
      key: license
```

## Authentication

Supported auth types: `system`, `ldap`, `active-directory`, `oidc`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.type` | `BINDPLANE_AUTH_TYPE` | — | No |
| `spec.config.auth.sessionsStrictMode` | `BINDPLANE_AUTH_SESSIONS_STRICT_MODE` | `false` | No |
| `spec.config.auth.sessionSecret` | `BINDPLANE_SESSION_SECRET` | — | No |
| `spec.config.auth.sessionSecretSecretRef` | `BINDPLANE_SESSION_SECRET` | — | No |
| `spec.config.auth.apiKey` | `BINDPLANE_SECRET_KEY` | — | No |
| `spec.config.auth.apiKeySecretRef` | `BINDPLANE_SECRET_KEY` | — | No |

`sessionSecretSecretRef` and `apiKeySecretRef` take precedence over plain-value fields when both are set.

### System auth

Set `spec.config.auth.type` to `system` for basic username/password authentication.

Username and password can be set as direct values or via Secret references (`usernameSecretRef`, `passwordSecretRef`). Each uses `name` and `key` to reference a Secret. Secret references take precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.username` | `BINDPLANE_USERNAME` | — | No |
| `spec.config.auth.usernameSecretRef` | `BINDPLANE_USERNAME` | — | No |
| `spec.config.auth.password` | `BINDPLANE_PASSWORD` | — | No |
| `spec.config.auth.passwordSecretRef` | `BINDPLANE_PASSWORD` | — | No |

Example (direct values):

```yaml
spec:
  config:
    auth:
      type: system
      username: admin
      password: "my-password"
```

Example (Secret references):

```yaml
spec:
  config:
    auth:
      type: system
      usernameSecretRef:
        name: bindplane-secrets
        key: auth-username
      passwordSecretRef:
        name: bindplane-secrets
        key: auth-password
```

### LDAP and Active Directory

Set `spec.config.auth.type` to `ldap` or `active-directory`. Both types share the same `ldap` configuration block.

Bind user and bind password can be set as direct values or via Secret references (`bindUserSecretRef`, `bindPasswordSecretRef`). Each uses `name` and `key` to reference a Secret. Secret references take precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.ldap.protocol` | `BINDPLANE_LDAP_PROTOCOL` | — | Yes (`ldap` or `ldaps`) |
| `spec.config.auth.ldap.server` | `BINDPLANE_LDAP_SERVER` | — | Yes |
| `spec.config.auth.ldap.port` | `BINDPLANE_LDAP_PORT` | — | Yes |
| `spec.config.auth.ldap.baseDN` | `BINDPLANE_LDAP_BASE_DN` | — | Yes |
| `spec.config.auth.ldap.bindUser` | `BINDPLANE_LDAP_BIND_USER` | — | No |
| `spec.config.auth.ldap.bindUserSecretRef` | `BINDPLANE_LDAP_BIND_USER` | — | No |
| `spec.config.auth.ldap.bindPassword` | `BINDPLANE_LDAP_BIND_PASSWORD` | — | No |
| `spec.config.auth.ldap.bindPasswordSecretRef` | `BINDPLANE_LDAP_BIND_PASSWORD` | — | No |
| `spec.config.auth.ldap.searchFilter` | `BINDPLANE_LDAP_SEARCH_FILTER` | — | No |
| `spec.config.auth.ldap.tls` | (see below) | — | No |
| `spec.config.auth.ldap.tlsSkipVerify` | `BINDPLANE_LDAP_TLS_SKIP_VERIFY` | `false` | No |

**LDAP TLS:** Specify a TLS CA for TLS verification. Specify a certificate and private key for TLS
with client auth (mutual TLS).

| TLS Field | Description |
|---|---|
| `spec.config.auth.ldap.tls.secretName` | Name of the Secret containing the cert, key, and optionally CA |
| `spec.config.auth.ldap.tls.certKey` | Key in the Secret for the TLS certificate (mutual TLS) |
| `spec.config.auth.ldap.tls.keyKey` | Key in the Secret for the TLS private key (mutual TLS) |
| `spec.config.auth.ldap.tls.caKey` | Key in the Secret for the CA certificate (optional; omit to use system CAs) |

Example (direct values for bind user and password):

```yaml
spec:
  config:
    auth:
      type: ldap
      ldap:
        protocol: ldaps
        server: ldap.example.com
        port: "636"
        baseDN: "dc=example,dc=com"
        bindUser: cn=bindplane,dc=example,dc=com
        bindPassword: "my-bind-password"
        tls:
          secretName: ldap-tls-secret
          certKey: tls.crt
          keyKey: tls.key
          caKey: ca.crt
```

Example (Secret references for bind user and password):

```yaml
spec:
  config:
    auth:
      type: ldap
      ldap:
        protocol: ldaps
        server: ldap.example.com
        port: "636"
        baseDN: "dc=example,dc=com"
        bindUserSecretRef:
          name: ldap-secrets
          key: bind-user
        bindPasswordSecretRef:
          name: ldap-secrets
          key: bind-password
        tls:
          secretName: ldap-tls-secret
          certKey: tls.crt
          keyKey: tls.key
          caKey: ca.crt
```

Example (Active Directory with Secret references):

```yaml
spec:
  config:
    auth:
      type: active-directory
      ldap:
        protocol: ldap
        server: ad.example.com
        port: "389"
        baseDN: "dc=example,dc=com"
        bindUserSecretRef:
          name: ad-secrets
          key: bind-user
        bindPasswordSecretRef:
          name: ad-secrets
          key: bind-password
```

### OIDC

Set `spec.config.auth.type` to `oidc`.

Client ID and client secret can be set as direct values or via Secret references (`clientIDSecretRef`, `clientSecretSecretRef`). Each uses `name` and `key` to reference a Secret. Secret references take precedence when both are set. Prefer Secret references in production.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.oidc.clientID` | `BINDPLANE_OIDC_OAUTH2_CLIENT_ID` | — | Yes |
| `spec.config.auth.oidc.clientIDSecretRef` | `BINDPLANE_OIDC_OAUTH2_CLIENT_ID` | — | Yes |
| `spec.config.auth.oidc.clientSecret` | `BINDPLANE_OIDC_OAUTH2_CLIENT_SECRET` | — | Yes |
| `spec.config.auth.oidc.clientSecretSecretRef` | `BINDPLANE_OIDC_OAUTH2_CLIENT_SECRET` | — | Yes |
| `spec.config.auth.oidc.issuer` | `BINDPLANE_OIDC_ISSUER` | — | Yes |
| `spec.config.auth.oidc.scopes` | `BINDPLANE_OIDC_SCOPES` | — | Yes (comma-separated) |
| `spec.config.auth.oidc.disableInvitations` | `BINDPLANE_OIDC_DISABLE_INVITATIONS` | `false` | No |

Example (direct values):

```yaml
spec:
  config:
    auth:
      type: oidc
      oidc:
        issuer: https://accounts.example.com
        scopes:
          - openid
          - profile
          - email
        clientID: "my-client-id"
        clientSecret: "my-client-secret"
```

Example (Secret references):

```yaml
spec:
  config:
    auth:
      type: oidc
      oidc:
        issuer: https://accounts.example.com
        scopes:
          - openid
          - profile
          - email
        clientIDSecretRef:
          name: oidc-secrets
          key: client-id
        clientSecretSecretRef:
          name: oidc-secrets
          key: client-secret
```

## Network

TLS is generally not configured on the Bindplane server when you use Ingress or Gateway API to terminate TLS. In that case, only `remoteURL` (and optionally `webURL`) need to reflect the external URL; the server continues to listen over HTTP inside the cluster.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.network.host` | `BINDPLANE_HOST` | — | No |
| `spec.config.network.port` | `BINDPLANE_PORT` | — | No |
| `spec.config.network.remoteURL` | `BINDPLANE_REMOTE_URL` | `http://<name>-node:3001` | No |
| `spec.config.network.webURL` | `BINDPLANE_WEB_URL` | — | No |
| `spec.config.network.corsAllowedOrigins` | `BINDPLANE_CORS_ALLOWED_ORIGINS` | — | No |
| `spec.config.network.tls` | (see below) | — | No |

`BINDPLANE_REMOTE_URL` is always set. When `spec.config.network.remoteURL` is not configured, it defaults to the internal node service URL (`http://<bindplane-name>-node:3001`). Override this when the Bindplane UI is accessed through an ingress or load balancer, e.g. `https://bindplane.my-corp.net`.

**Network TLS:** Configure server-side TLS (certificate and key) or mutual TLS (additionally a CA to verify client certificates). You provide a Secret name and key names; the operator mounts the Secret and sets the Bindplane environment variables to the mounted file paths.

| TLS Field | Environment Variable | Description |
|---|---|---|
| `spec.config.network.tls.minVersion` | `BINDPLANE_TLS_MIN_VERSION` | Minimum TLS version: `1.2` or `1.3`. Omit to use server default. |
| `spec.config.network.tls.secretName` | — | Name of the Secret containing the cert, key, and optionally CA |
| `spec.config.network.tls.certKey` | `BINDPLANE_TLS_CERT` | Key in the Secret for the TLS certificate (server or mutual TLS) |
| `spec.config.network.tls.keyKey` | `BINDPLANE_TLS_KEY` | Key in the Secret for the TLS private key (server or mutual TLS) |
| `spec.config.network.tls.caKey` | `BINDPLANE_TLS_CA` | Key in the Secret for the CA certificate (optional; enables mutual TLS, generally not used) |
| `spec.config.network.tls.skipVerify` | `BINDPLANE_TLS_SKIP_VERIFY` | Skip TLS verification (testing only). Default: not set. |

Valid combinations:

- **Server-side TLS:** Set `secretName`, `certKey`, and `keyKey` only.
- **Mutual TLS:** Set `secretName`, `certKey`, `keyKey`, and `caKey`.
- `minVersion` and `skipVerify` are optional in all cases.

Example (server-side TLS):

```yaml
spec:
  config:
    network:
      remoteURL: https://bindplane.my-corp.net
      tls:
        secretName: bindplane-tls
        certKey: tls.crt
        keyKey: tls.key
```

Example (mutual TLS with CA):

```yaml
spec:
  config:
    network:
      tls:
        minVersion: "1.3"
        secretName: bindplane-tls
        certKey: tls.crt
        keyKey: tls.key
        caKey: ca.crt
```

## Store

The store type is always `postgres`. `BINDPLANE_STORE_TYPE` is automatically set to `postgres` by the operator.

The following store-level settings apply regardless of backend. When omitted, Bindplane uses its own defaults.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.store.maxEvents` | `BINDPLANE_STORE_MAX_EVENTS` | 100 | No |
| `spec.config.store.eventMergeWindow` | `BINDPLANE_STORE_EVENT_MERGE_WINDOW` | `100ms` | No |
| `spec.config.store.summaryRollupRetentionDays` | `BINDPLANE_STORE_SUMMARY_ROLLUP_RETENTION_DAYS` | 365 | No |

`summaryRollupRetentionDays: 0` means indefinite retention (rollups are never deleted).

```yaml
spec:
  config:
    store:
      maxEvents: 200
      eventMergeWindow: "200ms"
      summaryRollupRetentionDays: 90
      postgres:
        host: postgres.default.svc
```

### PostgreSQL

Username and password can be set as direct values or via Secret references (`usernameSecretRef`, `passwordSecretRef`). Each uses `name` and `key` to reference a Secret. Secret references take precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.store.postgres.host` | `BINDPLANE_POSTGRES_HOST` | — | Yes |
| `spec.config.store.postgres.port` | `BINDPLANE_POSTGRES_PORT` | — | No |
| `spec.config.store.postgres.database` | `BINDPLANE_POSTGRES_DATABASE` | — | No |
| `spec.config.store.postgres.username` | `BINDPLANE_POSTGRES_USERNAME` | — | No |
| `spec.config.store.postgres.usernameSecretRef` | `BINDPLANE_POSTGRES_USERNAME` | — | No |
| `spec.config.store.postgres.password` | `BINDPLANE_POSTGRES_PASSWORD` | — | No |
| `spec.config.store.postgres.passwordSecretRef` | `BINDPLANE_POSTGRES_PASSWORD` | — | No |
| `spec.config.store.postgres.sslmode` | `BINDPLANE_POSTGRES_SSL_MODE` | `disable` | No |
| `spec.config.store.postgres.tls` | (see below) | — | No |
| `spec.config.store.postgres.connectTimeout` | `BINDPLANE_POSTGRES_CONNECT_TIMEOUT` | — | No |
| `spec.config.store.postgres.statementTimeout` | `BINDPLANE_POSTGRES_STATEMENT_TIMEOUT` | — | No |
| `spec.config.store.postgres.maxConnections` | `BINDPLANE_POSTGRES_MAX_CONNECTIONS` | — | No |
| `spec.config.store.postgres.maxIdleConnections` | `BINDPLANE_POSTGRES_MAX_IDLE_CONNECTIONS` | — | No |
| `spec.config.store.postgres.maxLifetime` | `BINDPLANE_POSTGRES_MAX_LIFETIME` | — | No |
| `spec.config.store.postgres.maxIdleTime` | `BINDPLANE_POSTGRES_MAX_IDLE_TIME` | — | No |
| `spec.config.store.postgres.schema` | `BINDPLANE_POSTGRES_SCHEMA` | — | No |

**PostgreSQL TLS:** By default TLS is disabled (`sslmode: disable`). To use TLS, set `sslmode` to `require`, `verify-ca`, or `verify-full` and configure `tls` with a Secret. Specify a CA (caKey) for server-side TLS verification; add certKey and keyKey for mutual TLS (client certificate). The operator mounts the Secret and sets the Bindplane environment variables to the mounted file paths.

| TLS Field | Environment Variable | Description |
|---|---|---|
| `spec.config.store.postgres.tls.secretName` | — | Name of the Secret containing the CA and optionally client cert and key |
| `spec.config.store.postgres.tls.caKey` | `BINDPLANE_POSTGRES_SSL_ROOT_CERT` | Key in the Secret for the root CA (server-side TLS) |
| `spec.config.store.postgres.tls.certKey` | `BINDPLANE_POSTGRES_SSL_CERT` | Key in the Secret for the client certificate (mutual TLS) |
| `spec.config.store.postgres.tls.keyKey` | `BINDPLANE_POSTGRES_SSL_KEY` | Key in the Secret for the client private key (mutual TLS) |

Valid combinations:

- **Server-side TLS:** Set `sslmode` (e.g. `verify-ca` or `verify-full`) and `tls.secretName` with `tls.caKey`.
- **Mutual TLS:** In addition, set `tls.certKey` and `tls.keyKey`.

Example (direct values):

```yaml
spec:
  config:
    store:
      postgres:
        host: postgres.example.com
        username: bindplane
        password: "my-pg-password"
```

Example (Secret references):

```yaml
spec:
  config:
    store:
      postgres:
        host: postgres.example.com
        usernameSecretRef:
          name: bindplane-secrets
          key: pg-username
        passwordSecretRef:
          name: bindplane-secrets
          key: pg-password
```

Example (PostgreSQL server-side TLS with CA):

```yaml
spec:
  config:
    store:
      postgres:
        host: postgres.example.com
        sslmode: verify-ca
        tls:
          secretName: postgres-tls
          caKey: ca.crt
```

Example (PostgreSQL mutual TLS):

```yaml
spec:
  config:
    store:
      postgres:
        host: postgres.example.com
        sslmode: verify-full
        tls:
          secretName: postgres-tls
          caKey: ca.crt
          certKey: tls.crt
          keyKey: tls.key
```

## Tracing

Tracing is optional. When `spec.config.tracing` is omitted or `type` is empty, tracing is disabled and no tracing environment variables are set.

Supported types: `otlp`, `google`. For `otlp`, configure the `otlp` block with endpoint and optional insecure flag. You can set a sampling rate (string, e.g. `"0.5"`) between 0 and 1.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.tracing.type` | `BINDPLANE_TRACING_TYPE` | — | No (omit to disable) |
| `spec.config.tracing.otlp.endpoint` | `BINDPLANE_TRACING_OTLP_ENDPOINT` | — | Yes when type is `otlp` |
| `spec.config.tracing.otlp.insecure` | `BINDPLANE_TRACING_OTLP_INSECURE` | `false` | No |
| `spec.config.tracing.samplingRate` | `BINDPLANE_TRACING_SAMPLING_RATE` | — | No |

Example (OTLP tracing):

```yaml
spec:
  config:
    tracing:
      type: otlp
      otlp:
        endpoint: http://otel-collector.observability.svc:4317
        insecure: false
      samplingRate: "0.5"
```

## Metrics

Metrics configuration is optional. When `spec.config.metrics` is omitted, the operator applies defaults: type `prometheus`, interval `60s`, and endpoint `/metrics`. When present, those fields use CRD defaults when not set.

Supported types: `prometheus`, `otlp`. For `prometheus`, the server exposes metrics on an HTTP path; you can optionally set basic auth via `username` and `password` or `passwordSecretRef`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.metrics.type` | `BINDPLANE_METRICS_TYPE` | `prometheus` | No |
| `spec.config.metrics.interval` | `BINDPLANE_METRICS_INTERVAL` | `60s` | No |
| `spec.config.metrics.prometheus.endpoint` | `BINDPLANE_METRICS_PROMETHEUS_ENDPOINT` | `/metrics` | No |
| `spec.config.metrics.prometheus.username` | `BINDPLANE_METRICS_PROMETHEUS_USERNAME` | — | No |
| `spec.config.metrics.prometheus.password` | `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` | — | No |
| `spec.config.metrics.prometheus.passwordSecretRef` | `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` | — | No |
| `spec.config.metrics.otlp.endpoint` | `BINDPLANE_METRICS_OTLP_ENDPOINT` | — | Yes when type is `otlp` |
| `spec.config.metrics.otlp.insecure` | `BINDPLANE_METRICS_OTLP_INSECURE` | `false` | No |

Example (default Prometheus metrics; optional—same as omitting `metrics`):

```yaml
spec:
  config:
    metrics:
      type: prometheus
      interval: "60s"
      prometheus:
        endpoint: /metrics
```

Example (Prometheus metrics with basic auth):

```yaml
spec:
  config:
    metrics:
      type: prometheus
      prometheus:
        endpoint: /metrics
        username: metrics-reader
        passwordSecretRef:
          name: bindplane-secrets
          key: metrics-password
```

Example (OTLP metrics):

```yaml
spec:
  config:
    metrics:
      type: otlp
      interval: "60s"
      otlp:
        endpoint: otel-collector.observability.svc:4317
        insecure: true
```

## TSDB

Bindplane requires a TSDB for agent health and throughput metrics.

- **Default deployment:** the operator deploys Bindplane's TSDB using a Prometheus StatefulSet.
- **Remote deployment:** you can use a user-managed TSDB backend (for example, VictoriaMetrics) via `spec.config.tsdb.remote`.

### TLS

TLS for TSDB remote write is configured under `spec.config.tsdb.tls`. Use either a **user-defined Secret** (`secretName` plus key names) or **cert-manager** (`certManager`), not both.

| CRD Field | Description |
|---|---|
| `spec.config.tsdb.tls.secretName` | Name of the Secret containing the TLS certificate, key, and optionally CA (user-defined TLS). Omit when using certManager. |
| `spec.config.tsdb.tls.certKey` | Key in the Secret for the TLS certificate. |
| `spec.config.tsdb.tls.keyKey` | Key in the Secret for the TLS private key. |
| `spec.config.tsdb.tls.caKey` | Key in the Secret for the CA certificate. |
| `spec.config.tsdb.tls.certManager` | Reference to a cert-manager Issuer or ClusterIssuer to issue server and client certs (mTLS). Mutually exclusive with secretName. |
| `spec.config.tsdb.tls.certManager.name` | Name of the Issuer or ClusterIssuer. |
| `spec.config.tsdb.tls.certManager.kind` | `Issuer` or `ClusterIssuer`. Default: `Issuer`. |
| `spec.config.tsdb.tls.certManager.group` | API group. Default: `cert-manager.io`. |
| `spec.config.tsdb.tls.skipVerify` | When `true`, set `BINDPLANE_PROMETHEUS_TLS_SKIP_VERIFY=true` to disable TLS certificate verification (testing only). |

When using cert-manager, see [Security: TLS and Secrets – Cert Manager and TSDB mTLS](security.md#cert-manager-and-tsdb-mtls-optional) for prerequisites and behavior.

Example (user-defined Secret):

```yaml
spec:
  config:
    tsdb:
      tls:
        secretName: my-tsdb-tls
        certKey: tls.crt
        keyKey: tls.key
        caKey: ca.crt
```

Example (cert-manager):

```yaml
spec:
  config:
    tsdb:
      tls:
        certManager:
          name: bindplane-ca-issuer
          kind: ClusterIssuer
          group: cert-manager.io
```

### Remote TSDB

Use `spec.config.tsdb.remote` when you want Bindplane to connect to a user-managed TSDB instead of the operator-managed default Prometheus StatefulSet.

| CRD Field | Description |
|---|---|
| `spec.config.tsdb.remote.enable` | Enables remote TSDB mode. |
| `spec.config.tsdb.remote.host` | Required when `enable=true`. Hostname or IP of the remote TSDB endpoint used for query operations. |
| `spec.config.tsdb.remote.port` | Port for the remote TSDB endpoint. Defaults to `9090`. |
| `spec.config.tsdb.remote.queryPathPrefix` | Optional PromQL path prefix (useful for systems like VictoriaMetrics or Mimir). |
| `spec.config.tsdb.remote.remoteWrite.host` | Optional remote-write host override. Must be set together with `remoteWrite.port`. |
| `spec.config.tsdb.remote.remoteWrite.port` | Optional remote-write port override. Must be set together with `remoteWrite.host`. |
| `spec.config.tsdb.remote.remoteWrite.endpoint` | Optional remote-write path. Defaults to `/api/v1/write`. |

Example (user-managed TSDB, e.g. VictoriaMetrics):

```yaml
spec:
  config:
    tsdb:
      remote:
        enable: true
        host: vmselect.monitoring.svc
        port: 8481
        queryPathPrefix: /select/0/prometheus
        remoteWrite:
          host: vminsert.monitoring.svc
          port: 8480
          endpoint: /insert/0/prometheus/api/v1/write
```

## Max concurrency

`maxConcurrency` and `agents.maxSimultaneousConnections` both control the maximum number of goroutines
servicing OpAMP connections concurrently. They generally use the same value and should only be changed
when directed by Bindplane support.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.maxConcurrency` | `BINDPLANE_MAX_CONCURRENCY` | `10` | No |
| `spec.config.agents.maxSimultaneousConnections` | `BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS` | `10` | No |

Do not change these fields unless directed by Bindplane support.

## Audit trail

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auditTrail.retentionDays` | `BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS` | `365` | No |

## Event bus

Event bus configuration controls the NATS integration health check. The health check sends an event over NATS and waits for responses from Bindplane components. Failures affect the status page only — they do not cause pod restarts or rollouts.

`requiredHosts` defaults to `floor(total/2)+1` where `total` is the sum of Node, NATS, and Jobs replicas. Jobs Migrate is a batch/v1 Job (not a long-running pod) and is excluded from this total. This default ensures a majority quorum. `interval` controls how frequently the health check runs.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.eventBus.health.requiredHosts` | `BINDPLANE_EVENT_BUS_HEALTH_REQUIRED_HOSTS` | `floor(total/2)+1` | No |
| `spec.config.eventBus.health.interval` | `BINDPLANE_EVENT_BUS_HEALTH_INTERVAL` | — | No |

Example:

```yaml
spec:
  config:
    eventBus:
      health:
        requiredHosts: 2
        interval: "30s"
```

## Logging

Bindplane always logs to stdout; the output destination is operator-managed and not user-configurable. The `spec.config.logging` block exposes only the log level.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.logging.level` | `BINDPLANE_LOGGING_LEVEL` | `info` | No |

Valid values for `level`: `debug`, `info`, `warn`, `error`.

The operator also sets `BINDPLANE_LOGGING_TYPE=stdout` on every Bindplane pod. This is not user-configurable.

Example:

```yaml
spec:
  config:
    logging:
      level: debug
```

## Status checks

The `spec.config.status` section configures Bindplane's authenticated status check endpoints. Status checks are **enabled by default**, including when the `status` block is omitted entirely — no configuration is required to use them.

Requests to the status endpoints are authenticated with one or more UUID keys. There are three ways the keys are supplied, in priority order:

1. **`keysSecretRef`** — reference a Kubernetes Secret you manage (value is comma-delimited UUIDs, supporting rotation).
2. **`keys`** — inline UUIDs (each must be a valid UUID; multiple entries support rotation).
3. **Operator-managed (default)** — when status is enabled and you supply neither `keys` nor `keysSecretRef`, the operator generates a UUID and stores it in a Secret named `<bindplane-name>-status-secret` under the key `status-keys`. The Bindplane pods reference it via `secretKeyRef`. This Secret is owned by the Bindplane CR (garbage-collected on delete), created once, and never overwritten, so the key is stable across reconciles. If you later supply your own `keys` or `keysSecretRef`, the pods switch to your value and the operator stops managing the generated Secret.

To rotate keys yourself, supply multiple UUIDs via `keys` or a comma-delimited `keysSecretRef`. Set `enabled: false` to turn the endpoints off entirely — no keys are emitted and no Secret is generated.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.status.enabled` | `BINDPLANE_STATUS_ENABLED` | `true` | No |
| `spec.config.status.keys` | `BINDPLANE_STATUS_KEYS` | auto-generated | No |
| `spec.config.status.keysSecretRef` | `BINDPLANE_STATUS_KEYS` | auto-generated | No |

Example — zero config (operator generates and manages the key):

```yaml
spec:
  config: {}
```

Example — user-supplied inline keys:

```yaml
spec:
  config:
    status:
      enabled: true
      keys:
        - "11111111-1111-1111-1111-111111111111"
        - "22222222-2222-2222-2222-222222222222"
```

Example — user-managed Secret:

```yaml
spec:
  config:
    status:
      enabled: true
      keysSecretRef:
        name: bindplane-status-keys
        key: keys
```

## Agents

The `spec.config.agents` section configures how Bindplane communicates with agents, including heartbeat timing, rebalancing, and authentication. When omitted, Bindplane uses its own defaults.

### Agents Authentication

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.auth.type` | `BINDPLANE_AGENTS_AUTH_TYPE` | `secretKey` | No |
| `spec.config.agents.auth.secretKey.headers` | `BINDPLANE_AGENTS_AUTH_SECRET_KEY_HEADERS` | `X-Bindplane-Authorization,Authorization` | No |

`auth.type` accepts `secretKey`.
`[]string` fields (headers) are comma-separated in the env var.

### Heartbeat

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.heartbeatInterval` | `BINDPLANE_AGENTS_HEARTBEAT_INTERVAL` | `30s` | No |
| `spec.config.agents.heartbeatTTL` | `BINDPLANE_AGENTS_HEARTBEAT_TTL` | `1m` | No |
| `spec.config.agents.heartbeatExpiryInterval` | `BINDPLANE_AGENTS_HEARTBEAT_EXPIRY_INTERVAL` | `30s` | No |

### Rebalance

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.rebalanceInterval` | `BINDPLANE_AGENTS_REBALANCE_INTERVAL` | `1h` | No |
| `spec.config.agents.rebalancePercentage` | `BINDPLANE_AGENTS_REBALANCE_PERCENTAGE` | `0` | No |
| `spec.config.agents.rebalanceJitter` | `BINDPLANE_AGENTS_REBALANCE_JITTER` | `0` | No |

`rebalancePercentage` and `rebalanceJitter` are integers in the range 0–100. A value of 0 disables that feature.

```yaml
spec:
  config:
    agents:
      heartbeatInterval: "45s"
      heartbeatTTL: "2m"
      rebalanceInterval: "30m"
      rebalancePercentage: 50
```

## Agent versions

The `spec.config.agentVersions` section configures how Bindplane syncs agent version metadata.
When omitted, Bindplane uses its own defaults.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agentVersions.syncInterval` | `BINDPLANE_AGENT_VERSIONS_SYNC_INTERVAL` | `1h` | No |

`syncInterval` must be at least `1h` (enforced by Bindplane at runtime).

```yaml
spec:
  config:
    agentVersions:
      syncInterval: "2h"
```

## Extra environment variables

Each component exposes an `extraEnv` field that injects arbitrary environment variables into its primary container. These are prepended **before** the operator-managed variables. Because Kubernetes uses the **last** occurrence of a duplicate name, operator-managed values always win over any colliding name in `extraEnv`.

| CRD Field | Description |
|---|---|
| `spec.bindplane.extraEnv` | Extra env vars for the Bindplane Node Deployment. |
| `spec.bindplaneJobs.extraEnv` | Extra env vars for the Bindplane Jobs Deployment. |
| `spec.bindplaneJobsMigrate.extraEnv` | Extra env vars for the Bindplane Jobs Migrate batch/v1 Job. |
| `spec.transformAgent.extraEnv` | Extra env vars for the Transform Agent Deployment. |
| `spec.tsdb.extraEnv` | Extra env vars for the TSDB (Prometheus) StatefulSet. |
| `spec.nats.extraEnv` | Extra env vars for the NATS StatefulSet. |
| `spec.opamp.extraEnv` | Extra env vars for the dedicated OpAMP Deployment (only used when `spec.opamp.enabled` is true). |

Each entry follows the standard Kubernetes `EnvVar` schema, which supports both inline values and references to Secrets or ConfigMaps via `valueFrom`.

Example (egress proxy for all Bindplane Node pods):

```yaml
spec:
  bindplane:
    extraEnv:
      - name: HTTP_PROXY
        value: "http://proxy.corp.example.com:3128"
      - name: HTTPS_PROXY
        value: "http://proxy.corp.example.com:3128"
      - name: NO_PROXY
        value: "localhost,127.0.0.1,.svc.cluster.local"
```

Example (secret reference for a Google Application credentials path):

```yaml
spec:
  bindplane:
    extraEnv:
      - name: GOOGLE_APPLICATION_CREDENTIALS
        valueFrom:
          secretKeyRef:
            name: gcp-credentials
            key: credentials-path
```

### Reserved env names

The following names are **always** reserved and may never appear in `extraEnv` because they are injected by the operator at runtime:

| Name | Reason |
|---|---|
| `KUBERNETES_NAMESPACE_NAME` | Injected via Downward API |
| `KUBERNETES_POD_NAME` | Injected via Downward API |
| `KUBERNETES_CONTAINER_NAME` | Injected by operator |
| `GOMEMLIMIT` | Set by the operator's Go runtime tuning |
| `GOMAXPROCS` | Set by the operator's Go runtime tuning |

Names starting with `BINDPLANE_` are also rejected by default because they map to fields in `spec.config`—use the CRD fields instead. To override this restriction (for advanced/unsupported use cases), start the operator with the `--allow-bindplane-extra-env` flag.

### Pod template vs extraEnv

The `podTemplate` field gives access to the full Kubernetes pod spec (tolerations, node selectors, affinity, security contexts, etc.). The operator intentionally **does not** merge environment variables from `podTemplate.spec.containers[*].env`—those entries are ignored for the primary container. Use `extraEnv` for all custom environment variable injection.

## Extra volumes and volume mounts

Each component exposes `extraVolumes` and `extraVolumeMounts` fields that mount arbitrary files or directories into its primary container. Configuration is per-component: each component independently declares which volumes it needs.

| CRD Field | Description |
|---|---|
| `spec.bindplane.extraVolumes` / `spec.bindplane.extraVolumeMounts` | Extra volumes/mounts for the Bindplane Node Deployment. |
| `spec.bindplaneJobs.extraVolumes` / `spec.bindplaneJobs.extraVolumeMounts` | Extra volumes/mounts for the Bindplane Jobs Deployment. |
| `spec.bindplaneJobsMigrate.extraVolumes` / `spec.bindplaneJobsMigrate.extraVolumeMounts` | Extra volumes/mounts for the Bindplane Jobs Migrate batch/v1 Job. |
| `spec.transformAgent.extraVolumes` / `spec.transformAgent.extraVolumeMounts` | Extra volumes/mounts for the Transform Agent Deployment. |
| `spec.tsdb.extraVolumes` / `spec.tsdb.extraVolumeMounts` | Extra volumes/mounts for the TSDB (Prometheus) StatefulSet. |
| `spec.nats.extraVolumes` / `spec.nats.extraVolumeMounts` | Extra volumes/mounts for the NATS StatefulSet. |
| `spec.opamp.extraVolumes` / `spec.opamp.extraVolumeMounts` | Extra volumes/mounts for the dedicated OpAMP Deployment (only when `spec.opamp.enabled` is true). |

Each entry in `extraVolumes` follows the standard Kubernetes `Volume` schema. Each entry in `extraVolumeMounts` follows the standard Kubernetes `VolumeMount` schema.

**Mount references are component-scoped.** Every `extraVolumeMounts[].name` must reference a volume defined in the **same component's** `extraVolumes`. You cannot reference operator-managed volumes or volumes from another component's `extraVolumes`.

Example — Redis CA certificate mounted into Node, OpAMP, and Jobs (not Migrate, NATS, Transform Agent, or TSDB):

```yaml
spec:
  bindplane:
    extraVolumes:
      - name: redis-ca
        secret:
          secretName: redis-ca-cert
    extraVolumeMounts:
      - name: redis-ca
        mountPath: /etc/redis-ca
  opamp:
    enabled: true
    extraVolumes:
      - name: redis-ca
        secret:
          secretName: redis-ca-cert
    extraVolumeMounts:
      - name: redis-ca
        mountPath: /etc/redis-ca
  bindplaneJobs:
    extraVolumes:
      - name: redis-ca
        secret:
          secretName: redis-ca-cert
    extraVolumeMounts:
      - name: redis-ca
        mountPath: /etc/redis-ca
```

Example — Extra Prometheus rules ConfigMap mounted only into TSDB:

```yaml
spec:
  tsdb:
    extraVolumes:
      - name: prom-rules
        configMap:
          name: my-prometheus-rules
    extraVolumeMounts:
      - name: prom-rules
        mountPath: /etc/prometheus/rules.d
```

### Allowed volume sources

The validating webhook restricts which volume source types are permitted. Only the following are accepted:

| Source | Notes |
|---|---|
| `secret` | Mount a Kubernetes Secret |
| `configMap` | Mount a Kubernetes ConfigMap |
| `projected` | Projected volume combining secrets, configMaps, serviceAccountTokens, or downwardAPI |
| `csi` | CSI ephemeral inline volume (e.g. for secrets-store-csi-driver) |
| `emptyDir` | Ephemeral scratch space; useful when the container needs a writable directory |
| `downwardAPI` | Expose pod/container metadata as files |

`hostPath` and all other sources are rejected for security reasons.

### Reserved volume names and mount paths

The following volume names and mount paths are managed by the operator and may not be used in `extraVolumes` or `extraVolumeMounts`:

| Reserved volume name | Reserved mount path |
|---|---|
| `ldap-tls` | `/etc/bindplane/ldap-tls` |
| `network-tls` | `/etc/bindplane/network-tls` |
| `postgres-tls` | `/etc/bindplane/postgres-tls` |
| `tsdb-remote-write-tls` | `/etc/bindplane/tsdb-remote-write-tls` |
| `nats-tls` | `/etc/bindplane/nats-tls` |
| `transform-agent-tls` | `/etc/bindplane/transform-agent-tls` |
| `tsdb-web-config` | `/etc/prometheus` |
| `tsdb-tls` | `/etc/tsdb-tls` |
| `tsdb-web-server-tls` | `/etc/tsdb-web-tls` |
| `tsdb-probe-client-tls` | `/etc/tsdb-probe-client` |
| `tsdb-probe-auth` | `/etc/tsdb-probe-auth` |
| `<name>-tsdb-data` _(dynamic)_ | `/prometheus` |

The `<name>-tsdb-data` volume is the TSDB StatefulSet's persistent data volume, named after the Bindplane CR (e.g. `my-bindplane-tsdb-data` for a CR named `my-bindplane`). Any `extraVolumes` entry for the TSDB component with that name is rejected.

### Pod template vs extraVolumeMounts

The `podTemplate` field gives access to the full Kubernetes pod spec. The operator intentionally **does not** merge volume mounts from `podTemplate.spec.containers[*].volumeMounts`—those entries are ignored for the primary container. Use `extraVolumeMounts` for all custom volume mount injection.

## Argo Rollouts (primary Bindplane component)

By default, the Bindplane Node workload is managed as a Kubernetes `Deployment`. When you set `spec.bindplane.argoRollout.enabled: true`, the operator switches the primary Node workload to an Argo Rollouts `Rollout` resource using the **BlueGreen** strategy.

**Prerequisites:**
- The [Argo Rollouts controller](https://argo-rollouts.readthedocs.io/en/stable/installation/) and its CRDs must be installed in the cluster.
- Only the BlueGreen strategy is supported.

> **Deployment order matters.** The Bindplane operator checks for the Argo Rollouts CRD at startup and registers a Kubernetes watch only when the CRD is present. Install the Argo Rollouts controller and CRDs **before** deploying the Bindplane operator. If you are migrating an existing installation to use Argo Rollouts, install Argo Rollouts first, then restart the Bindplane operator (`kubectl rollout restart deployment/<name> -n <namespace>`) so it picks up the new CRD and registers the watch.

Toggling `argoRollout.enabled` deletes the existing workload (Deployment or Rollout) and creates the new one. This causes a brief interruption; plan accordingly.

> **Note:** `spec.bindplane.strategy` and `spec.bindplane.argoRollout.enabled: true` are mutually exclusive. The operator will reject a CR that sets both.

| CRD Field | Default | Description |
|---|---|---|
| `spec.bindplane.argoRollout.enabled` | `false` | Switches the primary Node workload from a Deployment to an Argo Rollout (BlueGreen) |
| `spec.bindplane.argoRollout.autoPromotionEnabled` | `true` | Automatically promotes the new ReplicaSet to active once it is available |
| `spec.bindplane.argoRollout.scaleDownDelaySeconds` | Argo default (30s) | Seconds the previous ReplicaSet stays running after promotion |

> **Recommended: when `spec.bindplane.argoRollout.enabled: true`, also set `spec.opamp.enabled: true`.** BlueGreen promotions swap active traffic atomically; routing OpAMP/agent traffic to a dedicated Deployment prevents agent reconnect storms during promotion. See the [OpAMP deployment split](#opamp-deployment-split) section for details.

### Minimal example (ArgoRollout only)

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane-sample
spec:
  bindplane:
    argoRollout:
      enabled: true
```

### Recommended example (ArgoRollout + dedicated OpAMP)

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane-sample
spec:
  bindplane:
    argoRollout:
      enabled: true
      autoPromotionEnabled: true
      scaleDownDelaySeconds: 60
  opamp:
    enabled: true
    replicas: 3
```

## OpAMP deployment split

By default, the Bindplane Node deployment serves both the frontend (UI and REST API) and OpAMP/agent traffic on the same pods. When you have a large fleet of agents but modest UI traffic, you can scale agent handling independently by enabling a dedicated OpAMP deployment.

When `spec.opamp.enabled` is `true`, the operator provisions a second Deployment (`<name>-opamp`) running the same Bindplane EE image in `node` mode. Both Deployments share the same configuration (license, store, event bus, auth). A separate Service (`<name>-opamp`) exposes the OpAMP pods.

> **Warning:** The OpAMP deployment shares the same Bindplane configuration as the primary Node deployment (license, store, event bus, auth). Changing `spec.opamp.maxSimultaneousConnections` overrides `spec.config.agents.maxSimultaneousConnections` for the OpAMP pods only; the frontend (Node) pods continue to use the shared value.

| CRD Field | Default | Description |
|---|---|---|
| `spec.opamp.enabled` | `false` | Enables the dedicated OpAMP deployment |
| `spec.opamp.replicas` | `3` | Number of OpAMP replicas (ignored when autoscaling is enabled) |
| `spec.opamp.resources` | 2 CPU / 2 GiB | Compute resources for the OpAMP container |
| `spec.opamp.podTemplate` | — | Pod template overrides (same merge rules as other components) |
| `spec.opamp.disablePodDisruptionBudget` | `false` | Disables the operator-managed PDB |
| `spec.opamp.minReadySeconds` | termination grace period | Minimum seconds a pod must be ready before considered available |
| `spec.opamp.strategy` | RollingUpdate maxSurge=1 maxUnavailable=0 | Rollout strategy |
| `spec.opamp.autoscaling` | — | HPA configuration (same structure as `spec.bindplane.autoscaling`) |
| `spec.opamp.maxSimultaneousConnections` | shared value | Per-deployment override for `BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS` |
| `spec.opamp.shutdownGracePeriodTarget` | — | Fraction (0–1) of shutdown grace period for OpAMP drain |

### Example: enabling OpAMP split with HPA

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane-sample
spec:
  config:
    license: "my-license-key"
    store:
      postgres:
        host: postgres.example.com
  bindplane:
    replicas: 3
    resources:
      requests:
        cpu: "1000m"
        memory: "1024Mi"
      limits:
        memory: "1024Mi"
  opamp:
    enabled: true
    resources:
      requests:
        cpu: "2000m"
        memory: "2048Mi"
      limits:
        memory: "2048Mi"
    maxSimultaneousConnections: 2000
    shutdownGracePeriodTarget: "0.6"
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 20
```

### Ingress routing guidance

Agents connect to Bindplane over WebSocket on `/v1/opamp` and HTTP on `/v1/agent/*`. When the OpAMP split is enabled, route those paths to the `<name>-opamp` Service and route all other traffic to `<name>-node`.

Example Ingress (nginx ingress controller):

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bindplane
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  rules:
  - host: bindplane.example.com
    http:
      paths:
      - path: /v1/opamp
        pathType: Prefix
        backend:
          service:
            name: bindplane-sample-opamp
            port:
              number: 3001
      - path: /v1/agent
        pathType: Prefix
        backend:
          service:
            name: bindplane-sample-opamp
            port:
              number: 3001
      - path: /
        pathType: Prefix
        backend:
          service:
            name: bindplane-sample-node
            port:
              number: 3001
```

> **Note:** If you use BindplaneGateway to expose Bindplane, configure it to route OpAMP traffic to the `<name>-opamp` Service. Refer to the BindplaneGateway documentation for routing configuration. The operator does not configure BindplaneGateway.

> **Note:** The operator provisions a regular ClusterIP Service for the OpAMP deployment (`<name>-opamp`). A headless Service is not created because agents do not require pod-direct DNS — load balancing through the ClusterIP Service is sufficient.

## Container images

By default, all service container images are derived from `spec.version` using three image repositories:

| Service | Default image |
|---|---|
| Node, OpAMP, NATS, Jobs, Jobs Migrate | `ghcr.io/observiq/bindplane-ee:<version>` |
| Transform Agent | `ghcr.io/observiq/bindplane-transform-agent:<version>-bindplane` |
| TSDB (Prometheus) | `ghcr.io/observiq/bindplane-prometheus:<version>` |

This is the recommended approach: change `spec.version` to roll all services together.

### Per-service image override

Each component spec exposes an `image` field that accepts a full OCI reference. When set, it is used verbatim — the value completely replaces the version-derived default for that service only. Other services continue to derive their image from `spec.version`.

| CRD Field | Applies to |
|---|---|
| `spec.bindplane.image` | Bindplane Node |
| `spec.opamp.image` | OpAMP dedicated deployment |
| `spec.nats.image` | NATS |
| `spec.bindplaneJobs.image` | Bindplane Jobs |
| `spec.bindplaneJobsMigrate.image` | Bindplane Jobs Migrate (migration batch Job) |
| `spec.transformAgent.image` | Transform Agent |
| `spec.tsdb.image` | TSDB (Prometheus) |

Example — mirror all images through a private registry while keeping the global version:

```yaml
spec:
  version: "1.99.1"
  bindplane:
    image: "registry.corp.example.com/bindplane/bindplane-ee:1.99.1"
  nats:
    image: "registry.corp.example.com/bindplane/bindplane-ee:1.99.1"
  bindplaneJobs:
    image: "registry.corp.example.com/bindplane/bindplane-ee:1.99.1"
  bindplaneJobsMigrate:
    image: "registry.corp.example.com/bindplane/bindplane-ee:1.99.1"
  transformAgent:
    image: "registry.corp.example.com/bindplane/bindplane-transform-agent:1.99.1-bindplane"
  tsdb:
    image: "registry.corp.example.com/bindplane/bindplane-prometheus:1.99.1"
```

Example — pin only Jobs Migrate to a hotfix image:

```yaml
spec:
  version: "1.99.1"
  bindplaneJobsMigrate:
    image: "ghcr.io/observiq/bindplane-ee:1.99.1-hotfix"
```

> **Warning:** `spec.bindplaneJobsMigrate.image` controls the image used by the database migration Job. The operator gates downstream rollouts on this Job completing successfully and records the migrate image in `status.migratedImage`. If you pin jobs-migrate to a different version than node, jobs, and nats, ensure the migration is compatible with the runtime image — the operator does not enforce this constraint.

The `VERSION` column shown by `kubectl get bindplane` always reflects `spec.version`, regardless of per-service overrides.

## Service account annotations

Each operator-managed ServiceAccount supports user-defined annotations. This is the standard mechanism for binding a Kubernetes ServiceAccount to a cloud IAM identity without storing credentials in the cluster — used by AWS IRSA (IAM Roles for Service Accounts) and GKE Workload Identity.

| CRD Field | Component |
|---|---|
| `spec.bindplane.serviceAccount.annotations` | Bindplane Node |
| `spec.opamp.serviceAccount.annotations` | OpAMP dedicated deployment |
| `spec.nats.serviceAccount.annotations` | NATS |
| `spec.bindplaneJobs.serviceAccount.annotations` | Bindplane Jobs |
| `spec.bindplaneJobsMigrate.serviceAccount.annotations` | Bindplane Jobs Migrate |
| `spec.transformAgent.serviceAccount.annotations` | Transform Agent |
| `spec.tsdb.serviceAccount.annotations` | TSDB (Prometheus) |

Annotations are reconciled on every sync — adding, updating, or removing an annotation in the spec is reflected on the live ServiceAccount within one reconcile cycle.

Example — AWS IRSA on the primary Bindplane Node:

```yaml
spec:
  bindplane:
    serviceAccount:
      annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/bindplane-node-role
```

Example — GKE Workload Identity on the primary Bindplane Node:

```yaml
spec:
  bindplane:
    serviceAccount:
      annotations:
        iam.gke.io/gcp-service-account: bindplane@my-project.iam.gserviceaccount.com
```

Example — different IAM roles per component:

```yaml
spec:
  bindplane:
    serviceAccount:
      annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/bindplane-node-role
  tsdb:
    serviceAccount:
      annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/bindplane-tsdb-role
```

## Scope

All configuration options above are applied to the following Bindplane services:

- Node
- NATS
- Jobs
- Jobs Migrate (batch/v1 Job — runs at install time and on image upgrades; not a long-running pod)

## Force migration

To force a migration run without changing the image (e.g. after a failed Job), annotate the `Bindplane` resource:

```bash
kubectl patch bindplane <name> -n <namespace> \
  --type=merge \
  -p '{"metadata":{"annotations":{"k8s.bindplane.com/force-migrate":"true"}}}'
```

On the next reconcile the controller **deletes the existing Jobs Migrate Job** (along with its pods), clears this annotation, and resets `status.migratedImage`. It then creates a fresh Jobs Migrate Job via the normal image-change flow. Deleting the existing Job is what allows a retry at an unchanged image — for example, re-running after a failure.

The migrate Job runs with `RestartPolicy: Never` and `backoffLimit: 3`, so each failed attempt runs in its own pod and those pods are retained for log inspection (`kubectl logs <migrate-pod>`) until the Job's TTL (24h) elapses.

## Lifecycle

This section describes the operator's lifecycle contract for the `Bindplane` custom resource: how reconciliation is paused, how owned resources are cleaned up, how status phases are progressed, and how database migrations gate workload rollouts.

### Pause annotation

Set the annotation `k8s.bindplane.com/pause-reconciliation: "true"` on the `Bindplane` resource to suspend all operator reconciliation. While paused:

- No resources are created, updated, or deleted.
- The `Reconciled` condition is set to `False` with `Reason: Paused`.
- The `status.phase` is set to `Paused`.

Remove the annotation (or set it to `"false"`) to resume reconciliation. The operator will re-apply the current desired state on the next reconcile cycle.

```bash
# Pause reconciliation
kubectl annotate bindplane <name> -n <namespace> \
  k8s.bindplane.com/pause-reconciliation=true

# Resume reconciliation
kubectl annotate bindplane <name> -n <namespace> \
  k8s.bindplane.com/pause-reconciliation-
```

### Finalizer and garbage collection

The operator adds a finalizer (`k8s.bindplane.com/finalizer`) to every `Bindplane` CR on the first reconcile. This finalizer ensures the operator has a chance to run cleanup logic before Kubernetes removes the CR from etcd.

When the CR is deleted:

1. The operator sets `status.phase = Deleting` so observers know deletion is in progress.
2. The finalizer is removed, allowing Kubernetes to proceed with object deletion.
3. All namespaced resources owned by the CR (Deployments, StatefulSets, Services, ServiceAccounts, Secrets, ConfigMaps, PodDisruptionBudgets, HPAs, and cert-manager Certificates) are garbage-collected automatically via Kubernetes owner-reference GC — there is no need for explicit cleanup calls in the operator.

Cluster-scoped resources (e.g., ClusterRole, ClusterRoleBinding) are install-time artifacts managed by the operator bundle; they are not owned by the `Bindplane` CR and are not affected by CR deletion.

### Conditions and phases

The operator reports overall state via `status.conditions` and the `status.phase` field.

**Conditions**

| Type | Status | Reason | Meaning |
|------|--------|--------|---------|
| `Reconciled` | `True` | `Reconciled` | All resources reconciled successfully |
| `Reconciled` | `False` | `Paused` | Reconciliation suspended by annotation |
| `Reconciled` | `False` | `Invalid` | CR failed validation; no resources were mutated |
| `Reconciled` | `False` | `MigrationFailed` | The Jobs Migrate Job failed; downstream workloads are blocked |

**Phases**

| Phase | Meaning |
|-------|---------|
| `Pending` | CR was just created; first reconcile has not run yet |
| `ApplyingChanges` | One or more workloads are not yet at their desired replica count |
| `Ready` | All required workloads have their desired replica count ready |
| `Degraded` | CR failed validation; no resources are being managed |
| `Paused` | Reconciliation is suspended via the pause annotation |
| `Deleting` | CR deletion is in progress; finalizer is being removed |

The `status.observedGeneration` field is set to the `metadata.generation` of the CR after every successful reconcile, allowing GitOps tools and `kubectl wait` to detect when the operator has processed the latest spec version.

### Migration contract

When `spec.version` changes (or the `k8s.bindplane.com/force-migrate` annotation is set), the operator:

1. Creates a new `Jobs Migrate` (`batch/v1 Job`) before updating any long-running workloads (Jobs, NATS, Node).
2. Blocks all downstream workload updates until the Jobs Migrate Job completes successfully (requeues every 10 seconds while the Job is still running).
3. On success, records the migrated image in `status.migratedImage` and proceeds to roll out updated workloads.
4. On failure (the Job reaches a terminal `Failed` state after exhausting `backoffLimit`), sets the `Reconciled` condition to `False` with `Reason: MigrationFailed`, sets `status.phase = Degraded`, and **halts the rollout without requeueing** — a terminal Job will not change state on its own, so the operator stops retrying to avoid log spam. The failure remains visible in status until you intervene: inspect the retained failed pods for logs, fix the underlying cause, then change `spec.version` or set the `k8s.bindplane.com/force-migrate` annotation (see [Force migration](#force-migration)) to retry. Either action triggers a fresh reconcile.

This ordering guarantees that the database schema is always compatible with all running workloads before any new binary version is activated.

## Examples

### Minimal configuration

A minimal `Bindplane` custom resource using Secret references for sensitive values:

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane-sample
spec:
  config:
    licenseSecretRef:
      name: bindplane-secrets
      key: license
    auth:
      type: system
      usernameSecretRef:
        name: bindplane-secrets
        key: auth-username
      passwordSecretRef:
        name: bindplane-secrets
        key: auth-password
    store:
      postgres:
        host: test-postgres-rw.postgres.svc.cluster.local
        port: "5432"
        database: testdb
        usernameSecretRef:
          name: bindplane-secrets
          key: pg-username
        passwordSecretRef:
          name: bindplane-secrets
          key: pg-password
```
