# Bindplane Configuration

Configuration is provided via the `spec.config` field of the `Bindplane` custom resource. The operator translates these fields into environment variables on the Node, NATS, Jobs, and Jobs Migrate deployments.

## Sensitive Fields and Secret References

The following fields are considered sensitive. Each supports two options:

1. **Direct value** — set the field inline in the CR (simpler, but the value is visible in the CR and etcd).
2. **Secret reference** — reference a key in a Kubernetes Secret. The sibling `*Secret` field takes precedence when both are set.

| Sensitive Field | Secret Reference Field |
|---|---|
| `spec.config.license` | `spec.config.licenseSecret` |
| `spec.config.auth.username` | `spec.config.auth.usernameSecret` |
| `spec.config.auth.password` | `spec.config.auth.passwordSecret` |
| `spec.config.auth.ldap.bindPassword` | `spec.config.auth.ldap.bindPasswordSecret` |
| `spec.config.auth.oidc.clientID` | `spec.config.auth.oidc.clientIDSecret` |
| `spec.config.auth.oidc.clientSecret` | `spec.config.auth.oidc.clientSecretSecret` |
| `spec.config.store.postgres.username` | `spec.config.store.postgres.usernameSecret` |
| `spec.config.store.postgres.password` | `spec.config.store.postgres.passwordSecret` |

The `*Secret` fields follow the standard Kubernetes `SecretKeySelector` shape — `name` (Secret name) and `key` (key within the Secret).

### Example: direct values

```yaml
spec:
  config:
    license: "my-license-key"
    auth:
      username: admin
      password: "my-password"
    store:
      postgres:
        host: postgres.example.com
        username: bindplane
        password: "my-pg-password"
```

### Example: Secret references

```yaml
# Secret
apiVersion: v1
kind: Secret
metadata:
  name: bindplane-secrets
  namespace: bindplane
stringData:
  license: "my-license-key"
  auth-username: admin
  auth-password: "my-password"
  pg-username: bindplane
  pg-password: "my-pg-password"
---
# Bindplane CR
spec:
  config:
    licenseSecret:
      name: bindplane-secrets
      key: license
    auth:
      usernameSecret:
        name: bindplane-secrets
        key: auth-username
      passwordSecret:
        name: bindplane-secrets
        key: auth-password
    store:
      postgres:
        host: postgres.example.com
        usernameSecret:
          name: bindplane-secrets
          key: pg-username
        passwordSecret:
          name: bindplane-secrets
          key: pg-password
```

When a Secret reference is set, the kubelet resolves the secret value at pod start-up. If the referenced Secret or key does not exist, Kubernetes will surface a `CreateContainerConfigError` event on the pod.

## License

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.license` | `BINDPLANE_LICENSE` | — | Yes |

## Authentication

Supported auth types: `system`, `ldap`, `active-directory`, `oidc`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.type` | `BINDPLANE_AUTH_TYPE` | — | No |
| `spec.config.auth.sessionsStrictMode` | `BINDPLANE_AUTH_SESSIONS_STRICT_MODE` | `false` | No |

### System auth

Set `spec.config.auth.type` to `system` for basic username/password authentication.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.username` | `BINDPLANE_USERNAME` | — | No |
| `spec.config.auth.password` | `BINDPLANE_PASSWORD` | — | No |

### LDAP and Active Directory

Set `spec.config.auth.type` to `ldap` or `active-directory`. Both types share the same `ldap` configuration block.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.ldap.protocol` | `BINDPLANE_LDAP_PROTOCOL` | — | Yes (`ldap` or `ldaps`) |
| `spec.config.auth.ldap.server` | `BINDPLANE_LDAP_SERVER` | — | Yes |
| `spec.config.auth.ldap.port` | `BINDPLANE_LDAP_PORT` | — | Yes |
| `spec.config.auth.ldap.baseDN` | `BINDPLANE_LDAP_BASE_DN` | — | Yes |
| `spec.config.auth.ldap.bindUser` | `BINDPLANE_LDAP_BIND_USER` | — | No |
| `spec.config.auth.ldap.bindPassword` | `BINDPLANE_LDAP_BIND_PASSWORD` | — | No |
| `spec.config.auth.ldap.searchFilter` | `BINDPLANE_LDAP_SEARCH_FILTER` | — | No |
| `spec.config.auth.ldap.tlsCert` | `BINDPLANE_LDAP_TLS_CERT` | — | No |
| `spec.config.auth.ldap.tlsKey` | `BINDPLANE_LDAP_TLS_KEY` | — | No |
| `spec.config.auth.ldap.tlsCA` | `BINDPLANE_LDAP_TLS_CA` | — | No |
| `spec.config.auth.ldap.tlsSkipVerify` | `BINDPLANE_LDAP_TLS_SKIP_VERIFY` | `false` | No |

Example (LDAP with TLS):

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
        bindPasswordSecret:
          name: ldap-secrets
          key: bind-password
        tlsCA: /etc/ssl/certs/ca.pem
```

Example (Active Directory):

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
        bindUser: DOMAIN\svc-bindplane
        bindPasswordSecret:
          name: ad-secrets
          key: bind-password
```

### OIDC

Set `spec.config.auth.type` to `oidc`.

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.oidc.clientID` | `BINDPLANE_OIDC_OAUTH2_CLIENT_ID` | — | Yes |
| `spec.config.auth.oidc.clientSecret` | `BINDPLANE_OIDC_OAUTH2_CLIENT_SECRET` | — | Yes |
| `spec.config.auth.oidc.issuer` | `BINDPLANE_OIDC_ISSUER` | — | Yes |
| `spec.config.auth.oidc.scopes` | `BINDPLANE_OIDC_SCOPES` | — | Yes (comma-separated) |

Example:

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
        clientIDSecret:
          name: oidc-secrets
          key: client-id
        clientSecretSecret:
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

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.store.postgres.host` | `BINDPLANE_POSTGRES_HOST` | — | Yes |
| `spec.config.store.postgres.port` | `BINDPLANE_POSTGRES_PORT` | — | No |
| `spec.config.store.postgres.database` | `BINDPLANE_POSTGRES_DATABASE` | — | No |
| `spec.config.store.postgres.username` | `BINDPLANE_POSTGRES_USERNAME` | — | No |
| `spec.config.store.postgres.password` | `BINDPLANE_POSTGRES_PASSWORD` | — | No |
| `spec.config.store.postgres.sslmode` | `BINDPLANE_POSTGRES_SSL_MODE` | — | No |
| `spec.config.store.postgres.connectTimeout` | `BINDPLANE_POSTGRES_CONNECT_TIMEOUT` | — | No |
| `spec.config.store.postgres.statementTimeout` | `BINDPLANE_POSTGRES_STATEMENT_TIMEOUT` | — | No |
| `spec.config.store.postgres.maxConnections` | `BINDPLANE_POSTGRES_MAX_CONNECTIONS` | — | No |
| `spec.config.store.postgres.maxLifetime` | `BINDPLANE_POSTGRES_MAX_LIFETIME` | — | No |
| `spec.config.store.postgres.schema` | `BINDPLANE_POSTGRES_SCHEMA` | — | No |

## Scope

All configuration options above are applied to the following Bindplane services:

- Node
- NATS
- Jobs
- Jobs Migrate
