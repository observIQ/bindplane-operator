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
      type: postgres
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
      type: postgres
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

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.auth.type` | `BINDPLANE_AUTH_TYPE` | — | No |
| `spec.config.auth.username` | `BINDPLANE_USERNAME` | — | No |
| `spec.config.auth.password` | `BINDPLANE_PASSWORD` | — | No |

## Network

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.network.host` | `BINDPLANE_HOST` | — | No |
| `spec.config.network.port` | `BINDPLANE_PORT` | — | No |
| `spec.config.network.remoteURL` | `BINDPLANE_REMOTE_URL` | `http://<name>-node:3001` | No |

`BINDPLANE_REMOTE_URL` is always set. When `spec.config.network.remoteURL` is not configured, it defaults to the internal node service URL (`http://<bindplane-name>-node:3001`). Override this when the Bindplane UI is accessed through an ingress or load balancer, e.g. `https://bindplane.my-corp.net`.

## Store

| CRD Field | Environment Variable | Default | Required |
|---|---|---|---|
| `spec.config.store.type` | `BINDPLANE_STORE_TYPE` | — | Yes |

Currently only `postgres` is supported for `spec.config.store.type`.

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
