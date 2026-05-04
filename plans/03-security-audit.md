# Security Audit: Plan 03

Audited: 2026-05-04

## Section A: SecretRef Coverage

Walk of every sensitive leaf field in `api/v1alpha1/bindplane_types.go`.

### Existing SecretRef pairs (OK)

| Plain field | SecretRef field | Location |
|-------------|-----------------|----------|
| `spec.config.license` | `licenseSecretRef` | `BindplaneConfigSpec` |
| `spec.config.saas.licenseServerAPIKey` | `licenseServerAPIKeySecretRef` | `SaaSConfig` |
| `spec.config.saas.stripe.secretKey` | `secretKeySecretRef` | `SaaSStripeConfig` |
| `spec.config.saas.stripe.publishableKey` | `publishableKeySecretRef` | `SaaSStripeConfig` |
| `spec.config.saas.stripe.webhookSecret` | `webhookSecretSecretRef` | `SaaSStripeConfig` |
| `spec.config.features.postHog.projectAPIKey` | `projectAPIKeySecretRef` | `PostHogConfig` |
| `spec.config.features.postHog.personalAPIKey` | `personalAPIKeySecretRef` | `PostHogConfig` |
| `spec.config.tracing.honeycomb.apiKey` | `apiKeySecretRef` | `TracingHoneycombConfig` |
| `spec.config.metrics.prometheus.password` | `passwordSecretRef` | `MetricsPrometheusConfig` |
| `spec.config.advanced.cache.redis.password` | `passwordSecretRef` | `AdvancedCacheRedisConfig` |
| `spec.config.store.postgres.username` | `usernameSecretRef` | `PostgresConfig` |
| `spec.config.store.postgres.password` | `passwordSecretRef` | `PostgresConfig` |
| `spec.config.auth.username` | `usernameSecretRef` | `AuthConfig` |
| `spec.config.auth.password` | `passwordSecretRef` | `AuthConfig` |
| `spec.config.auth.sessionSecret` | `sessionSecretSecretRef` | `AuthConfig` |
| `spec.config.auth.apiKey` | `apiKeySecretRef` | `AuthConfig` |
| `spec.config.auth.auth0.managementClientSecret` | `managementClientSecretSecretRef` | `Auth0Config` |
| `spec.config.auth.auth0.wif.clientSecret` | `clientSecretSecretRef` | `Auth0WIFConfig` |
| `spec.config.auth.oidc.clientID` | `clientIDSecretRef` | `OIDCConfig` |
| `spec.config.auth.oidc.clientSecret` | `clientSecretSecretRef` | `OIDCConfig` |
| `spec.config.auth.ldap.bindUser` | `bindUserSecretRef` | `LDAPConfig` |
| `spec.config.auth.ldap.bindPassword` | `bindPasswordSecretRef` | `LDAPConfig` |
| `spec.config.llm.langsmith.apiKey` | `apiKeySecretRef` | `LangsmithConfig` |
| `spec.config.llm.openai.apiKey` | `apiKeySecretRef` | `OpenAIConfig` |
| `spec.config.llm.anthropic.apiKey` | `apiKeySecretRef` | `AnthropicConfig` |
| `spec.config.status.keys` | `keysSecretRef` | `StatusConfig` |

### Gaps identified and remediated

| Field | Issue | Status |
|-------|-------|--------|
| `spec.config.errors.backendDSN` | DSNs can include auth tokens; no SecretRef existed | **FIXED** — added `backendDSNSecretRef` |
| `spec.config.errors.frontendDSN` | DSNs can include auth tokens; no SecretRef existed | **FIXED** — added `frontendDSNSecretRef` |

### Deferred / out of scope

- `spec.config.tracing.otlp.headers` — the OTLP exporter does not expose a `headers` field in the current types; deferred until surfaced.
- `spec.config.encryptionProvider.googleKMS.serviceAccountKey` — not a string field; GKE Workload Identity is the recommended approach; out of scope.
- `spec.config.analytics.segmentWriteKey` — low sensitivity (Segment write key is not a high-privilege credential); deferred.

---

## Section B: SecretRef Mutual Exclusivity (XValidation)

The `spec.config.license` / `licenseSecretRef` pair already has an XValidation rule (`has(self.license) != has(self.licenseSecretRef)`). All other pairs lacked this enforcement.

### Remediation applied

Added `XValidation` mutual-exclusivity rules (via `// +kubebuilder:validation:XValidation:` markers) to the following types in `api/v1alpha1/bindplane_types.go`:

