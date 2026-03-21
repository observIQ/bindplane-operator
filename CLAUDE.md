# Bindplane Operator - Project Rules

## Project Overview

This is a Kubernetes operator for Bindplane, built using Operator SDK v1.42.0 and Kubebuilder v4. The operator manages Bindplane deployments and their supporting services (Transform Agent, Prometheus, etc.).

## Project Generation

This project was generated using `operator-sdk` CLI:

```bash
# Initial project setup
operator-sdk init --domain bindplane.com --repo github.com/observiq/bindplane-operator

# API and CRD generation
operator-sdk create api --group bindplane --version v1alpha1 --kind Bindplane --resource --controller
```

**IMPORTANT:** The PROJECT file tracks the project configuration. Do NOT manually edit it unless you understand the implications.

## Project Structure

```
bindplane-operator/
├── api/v1alpha1/              # API definitions and CRD types
│   ├── bindplane_types.go     # CRD spec/status types (EDIT THIS for schema changes)
│   ├── groupversion_info.go   # API group/version metadata (k8s.bindplane.com/v1alpha1)
│   └── zz_generated.deepcopy.go  # Auto-generated deep copy methods (DO NOT EDIT)
├── cmd/main.go                # Operator entry point
├── internal/controller/        # Controller implementation
│   ├── bindplane_controller.go  # Main reconcile loop + shared helpers
│   ├── transform_agent.go      # Transform Agent service implementation
│   ├── prometheus.go           # Prometheus service implementation
│   └── *_test.go              # Test files
├── config/                    # Kubernetes manifests
│   ├── crd/bases/             # Generated CRD YAML (DO NOT EDIT directly)
│   ├── rbac/                  # RBAC manifests
│   ├── manager/               # Manager deployment
│   └── samples/               # Example CR instances
└── Makefile                   # Build and deployment targets
```

## API Structure

- **API Group:** `k8s.bindplane.com`
- **API Version:** `v1alpha1`
- **Kind:** `Bindplane`
- **CRD Name:** `bindplanes.k8s.bindplane.com`
- **Plural Resource:** `bindplanes`
- **Singular Resource:** `bindplane`

The API group is defined in `api/v1alpha1/groupversion_info.go`. The CRD schema is defined in `api/v1alpha1/bindplane_types.go` using Go structs with kubebuilder markers.

## Controller Architecture

The controller is organized with **one file per service** for scalability:

- **`bindplane_controller.go`**: Main reconcile loop, shared helper functions, label constants, generic reconcile functions
- **`transform_agent.go`**: Transform Agent service (ServiceAccount, Deployment, Service)
- **`prometheus.go`**: Prometheus service (ServiceAccount, StatefulSet, Service)
- **`bindplane_jobs.go`**: Bindplane Jobs and Jobs Migrate services (ServiceAccount, Deployment)
- **`nats.go`**: NATS service (ServiceAccount, StatefulSet, Services)
- **`node.go`**: Bindplane Node service (ServiceAccount, Deployment, Service)

Each service file contains:
- `reconcile<Service>()` - Orchestrates all resources for that service
- `*<Service>ServiceAccount()` - Generates ServiceAccount
- `*<Service>Deployment()` or `*<Service>StatefulSet()` - Generates workload
- `*<Service>Service()` - Generates Service

## Deployment Descriptions

### Prometheus
- **Type**: StatefulSet
- **Purpose**: Stores short-term Bindplane metrics for agent health and agent throughput. Rollouts are stored in Postgres.
- **Binary**: Operates a vanilla (unmodified) Prometheus binary

