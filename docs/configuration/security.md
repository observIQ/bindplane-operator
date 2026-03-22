# Security: TLS and Secrets

This document describes how the Bindplane operator uses **TLS** and **Secrets** across all components. It covers both user-configurable options (detailed in [Configuration](configuration.md)) and operator-generated secrets.

## Overview

- **User-configurable:** You supply Secret names and key names (or direct values) in `spec.config`. The operator mounts those Secrets where needed and sets environment variables to the mounted file paths or injects values from Secrets. See [Configuration](configuration.md) for all options.
- **Operator-generated:** The operator creates and owns certain Secrets (e.g. TSDB basic auth for the default Prometheus-backed TSDB). These are not user-configurable; the operator generates credentials and uses them consistently across pods.

## User-configurable TLS and Secrets

The following are configured via the Bindplane custom resource and documented in [Configuration](configuration.md):

| Area | What you configure | How the operator uses it |
|------|--------------------|---------------------------|
| **License** | `spec.config.license` or `licenseSecretRef` | Injects `BINDPLANE_LICENSE` from value or Secret into Node, Jobs, NATS, and the Jobs Migrate Job. |
| **Authentication** | System auth (`username`/`password` or Secret refs), LDAP bind user/password and [LDAP TLS](configuration.md#ldap-and-active-directory), OIDC client ID/secret or Secret refs, agent auth (`spec.config.agents.auth`) | Sets auth-related env vars; mounts LDAP TLS Secret when `spec.config.auth.ldap.tls` is set. |
| **Network TLS** | [Network TLS](configuration.md#network): `spec.config.network.tls` (secretName, certKey, keyKey, caKey, minVersion, skipVerify) | Mounts the Secret at a fixed path and sets `BINDPLANE_TLS_*` env vars to the mounted file paths. Used when you want server-side or mutual TLS on the Bindplane server (often omitted when using [Ingress or Gateway API](configuration.md#network) to terminate TLS). |
| **PostgreSQL** | Postgres username/password (or Secret refs) and [PostgreSQL TLS](configuration.md#postgresql) (`spec.config.store.postgres.tls`, sslmode) | Injects credentials; mounts Postgres TLS Secret when TLS is configured and sets `BINDPLANE_POSTGRES_SSL_*` env vars. |
| **Metrics (Prometheus)** | Optional basic auth for the HTTP endpoint where Bindplane **exposes** its own metrics via `spec.config.metrics.prometheus.username` and `password` or `passwordSecretRef` | Sets `BINDPLANE_METRICS_PROMETHEUS_USERNAME` / `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` when configured. Distinct from Bindplane TSDB remote write auth (below). |
| **Redis TLS** | `spec.config.advanced.cache.redis.tls` (secretName, certKey, keyKey, caKey, skipVerify, minTLSVersion) | Mounts the Secret and sets `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_*` env vars to the mounted file paths. Only applicable when Redis is used as the distributed cache backend. |

Details, field names, and examples for all of the above are in [Configuration](configuration.md).

## Operator-generated Secrets

### TSDB basic auth (default Prometheus deployment)

For the default operator-managed TSDB (implemented with Prometheus), the operator enables basic authentication on the TSDB endpoint. Credentials are **not** user-configurable.

- The operator generates a username and password and stores them in a Secret named **`<bindplane-name>-tsdb-basic-auth`**.
- The Secret contains: `username`, `password`, and `web-config` (Prometheus `basic_auth_users` YAML used for the TSDB web config file).
- The Secret is **created once** when it does not exist; the operator does not update its data on later reconciles, so credentials are not rotated unexpectedly.
- The TSDB pod (Prometheus) is configured with `--web.config.file` pointing at the mounted `web-config` content, so the TSDB UI/API require basic auth.
- All Bindplane workloads (Node, Jobs, NATS, and the Jobs Migrate Job) receive the same username and password via **`BINDPLANE_PROMETHEUS_AUTH_USERNAME`** and **`BINDPLANE_PROMETHEUS_AUTH_PASSWORD`** (from that Secret) so they can authenticate when using the remote write client to send agent throughput and health metrics to the TSDB. These are **not** the same as `BINDPLANE_METRICS_PROMETHEUS_USERNAME` / `BINDPLANE_METRICS_PROMETHEUS_PASSWORD`, which configure basic auth for the endpoint where Bindplane **exposes** its own metrics (see [Configuration – Metrics](configuration.md#metrics)).

To retrieve the password for manual access (e.g. to open the TSDB UI in a browser):

```bash
kubectl get secret <bindplane-name>-tsdb-basic-auth -n <namespace> -o jsonpath='{.data.password}' | base64 -d
```

## Cert-manager TLS (optional)

You can use [cert-manager](https://cert-manager.io/) to have the operator automatically issue and rotate TLS certificates for selected in-cluster interfaces, instead of supplying your own TLS Secrets.

**Scope:** cert-manager integration is supported for two in-cluster interfaces:

- **TSDB remote write** (Bindplane → Prometheus): configured via `spec.tsdb.tls.certManager` (server cert) and `spec.config.tsdb.tls.certManager` (client cert). When both are set, mTLS is enabled.
- **NATS** (Bindplane → NATS, and NATS → NATS cluster): configured via `spec.config.nats.tls.certManager`. A single cert with both `ServerAuth` and `ClientAuth` EKUs is issued and used for the client port (4222), cluster port (6222), and HTTP monitoring port (8222). This is **cert-manager only** — there is no user-provided secret path for NATS TLS.

cert-manager integration does **not** apply to:

- Bindplane’s primary HTTP interface (port 3001) — use `spec.config.network.tls` with a user-provided Secret
- Bindplane’s connection to PostgreSQL — use `spec.config.store.postgres.tls` with a user-provided Secret
- Bindplane’s connection to the Transform Agent — no TLS support

### TSDB remote write mTLS

Enabling it turns on mTLS for the Bindplane → TSDB path: the operator creates cert-manager `Certificate` resources for a TSDB server cert, a TSDB probe client cert (used by Prometheus’s own exec probes), and a Bindplane client cert; mounts the issued certs; and configures both the TSDB and Bindplane pods accordingly.

### Prerequisites

1. **Install cert-manager** in the cluster. Use the official installation guide:
   - [cert-manager installation](https://cert-manager.io/docs/installation/)
   - Ensure the cert-manager controller and webhook are running (e.g. in the `cert-manager` namespace).

2. **Create an Issuer or ClusterIssuer** that can issue TLS certificates (e.g. a CA Issuer or ClusterIssuer). See [cert-manager Issuers](https://cert-manager.io/docs/configuration/).

### Opt-in and configuration

**TSDB:**
- For **cert-manager**: set `spec.tsdb.tls.certManager` (server) and/or `spec.config.tsdb.tls.certManager` (client) with `name` (required), and optionally `kind` (`Issuer` or `ClusterIssuer`, default `Issuer`) and `group` (default `cert-manager.io`).
- For **user-defined TLS**: set `spec.config.tsdb.tls.secretName` and optionally `certKey`, `keyKey`, `caKey`.
- **Mutually exclusive**: do not set both `secretName` and `certManager` for the same TSDB TLS config.

**NATS:**
- Set `spec.config.nats.tls.certManager` with `name` (required), and optionally `kind` and `group`. No `secretName` option exists for NATS TLS.

### Behavior

- The operator creates cert-manager `Certificate` resources (owner-referenced to the Bindplane custom resource). cert-manager issues the certificates and writes them into Kubernetes Secrets.
- The operator mounts those Secrets into the relevant pods and sets the appropriate environment variables (e.g. `BINDPLANE_PROMETHEUS_ENABLE_TLS`, `BINDPLANE_PROMETHEUS_TLS_CERT`, `BINDPLANE_PROMETHEUS_TLS_KEY`, `BINDPLANE_PROMETHEUS_TLS_CA` for TSDB remote write; `BINDPLANE_NATS_ENABLE_TLS`, `BINDPLANE_NATS_TLS_*` for NATS).
- If you use a user-managed remote TSDB (for example, VictoriaMetrics) via `spec.config.tsdb.remote.enable=true`, configure connectivity under `spec.config.tsdb.remote` and use TLS settings appropriate for that backend.
- Certificate renewal and rotation are handled by cert-manager; the operator does not modify the Secret data after cert-manager writes it.

## Validating Admission Webhook

The operator includes a Kubernetes [validating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
that enforces correctness on `Bindplane` custom resources at admission time — before they are persisted by the API server.

### cert-manager requirement

The webhook server runs on port 9443 inside the operator pod and requires a valid TLS certificate at startup. The default install path (`install.yaml`) uses
[cert-manager](https://cert-manager.io/) to provision it:

- A `Issuer` and a `Certificate` resource are created in the operator namespace.
- cert-manager writes the certificate into the Secret `bindplane-operator-webhook-server-cert`.
- The operator pod mounts that Secret at `/tmp/k8s-webhook-server/serving-certs`.
- cert-manager injects the CA bundle into the `ValidatingWebhookConfiguration` so the Kubernetes API server can verify the operator's TLS certificate.

**cert-manager must be installed before applying `install.yaml`.** See the [cert-manager installation guide](https://cert-manager.io/docs/installation/).

### Deploying without the webhook

If cert-manager is unavailable, use `install-no-webhook.yaml` instead:

```bash
kubectl apply \
  --server-side \
  -f https://github.com/observiq/bindplane-operator/releases/latest/download/install-no-webhook.yaml
```

This install path:

- Does not create a `ValidatingWebhookConfiguration` or webhook `Service`.
- Does not require cert-manager.
- Starts the operator with `--enable-validating-webhook=false` (port 9443 is not opened).
- Skips admission-time validation; invalid `Bindplane` specs are not rejected at create/update time.

## Summary

| Secret / TLS | User-configurable? | Env vars (where applicable) | Where configured | Documentation |
|--------------|--------------------|-----------------------------|------------------|----------------|
| License | Yes | `BINDPLANE_LICENSE` | `spec.config.license` or `licenseSecretRef` | [Configuration – License](configuration.md#license) |
| Auth (system, LDAP, OIDC) | Yes | Various (`BINDPLANE_USERNAME`, `BINDPLANE_PASSWORD`, LDAP, OIDC) | `spec.config.auth` and refs | [Configuration – Authentication](configuration.md#authentication) |
| LDAP TLS | Yes | `BINDPLANE_LDAP_TLS_*` (paths from mounted Secret) | `spec.config.auth.ldap.tls` | [Configuration – LDAP](configuration.md#ldap-and-active-directory) |
| Network TLS | Yes | `BINDPLANE_TLS_*` (paths from mounted Secret) | `spec.config.network.tls` | [Configuration – Network](configuration.md#network) |
| Postgres credentials & TLS | Yes | `BINDPLANE_POSTGRES_*`, `BINDPLANE_POSTGRES_SSL_*` | `spec.config.store.postgres` and `tls` | [Configuration – PostgreSQL](configuration.md#postgresql) |
| Bindplane exposes own metrics (optional basic auth) | Yes | `BINDPLANE_METRICS_PROMETHEUS_USERNAME`, `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` | `spec.config.metrics.prometheus` | [Configuration – Metrics](configuration.md#metrics) |
| TSDB remote write auth (default operator-managed TSDB) | No (operator-generated) | `BINDPLANE_PROMETHEUS_AUTH_USERNAME`, `BINDPLANE_PROMETHEUS_AUTH_PASSWORD` | — | This document (above) |
| TSDB TLS / mTLS (cert-manager or user secret) | Yes (opt-in) | `BINDPLANE_PROMETHEUS_ENABLE_TLS`, `BINDPLANE_PROMETHEUS_TLS_*` (when enabled) | `spec.tsdb.tls`, `spec.config.tsdb.tls` | This document (cert-manager TLS); [Configuration – TSDB](configuration.md#tsdb) |
| NATS TLS / mTLS (cert-manager only) | Yes (opt-in) | `BINDPLANE_NATS_ENABLE_TLS`, `BINDPLANE_NATS_TLS_*` | `spec.config.nats.tls.certManager` | This document (cert-manager TLS) |
| Redis TLS | Yes | `BINDPLANE_ADVANCED_CACHE_REDIS_TLS_*` | `spec.config.advanced.cache.redis.tls` | [Configuration – Advanced](configuration.md#advanced) |
| Validating admission webhook TLS (operator install) | No (operator infrastructure) | — | `config/default` (cert-manager required); use `config/overlays/no-webhook` / `install-no-webhook.yaml` to disable | This document |