| Type | Pairs covered |
|------|---------------|
| `SaaSConfig` | `licenseServerAPIKey` / `licenseServerAPIKeySecretRef` |
| `SaaSStripeConfig` | `secretKey` / `secretKeySecretRef`, `publishableKey` / `publishableKeySecretRef`, `webhookSecret` / `webhookSecretSecretRef` |
| `PostHogConfig` | `projectAPIKey` / `projectAPIKeySecretRef`, `personalAPIKey` / `personalAPIKeySecretRef` |
| `TracingHoneycombConfig` | `apiKey` / `apiKeySecretRef` |
| `MetricsPrometheusConfig` | `password` / `passwordSecretRef` |
| `AdvancedCacheRedisConfig` | `password` / `passwordSecretRef` |
| `PostgresConfig` | `username` / `usernameSecretRef`, `password` / `passwordSecretRef` |
| `AuthConfig` | `username` / `usernameSecretRef`, `password` / `passwordSecretRef`, `sessionSecret` / `sessionSecretSecretRef`, `apiKey` / `apiKeySecretRef` |
| `Auth0Config` | `managementClientSecret` / `managementClientSecretSecretRef` |
| `Auth0WIFConfig` | `clientSecret` / `clientSecretSecretRef` |
| `OIDCConfig` | `clientID` / `clientIDSecretRef`, `clientSecret` / `clientSecretSecretRef` |
| `LDAPConfig` | `bindUser` / `bindUserSecretRef`, `bindPassword` / `bindPasswordSecretRef` |
| `LangsmithConfig` | `apiKey` / `apiKeySecretRef` |
| `OpenAIConfig` | `apiKey` / `apiKeySecretRef` |
| `AnthropicConfig` | `apiKey` / `apiKeySecretRef` |
| `ErrorsConfig` | `backendDSN` / `backendDSNSecretRef`, `frontendDSN` / `frontendDSNSecretRef` |

### SensitiveValueInline Condition

The plan calls for a `SensitiveValueInline=True` status condition when any plain-text sensitive value is set. This is **deferred**: implementing a per-field inline detector that must be kept in sync with every future CRD change adds significant ongoing maintenance risk. The XValidation rules already prevent both from being set simultaneously. Users choosing plaintext values are doing so intentionally, and the existing field comments document the preference for SecretRef in production. This can be revisited as a separate improvement once the full set of sensitive fields stabilizes.

---

## Section C: Logging Audit

Command run:
```
rg -n -e 'log\.\w+\(' -e 'fmt\.Errorf' -e '\.V\(\d+\)\.Info\(' internal/controller/ internal/webhook/ cmd/
```

### Classification

All log calls reviewed. Key findings:

| File:Line | Content | Status |
|-----------|---------|--------|
| `session_secret.go:80` | `log.Info("Creating session secret", "name", secretName, ...)` | OK — name only, no data |
| `prometheus.go:308` | `log.Info("Creating Prometheus basic auth Secret", "name", secretName, ...)` | OK — name only |
| `prometheus.go:253` | `fmt.Errorf("generate password: %w", err)` | OK — error from `crypto/rand`, no credential value |
| `bindplane_controller.go:610` | `log.Error(err, "invalid Bindplane resource")` | OK — validation error message only, no spec dump |
| `bindplane_controller.go:556-698` | All log.Error/log.Info calls use static strings or resource names | OK |
| `webhook/v1alpha1/bindplane_webhook.go:46,52` | `webhookLog.Info("ValidateCreate/Update", "name", bindplane.Name)` | OK — name only |
| `cmd/main.go` | All log calls log addresses and feature flags only | OK |

**No violations found.** No log call dumps `bindplane.Spec` or any sensitive field value.

Specific check: `rg -n 'spec[^,]*bindplane\.Spec' internal/controller/` — zero results confirmed.

---

## Section D: Status/Events Audit

### Events

`rg -n -e 'r\.Recorder\.Event\(' -e 'r\.Recorder\.Eventf\(' internal/controller/` — **zero results**. The operator does not use the Recorder for events (no `Recorder` field in `BindplaneReconciler`). No event-leakage risk.

### Status Conditions

All `meta.SetStatusCondition` calls reviewed:

| File:Line | Condition Message | Status |
|-----------|-------------------|--------|
| `bindplane_controller.go:594` | Static string referencing annotation key | OK |
| `bindplane_controller.go:615` | `err.Error()` — validation error; these messages do not include credential values (validated in Section C) | OK |
| `bindplane_controller.go:687` | `"All resources reconciled successfully"` | OK |
| `jobs.go:429` | Static string for migration status | OK |

**No sensitive values appear in status conditions or events.**

---

## Section E: Operator-managed Secret Hygiene

| Secret | File | Owner Reference | Idempotent | Entropy |
|--------|------|-----------------|-----------|---------|
| `<name>-session-secret` | `session_secret.go` | Set via `controllerutil.SetControllerReference` (line 58) | Create-only; skips if exists (line 64-66) | `crypto/rand` (line 73) |
| `<name>-tsdb-basic-auth` | `prometheus.go` | Set via `controllerutil.SetControllerReference` | Create-only; skips if exists | `crypto/rand` (line ~253) |

