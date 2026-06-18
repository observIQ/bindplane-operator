# Bindplane Node Autoscaling

This document describes horizontal pod autoscaling (HPA) for **Bindplane Node** — the component that exposes the Bindplane UI, API, and OpAMP endpoints.

Autoscaling is **disabled by default**.

## Table of contents

- [Background: Why Node autoscaling requires care](#background-why-node-autoscaling-requires-care)
- [Enabling autoscaling](#enabling-autoscaling)
- [Configuration reference](#configuration-reference)
  - [Scale-down policies](#scale-down-policies)
- [Default behavior](#default-behavior)
- [Override examples](#override-examples)
  - [Enable with all defaults](#enable-with-all-defaults)
  - [Increase the replica ceiling](#increase-the-replica-ceiling)
  - [Full custom configuration](#full-custom-configuration)
- [Rolling update configuration](#rolling-update-configuration)
- [Interaction with other settings](#interaction-with-other-settings)
- [Recommendations](#recommendations)

---

## Background: Why Node autoscaling requires care

Bindplane Node maintains **persistent, stateful WebSocket connections** from every managed agent using the OpAMP protocol. These connections have two important properties:

- **Cheap to maintain** — an idle connected agent consumes negligible resources on the Node pod.
- **Expensive to re-establish** — when an agent reconnects it must re-authenticate and re-send its full configuration. At scale, thousands of
simultaneous reconnects create a large CPU and network spike.

This asymmetry creates a **scale-down churn loop** if autoscaling is misconfigured:

1. Load drops below the CPU threshold → HPA scales down and removes a pod.
2. All agents that were connected to the removed pod reconnect to surviving pods simultaneously.
3. The reconnection storm pushes CPU above the threshold.
4. HPA scales back up.
5. Repeat.

**The solution is to scale down slowly** — one pod at a time, no faster than once every five minutes. The default configuration enforces this. If you override scale-down behavior, understand the tradeoff before shortening the policy window.

---

## Enabling autoscaling

Set `spec.bindplane.autoscaling.enabled: true`. When enabled:

- The operator creates a `HorizontalPodAutoscaler` targeting the Node `Deployment`.
- `spec.bindplane.replicas` is **ignored** — the HPA controls the replica count.
- The operator deletes the HPA and resumes using `spec.bindplane.replicas` if you later set `enabled: false`.

---

## Configuration reference

All fields are optional. Omitted fields use the documented defaults.

| CRD Field | Type | Default | Required | Description |
|---|---|---|---|---|
| `spec.bindplane.autoscaling.enabled` | `bool` | `false` | No | Enables the HPA. When `false`, static replicas are used. |
| `spec.bindplane.autoscaling.minReplicas` | `integer` | `2` | No | Minimum replica count the HPA may scale down to. |
| `spec.bindplane.autoscaling.maxReplicas` | `integer` | `10` | No | Maximum replica count the HPA may scale up to. |
| `spec.bindplane.autoscaling.metrics` | [`[]MetricSpec`](https://pkg.go.dev/k8s.io/api/autoscaling/v2#MetricSpec) | CPU at 50% utilization | No | Metrics used to calculate the desired replica count. When omitted, scales on CPU at 50% target utilization. |
| `spec.bindplane.autoscaling.behavior` | [`HorizontalPodAutoscalerBehavior`](https://pkg.go.dev/k8s.io/api/autoscaling/v2#HorizontalPodAutoscalerBehavior) | Slow scale-down | No | Scaling behavior in both Up and Down directions. When omitted, applies a slow scale-down policy (1 pod per 5 minutes) to prevent agent reconnection storms. |

---

## Default behavior

When `enabled: true` and no other fields are set, the operator creates an HPA equivalent to:

```yaml
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: <bindplane-name>-node
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      selectPolicy: Min
      policies:
        - type: Pods
          value: 1
          periodSeconds: 300
```

This means:
- Scale **up** when average CPU across all Node pods exceeds 50% of each pod's CPU request.
- Scale **down** by at most one pod every five minutes, and only after the load has been below the threshold for five consecutive minutes.

---

## Override examples

### Enable with all defaults

```yaml
spec:
  bindplane:
    autoscaling:
      enabled: true
```

### Increase the replica ceiling

```yaml
spec:
  bindplane:
    autoscaling:
      enabled: true
      maxReplicas: 20
```

All other settings use their defaults (min 2, CPU target 50%, slow scale-down).

### Full custom configuration

```yaml
spec:
  bindplane:
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 15
      metrics:
        - type: Resource
          resource:
            name: cpu
            target:
              type: Utilization
              averageUtilization: 60
      behavior:
        scaleDown:
          stabilizationWindowSeconds: 600
          selectPolicy: Min
          policies:
            - type: Pods
              value: 1
              periodSeconds: 600
```

---

## Rolling update configuration

Two `spec.bindplane` fields control how the Node Deployment rolls out updates. They work together to pace rolling updates safely for a stateful WebSocket workload.

| CRD Field | Type | Default | Description |
|---|---|---|---|
| `spec.bindplane.minReadySeconds` | `integer` | Termination grace period | Minimum seconds a new pod must be continuously ready before it is considered available and the next pod is replaced. |
| `spec.bindplane.strategy` | [`DeploymentStrategy`](https://pkg.go.dev/k8s.io/api/apps/v1#DeploymentStrategy) | RollingUpdate maxSurge=1 / maxUnavailable=0 | Rollout strategy for the Node Deployment. |

### `minReadySeconds`

By default, `minReadySeconds` is set to the same value as the pod's termination grace period (see `spec.config.advanced.server.opampShutdownGracePeriod`). This pacing is intentional: when a Node pod is removed, the agents that were connected to it begin reconnecting. By holding off on replacing the next pod until `minReadySeconds` has elapsed on the new pod, the operator ensures the new pod has been accepting connections for at least as long as the previous pod took to drain. This gives reconnecting agents time to establish new connections across the healthy pool before another pod is taken out of service. Without this delay, a second pod could start terminating while agents from the first pod are still mid-reconnect, amplifying the reconnection storm.

Set `spec.bindplane.minReadySeconds: 0` to disable this pacing entirely (not recommended for production).

### `strategy`

The default strategy is `RollingUpdate` with `maxSurge: 1` and `maxUnavailable: 0`. This means the Deployment always brings up one new pod before removing any old pod, ensuring the desired replica count is never dipped below during an update. Override with any valid [`DeploymentStrategy`](https://pkg.go.dev/k8s.io/api/apps/v1#DeploymentStrategy).

### Example

```yaml
spec:
  bindplane:
    minReadySeconds: 30
    strategy:
      type: RollingUpdate
      rollingUpdate:
        maxSurge: 1
        maxUnavailable: 0
```

---

## Interaction with other settings

### `spec.bindplane.replicas`

When `autoscaling.enabled: true`, `spec.bindplane.replicas` has no effect. The Deployment's replica field is left unset so the HPA has full control. When autoscaling is disabled, the static `replicas` value is used.

### PodDisruptionBudget

The operator creates a `PodDisruptionBudget` with `minAvailable: 1` by default (controlled by `spec.bindplane.disablePodDisruptionBudget`). Keep the PDB enabled when using autoscaling — it prevents the cluster autoscaler and voluntary evictions from removing too many pods at once.

### OpAMP shutdown grace period

When a Node pod is removed (by the HPA or any other mechanism), connected agents have until the pod's termination grace period expires to reconnect. Configure `spec.config.advanced.server.opampShutdownGracePeriod` to give Bindplane Node time to drain connections gracefully before the pod exits. The operator automatically extends the pod's `terminationGracePeriodSeconds` to 125% of this value.

---

## Database load warning

> **WARNING:** Each additional Bindplane Node pod increases the maximum number of agents that can be concurrently connected to the system. More concurrent agents means more concurrent reads and writes to the PostgreSQL database — query volume, connection pool usage, and peak load all scale with the Node replica count. Before raising `maxReplicas`, ensure your database is sized to handle the resulting increase in concurrent agent activity. Monitor database CPU, connection count, and query latency as you scale up, and provision database resources accordingly.

---

## Recommendations

1. **Start with defaults.** The default scale-down policy (one pod per five minutes) is deliberately conservative. Tighten it only after validating that your agent reconnection behavior is well-understood.

2. **Set `opampShutdownGracePeriod`.** A grace period of 30–60 seconds gives agents time to reconnect before a pod disappears entirely.

3. **Keep the PDB enabled.** `spec.bindplane.disablePodDisruptionBudget: false` (the default) ensures voluntary disruptions cannot remove more than one pod at a time regardless of HPA decisions.

4. **Size CPU requests accurately.** The HPA scales on CPU *utilization* relative to the pod's CPU request. If the request is too low, the HPA will scale up prematurely; if too high, it will scale up too late. Tune `spec.bindplane.podTemplate.spec.containers[0].resources.requests.cpu` based on observed steady-state usage.

5. **Test under load before production.** Simulate agent churn in a staging environment to verify that the scale-down policy you chose does not produce a churn loop under your workload.
