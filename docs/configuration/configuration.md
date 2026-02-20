# Bindplane Configuration

This document describes **Bindplane configuration**—the `spec.config` field and related Bindplane server settings (license, auth, network, store, tracing, metrics). For the full list of custom resource options (all CRD fields, including component specs and pod templates), see the [API Reference](api.md). The API reference is generated from the CRD; run `make generate-api-docs` to regenerate it.

Configuration is provided via the `spec.config` field of the `Bindplane` custom resource. The operator translates these fields into environment variables on the Node, NATS, Jobs, and Jobs Migrate deployments.

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
- [Offline](#offline)
- [Max concurrency](#max-concurrency)
- [Audit trail](#audit-trail)
- [Scope](#scope)
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

## Offline

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.offline` | `BINDPLANE_OFFLINE` | — | No |

## Max concurrency

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.maxConcurrency` | `BINDPLANE_MAX_CONCURRENCY` | `10` | No |

Do not change `maxConcurrency` unless directed by Bindplane support.

## Audit trail

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auditTrail.retentionDays` | `BINDPLANE_AUDIT_TRAIL_RETENTION_DAYS` | `365` | No |

## Scope

All configuration options above are applied to the following Bindplane services:

- Node
- NATS
- Jobs
- Jobs Migrate

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
