# Bindplane Configuration

This document describes **Bindplane configuration**—the `spec.config` field and related Bindplane server settings (license, auth, network, store, tracing, metrics). For the full list of custom resource options (all CRD fields, including component specs and pod templates), see the [API Reference](api.md). The API reference is generated from the CRD; run `make generate-api-docs` to regenerate it. For an overview of TLS and Secret usage (including operator-generated secrets), see [Security: TLS and Secrets](security.md).

Configuration is provided via the `spec.config` field of the `Bindplane` custom resource. The operator translates these fields into environment variables on the Node, NATS, Jobs, and Jobs Migrate workloads.

## Table of contents

- [License](#license)
- [Authentication](#authentication)
  - [System auth](#system-auth)
  - [LDAP and Active Directory](#ldap-and-active-directory)
  - [OIDC](#oidc)
  - [Auth0](#auth0)
- [Network](#network)
- [Store](#store)
  - [PostgreSQL](#postgresql)
- [Tracing](#tracing)
- [Metrics](#metrics)
- [TSDB](#tsdb)
- [Max concurrency](#max-concurrency)
- [Audit trail](#audit-trail)
- [Profiling](#profiling)
- [Pprof](#pprof)
- [Status](#status)
- [Event bus](#event-bus)
- [Analytics](#analytics)
- [Logging](#logging)
- [Advanced](#advanced)
  - [Store stats](#store-stats)
  - [Server](#server)
  - [Rollout](#rollout)
  - [Cache](#cache)
    - [Redis](#redis)
- [Agents](#agents)
  - [Authentication](#agents-authentication)
  - [Heartbeat](#heartbeat)
  - [Rebalance](#rebalance)
  - [Duplication Prevention](#duplication-prevention)
- [Agent versions](#agent-versions)
- [SaaS](#saas)
  - [Stripe](#stripe)
- [Encryption provider](#encryption-provider)
- [Features](#features)
  - [PostHog](#posthog)
  - [Feature overrides](#feature-overrides)
- [Errors](#errors)
- [LLM](#llm)
- [Quotas](#quotas)
- [Extra environment variables](#extra-environment-variables)
  - [Reserved env names](#reserved-env-names)
  - [Pod template vs extraEnv](#pod-template-vs-extraenv)
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

Supported auth types: `system`, `ldap`, `active-directory`, `oidc`, `auth0`.

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

### Auth0

Set `spec.config.auth.type` to `auth0`. The management client secret and WIF client secret support both plain values and Secret references; Secret references take precedence.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.auth0.clientID` | `BINDPLANE_AUTH0_CLIENT_ID` | — | No |
| `spec.config.auth.auth0.domain` | `BINDPLANE_AUTH0_DOMAIN` | — | No |
| `spec.config.auth.auth0.audience` | `BINDPLANE_AUTH0_AUDIENCE` | — | No |
| `spec.config.auth.auth0.managementDomain` | `BINDPLANE_AUTH0_MANAGEMENT_DOMAIN` | — | No |
| `spec.config.auth.auth0.managementClientID` | `BINDPLANE_AUTH0_MANAGEMENT_CLIENT_ID` | — | No |
| `spec.config.auth.auth0.managementClientSecret` | `BINDPLANE_AUTH0_MANAGEMENT_CLIENT_SECRET` | — | No |
| `spec.config.auth.auth0.managementClientSecretSecretRef` | `BINDPLANE_AUTH0_MANAGEMENT_CLIENT_SECRET` | — | No |
| `spec.config.auth.auth0.sso.enabled` | `BINDPLANE_AUTH_AUTH0_SSO_ENABLED` | `false` | No |
| `spec.config.auth.auth0.sso.selfServiceProfileID` | `BINDPLANE_AUTH_AUTH0_SSO_SELF_SERVICE_PROFILE_ID` | — | No |
| `spec.config.auth.auth0.wif.clientID` | `BINDPLANE_AUTH_AUTH0_WIF_CLIENT_ID` | — | No |
| `spec.config.auth.auth0.wif.clientSecret` | `BINDPLANE_AUTH_AUTH0_WIF_CLIENT_SECRET` | — | No |
| `spec.config.auth.auth0.wif.clientSecretSecretRef` | `BINDPLANE_AUTH_AUTH0_WIF_CLIENT_SECRET` | — | No |
| `spec.config.auth.auth0.wif.audience` | `BINDPLANE_AUTH_AUTH0_WIF_AUDIENCE` | — | No |

```yaml
spec:
  config:
    auth:
      type: auth0
      sessionSecretSecretRef:
        name: bindplane-secrets
        key: session-secret
      apiKeySecretRef:
        name: bindplane-secrets
        key: api-key
      auth0:
        clientID: "your-client-id"
        domain: "example.auth0.com"
        audience: "https://api.example.com"
        managementDomain: "example.auth0.com"
        managementClientID: "mgmt-client-id"
        managementClientSecretSecretRef:
          name: bindplane-secrets
          key: auth0-mgmt-client-secret
        sso:
          enabled: true
          selfServiceProfileID: "ssp_xxx"
        wif:
          clientID: "wif-client-id"
          clientSecretSecretRef:
            name: bindplane-secrets
            key: auth0-wif-client-secret
          audience: "https://iam.googleapis.com/projects/..."
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
| `spec.config.network.rateLimits` | (see below) | — | No |

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

**Network Rate Limits:** Configure per-endpoint rate limiting for the REST and GraphQL APIs. Rate fields are decimal strings read by Bindplane as float64 (requests per second).

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.network.rateLimits.apiRate` | `BINDPLANE_NETWORK_RATE_LIMITS_API_RATE` | — | No |
| `spec.config.network.rateLimits.apiBurst` | `BINDPLANE_NETWORK_RATE_LIMITS_API_BURST` | — | No |
| `spec.config.network.rateLimits.graphqlRate` | `BINDPLANE_NETWORK_RATE_LIMITS_GRAPHQL_RATE` | — | No |
| `spec.config.network.rateLimits.graphqlBurst` | `BINDPLANE_NETWORK_RATE_LIMITS_GRAPHQL_BURST` | — | No |

Example (rate limits):

```yaml
spec:
  config:
    network:
      rateLimits:
        apiRate: "10"
        apiBurst: 20
        graphqlRate: "10"
        graphqlBurst: 20
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

Supported types: `otlp`, `google`, `honeycomb`. For `otlp`, configure the `otlp` block with endpoint and optional insecure flag. For `honeycomb`, configure the `honeycomb` block with an API key. You can set a sampling rate (string, e.g. `"0.5"`) between 0 and 1.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.tracing.type` | `BINDPLANE_TRACING_TYPE` | — | No (omit to disable) |
| `spec.config.tracing.otlp.endpoint` | `BINDPLANE_TRACING_OTLP_ENDPOINT` | — | Yes when type is `otlp` |
| `spec.config.tracing.otlp.insecure` | `BINDPLANE_TRACING_OTLP_INSECURE` | `false` | No |
| `spec.config.tracing.honeycomb.apiKey` | `BINDPLANE_TRACING_HONEYCOMB_API_KEY` | — | Yes when type is `honeycomb` |
| `spec.config.tracing.honeycomb.apiKeySecretRef` | `BINDPLANE_TRACING_HONEYCOMB_API_KEY` | — | Yes when type is `honeycomb` (use instead of `apiKey`) |
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

Example (Honeycomb tracing using a Secret reference):

```yaml
spec:
  config:
    tracing:
      type: honeycomb
      honeycomb:
        apiKeySecretRef:
          name: bindplane-honeycomb
          key: api-key
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

## Profiling

Profiling integrates Google Cloud Profiler into Bindplane components. When omitted or `enabled: false`, profiling is off and no profiling environment variables are set. When `enabled: true`, `projectID` is required (enforced by a CRD XValidation rule). The `serviceName` is set automatically per component (e.g. `bindplane-node`); it cannot be overridden via the CRD.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.profiling.enabled` | `BINDPLANE_PROFILING_ENABLED` | `false` | No |
| `spec.config.profiling.projectID` | `BINDPLANE_PROFILING_PROJECT_ID` | — | Yes when `enabled: true` |
| `spec.config.profiling.noCPU` | `BINDPLANE_PROFILING_NO_CPU` | `false` | No |
| `spec.config.profiling.noAlloc` | `BINDPLANE_PROFILING_NO_ALLOC` | `false` | No |
| `spec.config.profiling.noHeap` | `BINDPLANE_PROFILING_NO_HEAP` | `false` | No |
| `spec.config.profiling.noGoroutine` | `BINDPLANE_PROFILING_NO_GOROUTINE` | `false` | No |
| `spec.config.profiling.mutex` | `BINDPLANE_PROFILING_MUTEX` | `false` | No |

Example:

```yaml
spec:
  config:
    profiling:
      enabled: true
      projectID: my-gcp-project
```

## Pprof

Pprof exposes a Go pprof HTTP server on each Bindplane component for CPU and memory profiling. When omitted or `enabled: false`, the server is not started.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.pprof.enabled` | `BINDPLANE_PPROF_ENABLED` | `false` | No |
| `spec.config.pprof.endpoint` | `BINDPLANE_PPROF_ENDPOINT` | `127.0.0.1:6060` | No |

Example:

```yaml
spec:
  config:
    pprof:
      enabled: true
      endpoint: "127.0.0.1:6060"
```

## Status

Status configures the Bindplane status check endpoints. When `enabled: true`, at least one key must be provided via `keys` or `keysSecretRef` (enforced by a CRD XValidation rule). `keysSecretRef` takes precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.status.enabled` | `BINDPLANE_STATUS_ENABLED` | `true` | Yes |
| `spec.config.status.keys` | `BINDPLANE_STATUS_KEYS` | — | Yes when `enabled: true` (or use `keysSecretRef`) |
| `spec.config.status.keysSecretRef` | `BINDPLANE_STATUS_KEYS` | — | Yes when `enabled: true` (or use `keys`) |

Example (direct keys):

```yaml
spec:
  config:
    status:
      enabled: true
      keys:
        - my-status-key
```

Example (Secret reference):

```yaml
spec:
  config:
    status:
      enabled: true
      keysSecretRef:
        name: bindplane-secrets
        key: status-keys
```

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

## Analytics

Analytics configures Bindplane analytics reporting. When `disabled: true`, analytics reporting is turned off. Note that free licenses do not support disabling analytics; this setting is ignored for those license types. Do not set `segmentWriteKey` unless directed by Bindplane support.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.analytics.disabled` | `BINDPLANE_ANALYTICS_DISABLED` | `false` | No |
| `spec.config.analytics.segmentWriteKey` | `BINDPLANE_ANALYTICS_SEGMENT_WRITE_KEY` | — | No |

Example:

```yaml
spec:
  config:
    analytics:
      disabled: true
```

## Logging

Logging configures the log level and output destination for Bindplane components. When `spec.config.logging` is omitted entirely, no logging environment variables are set and Bindplane uses its own internal defaults. The `otlp` block is only relevant when `type` includes `otlp`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.logging.level` | `BINDPLANE_LOGGING_LEVEL` | `info` | No |
| `spec.config.logging.type` | `BINDPLANE_LOGGING_TYPE` | `stdout` | No |
| `spec.config.logging.otlp.endpoint` | `BINDPLANE_LOGGING_OTLP_ENDPOINT` | — | Yes when `type` includes `otlp` |
| `spec.config.logging.otlp.insecure` | `BINDPLANE_LOGGING_OTLP_INSECURE` | `false` | No |
| `spec.config.logging.otlp.interval` | `BINDPLANE_LOGGING_OTLP_INTERVAL` | — | No |

Valid values for `level`: `debug`, `info`, `warn`, `error`.

Valid values for `type`: `stdout`, `otlp`, `stdout,otlp`.

Example (stdout only):

```yaml
spec:
  config:
    logging:
      level: debug
      type: stdout
```

Example (OTLP):

```yaml
spec:
  config:
    logging:
      level: info
      type: otlp
      otlp:
        endpoint: otel-collector.observability.svc:4317
        insecure: true
        interval: "5s"
```

Example (stdout and OTLP):

```yaml
spec:
  config:
    logging:
      level: warn
      type: "stdout,otlp"
      otlp:
        endpoint: otel-collector.observability.svc:4317
        insecure: false
```

## Advanced

Advanced options allow fine-grained control of Bindplane's internal pipelines and distributed cache. They are not required for basic operation. When `spec.config.advanced` is omitted entirely, Bindplane uses its own internal defaults for all of these settings.

### Store stats

The store stats section tunes the measurement ingestion pipeline (how agent metrics are batched and saved to the backend store).

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.advanced.store.stats.batchFlushInterval` | `BINDPLANE_ADVANCED_STORE_STATS_BATCH_FLUSH_INTERVAL` | — | No |
| `spec.config.advanced.store.stats.workerCount` | `BINDPLANE_ADVANCED_STORE_STATS_WORKER_COUNT` | — | No |
| `spec.config.advanced.store.stats.enableSorting` | `BINDPLANE_ADVANCED_STORE_STATS_ENABLE_SORTING` | — | No |
| `spec.config.advanced.store.stats.metricChannelSize` | `BINDPLANE_ADVANCED_STORE_STATS_METRIC_CHANNEL_SIZE` | — | No |
| `spec.config.advanced.store.stats.batchChannelSize` | `BINDPLANE_ADVANCED_STORE_STATS_BATCH_CHANNEL_SIZE` | — | No |

Example:

```yaml
spec:
  config:
    advanced:
      store:
        stats:
          batchFlushInterval: "2s"
          workerCount: 8
          enableSorting: true
          metricChannelSize: 200
          batchChannelSize: 100
```

### Server

The server section configures HTTP and OpAMP server limits.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.advanced.server.maxRequestBytes` | `BINDPLANE_ADVANCED_SERVER_MAX_REQUEST_BYTES` | — | No |
| `spec.config.advanced.server.shutdownGracePeriod` | `BINDPLANE_ADVANCED_SERVER_SHUTDOWN_GRACE_PERIOD` | — | No |
| `spec.config.advanced.server.opampShutdownGracePeriodTarget` | `BINDPLANE_ADVANCED_SERVER_OPAMP_SHUTDOWN_GRACE_PERIOD_TARGET` | — | No |

- `maxRequestBytes`: Maximum request body size (in bytes) the server accepts, excluding offline agent uploads. Bindplane defaults to 10485760 (10 MiB) when omitted.
- `shutdownGracePeriod`: Total time the server waits for in-flight requests and connections to finish before forceful shutdown (e.g. `"5m"`). Valid range: 5s–1h. Bindplane defaults to 2m when omitted. Setting this field also adjusts the Node pod's termination grace period to 125% of this value.
- `opampShutdownGracePeriodTarget`: Fraction of `shutdownGracePeriod` allocated to draining OpAMP agent connections (0.1–1.0, stored as a string, e.g. `"0.6"`). Bindplane defaults to 0.25 when omitted.

Example:

```yaml
spec:
  config:
    advanced:
      server:
        maxRequestBytes: 20971520
        shutdownGracePeriod: "5m"
        opampShutdownGracePeriodTarget: "0.6"
```

### Rollout

The rollout section controls the background rollout updater.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.advanced.rollout.disableUpdater` | `BINDPLANE_ADVANCED_ROLLOUT_DISABLE_UPDATER` | `false` | No |

- `disableUpdater`: When `true`, disables the background rollout updater process.

Example:

```yaml
spec:
  config:
    advanced:
      rollout:
        disableUpdater: true
```

### Cache

The cache section configures the distributed cache backend. Currently only `redis` is supported as the cache type.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.advanced.cache.type` | `BINDPLANE_ADVANCED_CACHE_TYPE` | — | No |

#### Redis

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.advanced.cache.redis.address` | `BINDPLANE_ADVANCED_CACHE_REDIS_ADDRESS` | — | Yes when `type` is `redis` |
| `spec.config.advanced.cache.redis.password` | `BINDPLANE_ADVANCED_CACHE_REDIS_PASSWORD` | — | No |
| `spec.config.advanced.cache.redis.passwordSecretRef` | `BINDPLANE_ADVANCED_CACHE_REDIS_PASSWORD` | — | No |
| `spec.config.advanced.cache.redis.db` | `BINDPLANE_ADVANCED_CACHE_REDIS_DB` | — | No |
| `spec.config.advanced.cache.redis.readTimeout` | `BINDPLANE_ADVANCED_CACHE_REDIS_READ_TIMEOUT` | — | No |
| `spec.config.advanced.cache.redis.writeTimeout` | `BINDPLANE_ADVANCED_CACHE_REDIS_WRITE_TIMEOUT` | — | No |
| `spec.config.advanced.cache.redis.enableTLS` | `BINDPLANE_ADVANCED_CACHE_REDIS_ENABLE_TLS` | — | No |
| `spec.config.advanced.cache.redis.tls.secretName` | (mounts Secret) | — | No |
| `spec.config.advanced.cache.redis.tls.certKey` | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_CERT` | — | No |
| `spec.config.advanced.cache.redis.tls.keyKey` | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_KEY` | — | No |
| `spec.config.advanced.cache.redis.tls.caKey` | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_TLS_CA` | — | No |
| `spec.config.advanced.cache.redis.tls.skipVerify` | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_TLS_SKIP_VERIFY` | — | No |
| `spec.config.advanced.cache.redis.tls.minTLSVersion` | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_MIN_TLSVERSION` | — | No |

`passwordSecretRef` takes precedence over `password` when both are set.

When `tls.secretName` is set, the operator mounts the Secret at `/etc/bindplane/advanced-cache-redis-tls` and sets the TLS env vars to the corresponding file paths. Specify only the Secret name and key names; the operator manages the mount path.

Example (Redis without TLS):

```yaml
spec:
  config:
    advanced:
      cache:
        type: redis
        redis:
          address: redis.default.svc:6379
          passwordSecretRef:
            name: redis-credentials
            key: password
          db: 1
          readTimeout: "5s"
          writeTimeout: "5s"
```

Example (Redis with TLS):

```yaml
spec:
  config:
    advanced:
      cache:
        type: redis
        redis:
          address: redis.default.svc:6379
          passwordSecretRef:
            name: redis-credentials
            key: password
          enableTLS: true
          tls:
            secretName: redis-tls
            certKey: tls.crt
            keyKey: tls.key
            caKey: ca.crt
            minTLSVersion: "1.3"
```

## Agents

The `spec.config.agents` section configures how Bindplane communicates with agents, including heartbeat timing, rebalancing, and authentication. When omitted, Bindplane uses its own defaults.

### Agents Authentication

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.auth.type` | `BINDPLANE_AGENTS_AUTH_TYPE` | `secretKey` | No |
| `spec.config.agents.auth.secretKey.headers` | `BINDPLANE_AGENTS_AUTH_SECRET_KEY_HEADERS` | `X-Bindplane-Authorization,Authorization` | No |
| `spec.config.agents.auth.oauth.issuer` | `BINDPLANE_AGENTS_AUTH_OAUTH_ISSUER` | — | No |
| `spec.config.agents.auth.oauth.audiences` | `BINDPLANE_AGENTS_AUTH_OAUTH_AUDIENCES` | — | No |
| `spec.config.agents.auth.oauth.requiredClaims` | `BINDPLANE_AGENTS_AUTH_OAUTH_REQUIRED_CLAIMS` | — | No |
| `spec.config.agents.auth.oauth.requiredScopes` | `BINDPLANE_AGENTS_AUTH_OAUTH_REQUIRED_SCOPES` | — | No |
| `spec.config.agents.auth.oauth.cacheTTL` | `BINDPLANE_AGENTS_AUTH_OAUTH_CACHE_TTL` | `1h` | No |

`auth.type` accepts a single value or a comma-separated list (e.g. `"oauth,secretKey"`).
`[]string` fields (headers, audiences, requiredClaims, requiredScopes) are comma-separated in the env var.

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

### Connections

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.maxSimultaneousConnections` | `BINDPLANE_AGENTS_MAX_SIMULTANEOUS_CONNECTIONS` | `10` | No |
| `spec.config.agents.enableConnectionRegistryMiddleware` | `BINDPLANE_AGENTS_ENABLE_CONNECTION_REGISTRY_MIDDLEWARE` | `false` | No |
| `spec.config.agents.connectionRegistryHeartbeatInterval` | `BINDPLANE_AGENTS_CONNECTION_REGISTRY_HEARTBEAT_INTERVAL` | `15s` | No |
| `spec.config.agents.connectionRegistryStaleDuration` | `BINDPLANE_AGENTS_CONNECTION_REGISTRY_STALE_DURATION` | `45s` | No |
| `spec.config.agents.connectionRegistryLockTimeout` | `BINDPLANE_AGENTS_CONNECTION_REGISTRY_LOCK_TIMEOUT` | `2s` | No |
| `spec.config.agents.connectionClaimTimeout` | `BINDPLANE_AGENTS_CONNECTION_CLAIM_TIMEOUT` | `3s` | No |

See [Max concurrency](#max-concurrency) for details. Do not modify unless directed by Bindplane support.

`rebalancePercentage` and `rebalanceJitter` are integers in the range 0–100. A value of 0 disables that feature.

`connectionRegistryStaleDuration` must be greater than `connectionRegistryHeartbeatInterval`. `connectionClaimTimeout` must be greater than `connectionRegistryLockTimeout`. These fields only take effect when `enableConnectionRegistryMiddleware` is `true`.

### Duplication Prevention

`spec.config.agents.duplicationPrevention` configures detection and handling of duplicate agent connections (the same agent connecting from multiple sources). All fields are optional; defaults are shown in the table below.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agents.duplicationPrevention.enableMiddleware` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_ENABLE_MIDDLEWARE` | `false` | No |
| `spec.config.agents.duplicationPrevention.reassignID` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_REASSIGN_ID` | `false` | No |
| `spec.config.agents.duplicationPrevention.detectionStrategy` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_DETECTION_STRATEGY` | — | No |
| `spec.config.agents.duplicationPrevention.detectionGracePeriod` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_DETECTION_GRACE_PERIOD` | `3m` | No |
| `spec.config.agents.duplicationPrevention.minGracePeriodFailures` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_MIN_GRACE_PERIOD_FAILURES` | `3` | No |
| `spec.config.agents.duplicationPrevention.retryAfter` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_RETRY_AFTER` | `30s` | No |
| `spec.config.agents.duplicationPrevention.maxReassignmentAttempts` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_MAX_REASSIGNMENT_ATTEMPTS` | `3` | No |
| `spec.config.agents.duplicationPrevention.reassignmentCacheTTL` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_REASSIGNMENT_CACHE_TTL` | `24h` | No |
| `spec.config.agents.duplicationPrevention.reassignmentRetryAfter` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_REASSIGNMENT_RETRY_AFTER` | `5m` | No |
| `spec.config.agents.duplicationPrevention.enableDuplicateNotifications` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_ENABLE_DUPLICATE_NOTIFICATIONS` | `false` | No |
| `spec.config.agents.duplicationPrevention.enablePerOrgEnforcement` | `BINDPLANE_AGENTS_DUPLICATE_PREVENTION_ENABLE_PER_ORG_ENFORCEMENT` | `false` | No |

- `detectionStrategy` must be `grace_period` when set.
- `detectionGracePeriod` must be >= `connectionRegistryStaleDuration`. Used with the `grace_period` strategy.
- `minGracePeriodFailures`: both the grace period duration and this count must be satisfied before confirming a duplicate (minimum 1).
- `retryAfter`: Retry-After header value sent during the grace period window.
- `maxReassignmentAttempts`: valid range 1–10.
- `reassignmentCacheTTL`: maximum 7d.
- `reassignmentRetryAfter`: maximum 1h; sent when permanently rejecting after max attempts.

```yaml
spec:
  config:
    agents:
      auth:
        type: "oauth,secretKey"
        oauth:
          issuer: "https://auth.example.com"
          audiences:
            - "https://api.example.com"
          cacheTTL: "2h"
      heartbeatInterval: "45s"
      heartbeatTTL: "2m"
      rebalanceInterval: "30m"
      rebalancePercentage: 50
      enableConnectionRegistryMiddleware: true
      connectionRegistryHeartbeatInterval: "20s"
      connectionRegistryStaleDuration: "60s"
      connectionRegistryLockTimeout: "3s"
      connectionClaimTimeout: "5s"
      duplicationPrevention:
        enableMiddleware: true
        reassignID: true
        detectionStrategy: grace_period
        detectionGracePeriod: "3m"
        minGracePeriodFailures: 3
        retryAfter: "30s"
        maxReassignmentAttempts: 3
        reassignmentCacheTTL: "24h"
        reassignmentRetryAfter: "5m"
        enableDuplicateNotifications: true
        enablePerOrgEnforcement: true
```

## Agent versions

The `spec.config.agentVersions` section configures how Bindplane syncs agent version metadata.
When omitted, Bindplane uses its own defaults.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.agentVersions.syncInterval` | `BINDPLANE_AGENT_VERSIONS_SYNC_INTERVAL` | `1h` | No |
| `spec.config.agentVersions.clients` | `BINDPLANE_AGENT_VERSIONS_CLIENTS` | — | No |

`syncInterval` must be at least `1h` (enforced by Bindplane at runtime).
`clients` is a deprecated field; version clients are now configured per-agent-type via AgentType resources.
`clients` is a comma-separated list of version client identifiers (e.g. `"bdot,github"`).

```yaml
spec:
  config:
    agentVersions:
      syncInterval: "2h"
      clients:
        - bdot
        - github
```

## SaaS

The `spec.config.saas` section configures Bindplane SaaS-specific functionality including the license server, Stripe billing, and janitor settings.
When omitted, SaaS mode is disabled.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.saas.enabled` | `BINDPLANE_SAAS_ENABLED` | `false` | No |
| `spec.config.saas.licenseServerAddress` | `BINDPLANE_SAAS_LICENSE_SERVER_ADDRESS` | — | No |
| `spec.config.saas.licenseServerAPIKey` | `BINDPLANE_SAAS_LICENSE_SERVER_API_KEY` | — | No |
| `spec.config.saas.licenseServerAPIKeySecretRef` | `BINDPLANE_SAAS_LICENSE_SERVER_API_KEY` | — | No |
| `spec.config.saas.janitorOrganization` | `BINDPLANE_SAAS_JANITOR_ORGANIZATION` | — | No |
| `spec.config.saas.useStagePublicRSAKey` | `BINDPLANE_SAAS_USE_STAGE_PUBLIC_RSA_KEY` | `false` | No |

`licenseServerAPIKeySecretRef` takes precedence over `licenseServerAPIKey` when both are set.

### Stripe

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.saas.stripe.enabled` | `BINDPLANE_SAAS_STRIPE_ENABLED` | `false` | No |
| `spec.config.saas.stripe.secretKey` | `BINDPLANE_SAAS_STRIPE_SECRET_KEY` | — | No |
| `spec.config.saas.stripe.secretKeySecretRef` | `BINDPLANE_SAAS_STRIPE_SECRET_KEY` | — | No |
| `spec.config.saas.stripe.publishableKey` | `BINDPLANE_SAAS_STRIPE_PUBLISHABLE_KEY` | — | No |
| `spec.config.saas.stripe.publishableKeySecretRef` | `BINDPLANE_SAAS_STRIPE_PUBLISHABLE_KEY` | — | No |
| `spec.config.saas.stripe.webhookSecret` | `BINDPLANE_SAAS_STRIPE_WEBHOOK_SECRET` | — | No |
| `spec.config.saas.stripe.webhookSecretSecretRef` | `BINDPLANE_SAAS_STRIPE_WEBHOOK_SECRET` | — | No |
| `spec.config.saas.stripe.growthPlanIDs.baseRate` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_IDS_BASE_RATE` | — | No |
| `spec.config.saas.stripe.growthPlanIDs.usageRates` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_IDS_USAGE_RATES` | — | No |
| `spec.config.saas.stripe.growthPlanMeterNames.logs` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_METER_NAMES_LOGS` | — | No |
| `spec.config.saas.stripe.growthPlanMeterNames.metrics` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_METER_NAMES_METRICS` | — | No |
| `spec.config.saas.stripe.growthPlanMeterNames.traces` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_METER_NAMES_TRACES` | — | No |
| `spec.config.saas.stripe.growthPlanMeterNames.collectors` | `BINDPLANE_SAAS_STRIPE_GROWTH_PLAN_METER_NAMES_COLLECTORS` | — | No |

Secret references for Stripe keys take precedence over plain-value fields when both are set. Prefer Secret references in production.

## Features

The `spec.config.features` section configures the feature flag backend and enables optional feature overrides.
When omitted, feature flags are disabled and all overrides default to `false`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.features.type` | `BINDPLANE_FEATURES_TYPE` | — | No |

`type` accepts: `posthog`.

### PostHog

PostHog is the supported feature flag backend. When `type: posthog` is set, configure the connection via the fields below. API keys may be provided as plain values or as Secret references; the Secret reference takes precedence when both are set.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.features.postHog.projectAPIKey` | `BINDPLANE_FEATURES_POSTHOG_PROJECT_API_KEY` | — | No |
| `spec.config.features.postHog.projectAPIKeySecretRef` | `BINDPLANE_FEATURES_POSTHOG_PROJECT_API_KEY` | — | No |
| `spec.config.features.postHog.personalAPIKey` | `BINDPLANE_FEATURES_POSTHOG_PERSONAL_API_KEY` | — | No |
| `spec.config.features.postHog.personalAPIKeySecretRef` | `BINDPLANE_FEATURES_POSTHOG_PERSONAL_API_KEY` | — | No |
| `spec.config.features.postHog.endpoint` | `BINDPLANE_FEATURES_POSTHOG_ENDPOINT` | — | No |
| `spec.config.features.postHog.defaultFeatureFlagsPollingInterval` | `BINDPLANE_FEATURES_POSTHOG_DEFAULT_FEATURE_FLAGS_POLLING_INTERVAL` | — | No |

### Feature overrides

Feature overrides allow individual features to be force-enabled regardless of the feature flag backend result.
All override fields are boolean and default to `false`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.features.overrides.growthLicense` | `BINDPLANE_FEATURES_OVERRIDES_GROWTH_LICENSE` | `false` | No |
| `spec.config.features.overrides.secopsTheme` | `BINDPLANE_FEATURES_OVERRIDES_SECOPS_THEME` | `false` | No |
| `spec.config.features.overrides.secopsIntegration` | `BINDPLANE_FEATURES_OVERRIDES_SECOPS_INTEGRATION` | `false` | No |
| `spec.config.features.overrides.llmFeatures` | `BINDPLANE_FEATURES_OVERRIDES_LLM_FEATURES` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligence` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligenceSnapshotLogTypes` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE_SNAPSHOT_LOG_TYPES` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligenceOtelConfigImport` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE_OTEL_CONFIG_IMPORT` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligenceChronicleForwarderConfigImport` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE_CHRONICLE_FORWARDER_CONFIG_IMPORT` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligenceParseField` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE_PARSE_FIELD` | `false` | No |
| `spec.config.features.overrides.pipelineIntelligenceGenerateProcessors` | `BINDPLANE_FEATURES_OVERRIDES_PIPELINE_INTELLIGENCE_GENERATE_PROCESSORS` | `false` | No |
| `spec.config.features.overrides.rawConfigLegacy` | `BINDPLANE_FEATURES_OVERRIDES_RAW_CONFIG_LEGACY` | `false` | No |
| `spec.config.features.overrides.notifications` | `BINDPLANE_FEATURES_OVERRIDES_NOTIFICATIONS` | `false` | No |

## Errors

The `spec.config.errors` section configures error tracking (e.g., BetterStack). When omitted, error tracking is disabled.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.errors.enabled` | `BINDPLANE_ERRORS_ENABLED` | `false` | No |
| `spec.config.errors.backendDSN` | `BINDPLANE_ERRORS_BACKEND_DSN` | — | No |
| `spec.config.errors.frontendDSN` | `BINDPLANE_ERRORS_FRONTEND_DSN` | — | No |
| `spec.config.errors.environment` | `BINDPLANE_ERRORS_ENVIRONMENT` | — | No |

`backendDSN` and `frontendDSN` are service-specific DSNs (e.g., BetterStack ingest DSN). `environment` is reported to the error tracking service (e.g., `"production"`, `"staging"`).
## LLM

The `spec.config.llm` section configures large language model integrations. When omitted, LLM features are disabled.

### Gemini

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.llm.gemini.projectID` | `BINDPLANE_LLM_GEMINI_PROJECT_ID` | — | No |
| `spec.config.llm.gemini.location` | `BINDPLANE_LLM_GEMINI_LOCATION` | — | No |
| `spec.config.llm.gemini.vectorSearchRedis.address` | `BINDPLANE_LLM_GEMINI_VECTOR_SEARCH_REDIS_ADDRESS` | — | No |
| `spec.config.llm.gemini.vectorSearchRedis.enableTLS` | `BINDPLANE_LLM_GEMINI_VECTOR_SEARCH_REDIS_ENABLE_TLS` | `false` | No |

### LangSmith

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.llm.langsmith.enabled` | `BINDPLANE_LLM_LANGSMITH_ENABLED` | `false` | No |
| `spec.config.llm.langsmith.apiKey` | `BINDPLANE_LLM_LANGSMITH_API_KEY` | — | No |
| `spec.config.llm.langsmith.apiKeySecretRef` | `BINDPLANE_LLM_LANGSMITH_API_KEY` | — | No |
| `spec.config.llm.langsmith.projectName` | `BINDPLANE_LLM_LANGSMITH_PROJECT_NAME` | — | No |

### OpenAI

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.llm.openai.apiKey` | `BINDPLANE_LLM_OPENAI_API_KEY` | — | No |
| `spec.config.llm.openai.apiKeySecretRef` | `BINDPLANE_LLM_OPENAI_API_KEY` | — | No |

### Anthropic

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.llm.anthropic.apiKey` | `BINDPLANE_LLM_ANTHROPIC_API_KEY` | — | No |
| `spec.config.llm.anthropic.apiKeySecretRef` | `BINDPLANE_LLM_ANTHROPIC_API_KEY` | — | No |

For `apiKey` and `apiKeySecretRef`, the Secret reference takes precedence when both are set. Prefer Secret references in production.

## Quotas

The `spec.config.quotas` section configures usage quota enforcement. When omitted, Bindplane uses its own defaults.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.quotas.enabled` | `BINDPLANE_QUOTAS_ENABLED` | `false` | No |
| `spec.config.quotas.enforced` | `BINDPLANE_QUOTAS_ENFORCED` | `false` | No |
| `spec.config.quotas.projects.enabled` | `BINDPLANE_QUOTAS_PROJECTS_ENABLED` | `false` | No |
| `spec.config.quotas.projects.enforced` | `BINDPLANE_QUOTAS_PROJECTS_ENFORCED` | `false` | No |

`enabled` activates quota tracking; `enforced` causes violations to be rejected rather than only logged. The same semantics apply to `projects` (per-project quotas).

```yaml
spec:
  config:
    saas:
      enabled: true
      licenseServerAddress: "https://license.bindplane.com"
      licenseServerAPIKeySecretRef:
        name: bindplane-secrets
        key: saas-license-api-key
      stripe:
        enabled: true
        secretKeySecretRef:
          name: bindplane-secrets
          key: stripe-secret-key
        publishableKeySecretRef:
          name: bindplane-secrets
          key: stripe-publishable-key
        webhookSecretSecretRef:
          name: bindplane-secrets
          key: stripe-webhook-secret
        growthPlanIDs:
          baseRate: "price_xxx"
          usageRates: "price_yyy,price_zzz"
        growthPlanMeterNames:
          logs: "log_volume"
          metrics: "metric_volume"
          traces: "trace_volume"
          collectors: "collector_count"
```

## Encryption provider

The `spec.config.encryptionProvider` section configures at-rest encryption of sensitive store data. When omitted, Bindplane uses its built-in encryption.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.encryptionProvider.type` | `BINDPLANE_ENCRYPTIONPROVIDER_TYPE` | — | No |
| `spec.config.encryptionProvider.googleKMS.projectID` | `BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_PROJECTID` | — | No |
| `spec.config.encryptionProvider.googleKMS.location` | `BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_LOCATION` | — | No |
| `spec.config.encryptionProvider.googleKMS.keyRotationPeriod` | `BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_KEY_ROTATION_PERIOD` | — | No |
| `spec.config.encryptionProvider.googleKMS.keyDeletionJob` | `BINDPLANE_ENCRYPTIONPROVIDER_GOOGLEKMS_KEY_DELETION_JOB` | `false` | No |

`type` must be `googleKMS` when set. `keyDeletionJob` is injected only into the **Jobs Migrate** workload — it is not set on Node, NATS, or Jobs.

```yaml
spec:
  config:
    encryptionProvider:
      type: googleKMS
      googleKMS:
        projectID: my-gcp-project
        location: us-central1
        keyRotationPeriod: 30d
        keyDeletionJob: true
```

```yaml
spec:
  config:
    features:
      type: posthog
      postHog:
        projectAPIKeySecretRef:
          name: bindplane-secrets
          key: posthog-project-api-key
        personalAPIKeySecretRef:
          name: bindplane-secrets
          key: posthog-personal-api-key
        endpoint: "https://app.posthog.com"
        defaultFeatureFlagsPollingInterval: "30s"
      overrides:
        growthLicense: true
        pipelineIntelligence: true
```

```yaml
spec:
  config:
    errors:
      enabled: true
      backendDSN: "https://errors.example.com/backend-dsn"
      frontendDSN: "https://errors.example.com/frontend-dsn"
      environment: production
    llm:
      gemini:
        projectID: my-gcp-project
        location: us-central1
        vectorSearchRedis:
          address: "redis.example.com:6379"
          enableTLS: true
      langsmith:
        enabled: true
        apiKeySecretRef:
          name: llm-secrets
          key: langsmith-api-key
        projectName: bindplane-prod
      openai:
        apiKeySecretRef:
          name: llm-secrets
          key: openai-api-key
      anthropic:
        apiKeySecretRef:
          name: llm-secrets
          key: anthropic-api-key
    quotas:
      enabled: true
      enforced: true
      projects:
        enabled: true
        enforced: false
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

The controller clears this annotation and resets `status.migratedImage` on the next reconcile, then creates a new Jobs Migrate Job via the normal image-change flow.

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
