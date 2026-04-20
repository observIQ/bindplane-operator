# Bindplane Operator vs Helm Chart

Both the Bindplane Operator and the Bindplane Helm chart deploy Bindplane on Kubernetes, but they differ significantly in how they manage the deployment lifecycle, configuration, and ongoing operations. This document explains why the Operator is the recommended choice for production deployments. To get started with the Operator, see the [Getting Started guide](./getting-started.md).

## No Extra Tooling Required

The Helm chart requires the Helm CLI to be installed on every workstation and CI/CD pipeline that installs or upgrades Bindplane. This is an additional dependency to install, version-pin, and maintain across your team.

The Operator is installed once with `kubectl apply`. All ongoing management — installing, upgrading, and configuring Bindplane — is done with standard Kubernetes YAML and `kubectl`. No additional tooling is ever required.

## Ordered Upgrades

Helm applies all resource changes at once and has no built-in mechanism to enforce ordering across components during an upgrade.

The Operator can sequence upgrades when ordering matters. For example, the database migration Job runs and completes before the NATS cluster, Jobs service, and Bindplane Node are updated. If the migration Job fails, the Operator surfaces a `MigrationFailed` status on the `Bindplane` resource and leaves the currently running version intact.

## Mutual TLS Between Internal Services

The Helm chart supports one-way TLS for the connection between Bindplane and the internal Prometheus TSDB, using a user-managed Kubernetes secret. Mutual TLS (mTLS) between internal services is not supported.

The Operator integrates with [cert-manager](https://cert-manager.io/) to automatically issue and rotate certificates for **mutual TLS** between:

- Bindplane Node and the NATS event bus
- Bindplane Node and the internal TSDB (Prometheus)
- Bindplane and the Transform Agent

Once cert-manager is configured, the Operator handles certificate lifecycle automatically — no manual secret creation or rotation needed.

## Self-Healing Deployments

Helm applies desired state only when you run `helm install` or `helm upgrade`. If a Deployment, Service, or ServiceAccount is accidentally deleted or modified after the last Helm operation, Helm will not restore it until the next upgrade.

The Operator continuously watches the `Bindplane` resource and immediately reconciles any drift between desired and actual cluster state. If a component is accidentally removed or changed, the Operator restores it without any manual intervention.

## Validation Before Deployment

The Helm chart can only validate configuration at template rendering time, and only for values covered by its JSON Schema. Semantic errors — such as enabling a feature that requires another field to be set — are not caught until runtime.

The Operator ships a validating admission webhook that checks configuration correctness before the resource is accepted by the Kubernetes API server. Cross-field constraints, required combinations, and format requirements are validated immediately, providing clear error messages before anything is deployed.

## Automatic PodDisruptionBudgets

The Helm chart does not manage PodDisruptionBudgets. During node drain or cluster maintenance, Kubernetes may simultaneously evict multiple Bindplane pods, causing an availability gap.

The Operator automatically creates and manages PodDisruptionBudgets for all replicated Bindplane components, ensuring that cluster maintenance operations do not take down more pods than your deployment can safely tolerate.

---

## Helm Chart Support

The Bindplane Operator is the preferred deployment method as of May 2026. The Helm chart remains fully supported with no plans for deprecation — existing Helm-based deployments will continue to be supported by Bindplane.
