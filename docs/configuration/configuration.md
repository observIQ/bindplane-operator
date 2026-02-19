# Bindplane Configuration

This document describes **Bindplane configuration**—the `spec.config` field and related Bindplane server settings (license, auth, network, store). For the full list of custom resource options (all CRD fields, including component specs and pod templates), see the [API Reference](api.md).

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

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.network.host` | `BINDPLANE_HOST` | — | No |
| `spec.config.network.port` | `BINDPLANE_PORT` | — | No |
| `spec.config.network.remoteURL` | `BINDPLANE_REMOTE_URL` | `http://<name>-node:3001` | No |

`BINDPLANE_REMOTE_URL` is always set. When `spec.config.network.remoteURL` is not configured, it defaults to the internal node service URL (`http://<bindplane-name>-node:3001`). Override this when the Bindplane UI is accessed through an ingress or load balancer, e.g. `https://bindplane.my-corp.net`.

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
| `spec.config.store.postgres.sslmode` | `BINDPLANE_POSTGRES_SSL_MODE` | — | No |
| `spec.config.store.postgres.connectTimeout` | `BINDPLANE_POSTGRES_CONNECT_TIMEOUT` | — | No |
| `spec.config.store.postgres.statementTimeout` | `BINDPLANE_POSTGRES_STATEMENT_TIMEOUT` | — | No |
| `spec.config.store.postgres.maxConnections` | `BINDPLANE_POSTGRES_MAX_CONNECTIONS` | — | No |
| `spec.config.store.postgres.maxLifetime` | `BINDPLANE_POSTGRES_MAX_LIFETIME` | — | No |
| `spec.config.store.postgres.schema` | `BINDPLANE_POSTGRES_SCHEMA` | — | No |

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