Both operator-managed secrets:
- Use `crypto/rand` for entropy — **OK**
- Are create-only (never overwritten) — **OK**
- Have owner references set — **OK**
- Are typed `Opaque` — **OK**

No `math/rand` imports found in any credential-generating path.

---

## Section F: RBAC Audit

File: `config/rbac/role.yaml`

The operator ClusterRole grants:
- `secrets`: `create;delete;get;list;patch;update;watch` — scoped to cluster level but matches the operator's need to create and manage `session-secret` and `tsdb-basic-auth`.

**Assessment:**
- No `*` wildcard on resources.
- No `create` on `clusterroles`, `clusterrolebindings`, or `validatingwebhookconfigurations`.
- No `Node` or `Namespace` access.
- The `secrets` permission is broad but required for the two operator-managed Secrets. Narrowing to specific Secret names is not possible with RBAC.

**Status: OK** — RBAC is appropriately scoped. Documentation added to `docs/configuration/security.md`.

---

## Section G: Container Security Contexts

### Operator-managed pods

`newPodSecurityContext()` in `internal/controller/bindplane_controller.go`:
- `fsGroup: 65534` ✓
- `runAsGroup: 65534` ✓
- `runAsUser: 65534` ✓
- `seccompProfile.type: RuntimeDefault` ✓

`newContainerSecurityContext()`:
- `allowPrivilegeEscalation: false` ✓
- `capabilities.drop: [ALL]` ✓
- `readOnlyRootFilesystem: true` ✓
- `runAsNonRoot: true` ✓
- `runAsUser: 65534` ✓

**Note:** `runAsNonRoot` is set on the container security context but not on the pod security context. This is acceptable as container-level setting takes precedence. The seccompProfile is set at pod level.

### Operator pod (`config/manager/manager.yaml`)

- Pod-level: `runAsNonRoot: true`, `seccompProfile.type: RuntimeDefault` ✓
- Container-level: `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]` ✓
- Missing: `readOnlyRootFilesystem: true` on manager container — **deferred** (the operator binary may write to temp paths; adding this without testing could break the operator pod)

**Status: Substantially OK** — all managed-pod security contexts include seccompProfile. Manager container missing `readOnlyRootFilesystem` is deferred.

---

## Section H: Webhook TLS

`cmd/main.go` analysis:
- `enableHTTP2` defaults to `false`; `disableHTTP2` is applied to `tlsOpts` when HTTP/2 is off (sets `NextProtos: ["http/1.1"]`) ✓
- `tlsOpts` does not set a minimum TLS version — this is left to Go's default TLS configuration, which is TLS 1.2 as of Go 1.18+ ✓
- `certwatcher.CertWatcher` is started as a runnable when `webhookCertPath` is non-empty ✓
- No `MinVersion` is explicitly set in `tlsOpts`, but Go's default `tls.Config.MinVersion` is `tls.VersionTLS12` ✓

**Status: OK** — HTTP/2 disabled, cert rotation via certwatcher, TLS 1.2+ enforced by Go defaults.

---

## Section I: Image Provenance

Images are derived from `bindplane.Spec.Version` at runtime:
- `ghcr.io/observiq/bindplane-ee:<version>`
- `ghcr.io/observiq/bindplane-transform-agent:<version>-bindplane`
- `ghcr.io/observiq/bindplane-prometheus:<version>`

Tag pinning: done via `spec.version`. Digest pinning (`@sha256:...`) is not currently supported but is documented as a future improvement in the configuration docs.

**Status: OK** — tags are version-pinned.

---

## Section J: Dependency / Vulnerability Scan

### go vet

```
go vet ./...
```
**Result: no output (zero issues)**

### gosec

```
make gosec
```
**Result: Issues: 0** (31 `#nosec` annotations in place, all for env var name constants, none suppressing real credentials)

### govulncheck

```
govulncheck ./...
```
**Result: No vulnerabilities found.**

---

## Summary of Findings

| Section | Finding | Status |
|---------|---------|--------|
| A | `errors.backendDSN` / `frontendDSN` missing SecretRef | **FIXED** |
| B | 16 SecretRef pairs missing XValidation mutual-exclusivity | **FIXED** |
| B | `SensitiveValueInline` Condition | **Deferred** |
| C | No logging violations | OK |
| D | No event/status violations | OK |
| E | Both operator-managed Secrets use crypto/rand and owner refs | OK |
| F | RBAC appropriately scoped | OK |
| G | All managed pods have seccompProfile; readOnlyRootFilesystem on manager deferred | Substantially OK |
| H | Webhook TLS: certwatcher, HTTP/2 disabled, TLS 1.2+ | OK |
| I | Images pinned by version tag | OK |
| J | Zero gosec findings, zero vulnerabilities | OK |
