# Security: TLS and Secrets

This document describes how the Bindplane operator uses **TLS** and **Secrets** across all components. It covers both user-configurable options (detailed in [Configuration](configuration.md)) and operator-generated secrets.

## Overview

- **User-configurable:** You supply Secret names and key names (or direct values) in `spec.config`. The operator mounts those Secrets where needed and sets environment variables to the mounted file paths or injects values from Secrets. See [Configuration](configuration.md) for all options.
- **Operator-generated:** The operator creates and owns certain Secrets (e.g. Prometheus basic auth). These are not user-configurable; the operator generates credentials and uses them consistently across pods.

## User-configurable TLS and Secrets

The following are configured via the Bindplane custom resource and documented in [Configuration](configuration.md):

| Area | What you configure | How the operator uses it |
|------|--------------------|---------------------------|
| **License** | `spec.config.license` or `licenseSecretRef` | Injects `BINDPLANE_LICENSE` from value or Secret into Node, Jobs, Jobs Migrate, NATS. |
| **Authentication** | System auth (`username`/`password` or Secret refs), LDAP bind user/password and [LDAP TLS](configuration.md#ldap-and-active-directory), OIDC client ID/secret or Secret refs | Sets auth-related env vars; mounts LDAP TLS Secret when `spec.config.auth.ldap.tls` is set. |
| **Network TLS** | [Network TLS](configuration.md#network): `spec.config.network.tls` (secretName, certKey, keyKey, caKey, minVersion, skipVerify) | Mounts the Secret at a fixed path and sets `BINDPLANE_TLS_*` env vars to the mounted file paths. Used when you want server-side or mutual TLS on the Bindplane server (often omitted when using [Ingress or Gateway API](configuration.md#network) to terminate TLS). |
| **PostgreSQL** | Postgres username/password (or Secret refs) and [PostgreSQL TLS](configuration.md#postgresql) (`spec.config.store.postgres.tls`, sslmode) | Injects credentials; mounts Postgres TLS Secret when TLS is configured and sets `BINDPLANE_POSTGRES_SSL_*` env vars. |
| **Metrics (Prometheus)** | Optional basic auth for the HTTP endpoint where Bindplane **exposes** its own metrics via `spec.config.metrics.prometheus.username` and `password` or `passwordSecretRef` | Sets `BINDPLANE_METRICS_PROMETHEUS_USERNAME` / `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` when configured. Distinct from Prometheus remote write auth (below). |

Details, field names, and examples for all of the above are in [Configuration](configuration.md).

## Operator-generated Secrets

### Prometheus basic auth

The operator enables basic authentication for the **Prometheus component** (the Prometheus instance deployed by the operator). Credentials are **not** user-configurable.

- The operator generates a username and password and stores them in a Secret named **`<bindplane-name>-prometheus-basic-auth`**.
- The Secret contains: `username`, `password`, and `web-config` (Prometheus `basic_auth_users` YAML used for the Prometheus web config file).
- The Secret is **created once** when it does not exist; the operator does not update its data on later reconciles, so credentials are not rotated unexpectedly.
- The Prometheus pod is configured with `--web.config.file` pointing at the mounted `web-config` content, so the Prometheus UI and API require basic auth.
- All Bindplane pods (Node, Jobs, Jobs Migrate, NATS) receive the same username and password via **`BINDPLANE_PROMETHEUS_AUTH_USERNAME`** and **`BINDPLANE_PROMETHEUS_AUTH_PASSWORD`** (from that Secret) so they can authenticate when using the remote write client to send agent throughput and health metrics to Prometheus. These are **not** the same as `BINDPLANE_METRICS_PROMETHEUS_USERNAME` / `BINDPLANE_METRICS_PROMETHEUS_PASSWORD`, which configure basic auth for the endpoint where Bindplane **exposes** its own metrics (see [Configuration – Metrics](configuration.md#metrics)).

To retrieve the password for manual access (e.g. to open the Prometheus UI in a browser):

```bash
kubectl get secret <bindplane-name>-prometheus-basic-auth -n <namespace> -o jsonpath='{.data.password}' | base64 -d
```

## Cert Manager and Prometheus mTLS (optional)

You can use [cert-manager](https://cert-manager.io/) to have the operator automatically issue and rotate **mutual TLS** certificates for selected in-cluster interfaces, instead of supplying your own TLS Secrets.

**Scope:** This applies only to TLS you configure under `spec.config.prometheus.tls`. It does **not** change:

- Bindplane’s primary HTTP interface (port 3001)
- Bindplane’s connection to PostgreSQL
- Bindplane’s connection to the Transform Agent

**Current support:** The first supported interface is **Prometheus remote write** (Bindplane → Prometheus). Enabling it turns on mTLS for that path: the operator creates cert-manager `Certificate` resources for a Prometheus server cert and a client cert, mounts the issued certs, and configures Prometheus and Bindplane pods accordingly. The same pattern will be used in future releases for NATS and other internal interfaces.

### Prerequisites

1. **Install cert-manager** in the cluster. Use the official installation guide:
   - [cert-manager installation](https://cert-manager.io/docs/installation/)
   - Ensure the cert-manager controller and webhook are running (e.g. in the `cert-manager` namespace).

2. **Create an Issuer or ClusterIssuer** that can issue TLS certificates (e.g. a CA Issuer or ClusterIssuer). See [cert-manager Issuers](https://cert-manager.io/docs/configuration/).

### Opt-in and configuration

- For **cert-manager**: set `spec.config.prometheus.tls.certManager` with `name` (required), and optionally `kind` (`Issuer` or `ClusterIssuer`, default `Issuer`) and `group` (default `cert-manager.io`).
- For **user-defined TLS**: set `spec.config.prometheus.tls.secretName` and optionally `certKey`, `keyKey`, `caKey`.
- **Mutually exclusive**: do not set both `secretName` and `certManager` for Prometheus TLS.

### Behavior

- The operator creates cert-manager `Certificate` resources (owner-referenced to the Bindplane custom resource). cert-manager issues the certificates and writes them into Kubernetes Secrets.
- The operator mounts those Secrets into the relevant pods and sets the appropriate environment variables (e.g. `BINDPLANE_PROMETHEUS_ENABLE_TLS`, `BINDPLANE_PROMETHEUS_TLS_CERT`, `BINDPLANE_PROMETHEUS_TLS_KEY`, `BINDPLANE_PROMETHEUS_TLS_CA` for Prometheus remote write).
- Certificate renewal and rotation are handled by cert-manager; the operator does not modify the Secret data after cert-manager writes it.

## Summary

| Secret / TLS | User-configurable? | Env vars (where applicable) | Where configured | Documentation |
|--------------|--------------------|-----------------------------|------------------|----------------|
| License | Yes | `BINDPLANE_LICENSE` | `spec.config.license` or `licenseSecretRef` | [Configuration – License](configuration.md#license) |
| Auth (system, LDAP, OIDC) | Yes | Various (`BINDPLANE_USERNAME`, `BINDPLANE_PASSWORD`, LDAP, OIDC) | `spec.config.auth` and refs | [Configuration – Authentication](configuration.md#authentication) |
| LDAP TLS | Yes | `BINDPLANE_LDAP_TLS_*` (paths from mounted Secret) | `spec.config.auth.ldap.tls` | [Configuration – LDAP](configuration.md#ldap-and-active-directory) |
| Network TLS | Yes | `BINDPLANE_TLS_*` (paths from mounted Secret) | `spec.config.network.tls` | [Configuration – Network](configuration.md#network) |
| Postgres credentials & TLS | Yes | `BINDPLANE_POSTGRES_*`, `BINDPLANE_POSTGRES_SSL_*` | `spec.config.store.postgres` and `tls` | [Configuration – PostgreSQL](configuration.md#postgresql) |
| Bindplane exposes own metrics (optional basic auth) | Yes | `BINDPLANE_METRICS_PROMETHEUS_USERNAME`, `BINDPLANE_METRICS_PROMETHEUS_PASSWORD` | `spec.config.metrics.prometheus` | [Configuration – Metrics](configuration.md#metrics) |
| Prometheus remote write auth (operator Prometheus) | No (operator-generated) | `BINDPLANE_PROMETHEUS_AUTH_USERNAME`, `BINDPLANE_PROMETHEUS_AUTH_PASSWORD` | — | This document (above) |
| Internal mTLS (cert-manager, e.g. Prometheus remote write) | Yes (opt-in) | `BINDPLANE_PROMETHEUS_ENABLE_TLS`, `BINDPLANE_PROMETHEUS_TLS_*` (when enabled) | `spec.config.prometheus.tls` | This document (Cert Manager and Prometheus mTLS); [Configuration – Prometheus TLS](configuration.md#prometheus-tls) |