### Transform Agent
- **Type**: Deployment
- **Purpose**: Powers Bindplane [Live Preview](https://docs.bindplane.com/feature-guides/live-preview) feature
- **Binary**: Operates the Bindplane transform agent binary

### Bindplane Jobs Migrate
- **Type**: Deployment
- **Purpose**: Manages Postgres setup and migrations
- **Binary**: Operates the Bindplane binary
- **Event Bus**: Does not connect to the event bus

### Bindplane Jobs
- **Type**: Deployment
- **Purpose**: Manages periodic jobs, such as agent cleanup
- **Binary**: Operates the Bindplane binary
- **Event Bus**: Connects to the event bus

### Bindplane NATS
- **Type**: StatefulSet
- **Purpose**: Operates the Bindplane distributed event bus, allowing Bindplane nodes to communicate
- **Binary**: Operates the Bindplane binary with an embedded [NATS](https://nats.io/) server

### Bindplane Node
- **Type**: Deployment
- **Purpose**: Exposes the Bindplane UI, API, and OpAMP endpoints
- **Binary**: Operates the Bindplane binary
- **Event Bus**: Connects to the Bindplane NATS event bus

## When to Use operator-sdk CLI

**Use operator-sdk CLI for:**
- Creating new API versions (e.g., `operator-sdk create api --group bindplane --version v1beta1 --kind Bindplane`)
- Creating new CRD kinds (e.g., `operator-sdk create api --group bindplane --version v1alpha1 --kind NewResource`)
- Initial project scaffolding (already done)

**DO NOT use operator-sdk CLI for:**
- Adding fields to existing CRDs (edit `api/v1alpha1/bindplane_types.go` directly)
- Modifying controller logic (edit files in `internal/controller/` directly)
- Adding new services/resources (create new files in `internal/controller/`)

## Making Changes

### Adding/Modifying CRD Fields

1. Edit `api/v1alpha1/bindplane_types.go`:
   - Add fields to `BindplaneSpec` struct
   - Use kubebuilder markers for validation: `// +kubebuilder:validation:Enum=value1;value2`
   - Mark optional fields: `// +optional`
   - Required fields have no `+optional` marker and no `omitempty` in JSON tag

2. Regenerate manifests:
   ```bash
   make manifests
   ```

3. Regenerate deep copy code:
   ```bash
   make generate
   ```

### Adding a New Service

1. Create a new file in `internal/controller/` (e.g., `new_service.go`)

2. Follow the pattern from existing services:
   ```go
   // Reconcile function
   func (r *BindplaneReconciler) reconcileNewService(ctx context.Context, bindplane *bindplanev1alpha1.Bindplane, log logr.Logger) error {
       // Reconcile ServiceAccount
       sa := r.newServiceServiceAccount(bindplane)
       if err := r.reconcileServiceAccount(ctx, bindplane, sa, log); err != nil {
           return err
       }
       // Reconcile other resources...
       return nil
   }

   // Resource generation functions
   func (r *BindplaneReconciler) newServiceServiceAccount(bindplane *bindplanev1alpha1.Bindplane) *corev1.ServiceAccount { ... }
   func (r *BindplaneReconciler) newServiceDeployment(bindplane *bindplanev1alpha1.Bindplane) *appsv1.Deployment { ... }
   ```

3. Add the reconcile call to `bindplane_controller.go` in the main `Reconcile()` function

4. Add RBAC permissions if needed (kubebuilder markers in `bindplane_controller.go`)

5. Regenerate manifests:
   ```bash
   make manifests
   ```

### Adding RBAC Permissions

Add kubebuilder markers to `bindplane_controller.go`:
```go
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

Then run `make manifests` to regenerate `config/rbac/role.yaml`.

## Container Image Conventions

There are three image families, all tied to the same Bindplane release tag:

| Component | Image | Notes |
|-----------|-------|-------|
| Node, NATS, Jobs, Jobs Migrate | `ghcr.io/observiq/bindplane-ee:<version>` | Main Bindplane binary |
| Transform Agent | `ghcr.io/observiq/bindplane-transform-agent:<version>-bindplane` | Appends `-bindplane` suffix to tag |
| Prometheus (TSDB) | `ghcr.io/observiq/bindplane-prometheus:<version>` | Vanilla Prometheus binary |

### Single source of truth

All default images are derived from `defaultBindplaneVersion` in `internal/controller/bindplane_controller.go`:

```go
defaultBindplaneVersion = "1.98.1"
```

- `bindplaneJobsImage` (jobs.go) = `"ghcr.io/observiq/bindplane-ee:" + defaultBindplaneVersion`
- `natsImage` (nats.go) = `bindplaneJobsImage`
- `nodeImage` (node.go) = `natsImage`
- `transformAgentImage` (transform_agent.go) = `"ghcr.io/observiq/bindplane-transform-agent:" + defaultBindplaneVersion + "-bindplane"`
- `tsdbImage` (prometheus.go) = `"ghcr.io/observiq/bindplane-prometheus:" + defaultBindplaneVersion`

**To update all image defaults at once, change only `defaultBindplaneVersion`.**

## Code Conventions

### Labels

Use constants defined in `bindplane_controller.go`:
- `labelKeyName`, `labelKeyInstance`, `labelKeyComponent`, etc.
- `labelValueName`, `labelValueManagedBy`, etc.

Use helper functions:
- `getLabels(bindplane, component)` - Full label set
- `getSelectorLabels(bindplane, component)` - Selector labels only

### Helper Functions

Shared helpers in `bindplane_controller.go`:
- `int64Ptr(i int64) *int64`
- `boolPtr(b bool) *bool`
- `getLabels()` / `getSelectorLabels()`

### Generic Reconcile Functions

Use these for all resources:
- `reconcileServiceAccount()`
- `reconcileDeployment()`
- `reconcileStatefulSet()`
- `reconcileService()`

These handle:
- Setting controller reference (owner)
- Create if not exists
- Update if exists
- Proper error handling

### Resource Naming

Resources are named: `{bindplane.Name}-{component}`
- Example: `my-bindplane-transform-agent`
- Example: `my-bindplane-prometheus`

## Build and Deploy

### Local Development

1. Generate code: `make generate`
2. Generate manifests: `make manifests`
3. Build: `make build`
4. Run locally: `make run`

### Docker Build

```bash
# Configure Minikube Docker
eval $(minikube docker-env)

# Build image
make docker-build IMG=bindplane-operator:local
```

### Deploy to Cluster

```bash
# Install CRDs
make install

# Deploy operator
make deploy IMG=bindplane-operator:local
```

## Files to NEVER Edit Manually

- `api/v1alpha1/zz_generated.deepcopy.go` - Auto-generated
- `config/crd/bases/*.yaml` - Auto-generated (edit types.go instead)
- `config/rbac/role.yaml` - Auto-generated (use kubebuilder markers)
- `PROJECT` - Tool configuration (only edit if you know what you're doing)

## Testing

- Unit tests: `make test`
- E2E tests: `make test-e2e`
- Run controller locally: `make run`

## Quality Checks (Required After Every Change)

After making any code changes, always run the following in order and resolve all findings before finishing:

1. **Tests** — `make test` (or target changed packages: `go test ./internal/controller/...`)
2. **Linter** — `make lint` (or target changed files: `golangci-lint run ./internal/controller/...`)
3. **Security scanner** — `make gosec` (or target changed files: `gosec ./internal/controller/...`)

When only specific files changed, prefer the targeted form to save time. Resolve every finding; do not skip or suppress issues without a documented reason (`// #nosec GXX -- reason`).

## Important Notes

1. **Always run `make manifests` after changing types** - This regenerates CRDs and RBAC
2. **Always run `make generate` after changing types** - This regenerates deep copy code
3. **Service separation** - Each service gets its own file in `internal/controller/`
4. **Shared code** - Constants, helpers, and generic reconcile functions go in `bindplane_controller.go`
5. **RBAC** - Use kubebuilder markers, don't edit `role.yaml` directly
6. **Labels** - Use constants and helper functions, never hardcode label keys

## Common Tasks

### Add a new field to the CRD
1. Edit `api/v1alpha1/bindplane_types.go`
2. Run `make manifests` and `make generate`
3. Update controller to use the new field

### Add a new supporting service
1. Create new file in `internal/controller/` (e.g., `newservice.go`)
2. Implement reconcile and resource generation functions
3. Add reconcile call to main `Reconcile()` function
4. Add RBAC markers if needed
5. Run `make manifests`

### Change API group or version
1. Edit `api/v1alpha1/groupversion_info.go`
2. Edit `PROJECT` file (group field)
3. Run `make manifests` to regenerate CRDs
4. Update all references in code and config files

## Documentation Sync Rule

When adding or modifying any field under `spec.config` (i.e., any field in `BindplaneConfigSpec` or its nested structs in `api/v1alpha1/bindplane_types.go`), you MUST also update `docs/configuration/configuration.md` to document the new or changed field.

The doc uses the following format for each section:
- A prose description of the field's purpose and behavior
- A markdown table of CRD fields, environment variables, defaults, and whether the field is required
- One or more YAML examples

The table of contents at the top of `configuration.md` must also be updated when adding new sections.

## Security Docs Sync Rule

When adding or modifying any authentication option (e.g. any field under `spec.config.auth`, agent auth, OAuth, OIDC, LDAP) or any TLS configuration (e.g. any field under `spec.config.network.tls`, `spec.config.store.postgres.tls`, `spec.config.nats.tls`, `spec.config.tsdb.tls`, `spec.tsdb.tls`, `spec.config.advanced.cache.redis.tls`, or LDAP TLS), you MUST also update `docs/configuration/security.md` to reflect the change.

The doc covers:
- The user-configurable TLS and secrets table (what the user sets, how the operator uses it)
- The cert-manager section (which interfaces support cert-manager-issued certs)
- The summary table at the bottom (every Secret/TLS area, whether it is user-configurable, env vars, where configured, and a link to docs)

## API Docs Generation Rule

When making any changes to files under `api/` (i.e., any modification to CRD types, group/version info, or other API definitions), you MUST run `make generate-api-docs` after completing the changes to regenerate the API reference documentation.
