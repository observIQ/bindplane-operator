# Monitoring the Bindplane Operator

The Bindplane Operator exposes Prometheus metrics from the controller-runtime framework. These metrics provide visibility into reconciliation activity, workqueue health, Kubernetes API client behavior, and leader election status.

## Table of Contents

- [Metrics Endpoint](#metrics-endpoint)
- [Prometheus Operator (ServiceMonitor)](#prometheus-operator-servicemonitor)
- [Annotation-Based Scraping](#annotation-based-scraping)
- [Metrics Reference](#metrics-reference)

## Metrics Endpoint

| Property | Value |
|---|---|
| Port | `8443` |
| Path | `/metrics` |
| Scheme | `https` |

The metrics server uses HTTPS by default (`--metrics-secure=true`). When no external certificate is configured, controller-runtime automatically generates a self-signed certificate at startup. This means HTTPS is always active regardless of whether the validating webhook or cert-manager is installed — the metrics TLS certificate is independent of both.

For production environments, it is recommended to use cert-manager to issue a proper certificate for the metrics server. See `config/prometheus/servicemonitor_tls_patch.yaml` for the cert-manager TLS patch.

## Prometheus Operator (ServiceMonitor)

A `ServiceMonitor` resource is included at `config/prometheus/monitor.yaml` for clusters running the [Prometheus Operator](https://prometheus-operator.dev/). It is disabled by default.

### Prerequisites

- Prometheus Operator installed in the cluster (provides the `monitoring.coreos.com/v1` CRD)
- Prometheus configured to discover `ServiceMonitor` resources in the operator namespace

### Enabling the ServiceMonitor

Uncomment the `../prometheus` line in `config/default/kustomization.yaml`:

```yaml
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

Then re-apply the manifests:

```bash
kubectl apply -k config/default
```

### ServiceMonitor Configuration

The default `ServiceMonitor` scrapes over HTTPS with `insecureSkipVerify: true` because the metrics server uses a self-signed certificate by default:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: bindplane-operator-controller-manager-metrics-monitor
  namespace: bindplane-operator-system
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        insecureSkipVerify: true
  selector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: bindplane-operator
```

#### Using cert-manager for TLS Verification

To eliminate `insecureSkipVerify: true`, enable cert-manager and apply the TLS patch at `config/prometheus/servicemonitor_tls_patch.yaml`. This configures the `ServiceMonitor` to use the CA certificate from the `metrics-server-cert` secret issued by cert-manager.

## Annotation-Based Scraping

For clusters that use annotation-based Prometheus scrape discovery (e.g., `prometheus-community/prometheus` Helm chart), the operator pod template includes the standard scrape annotations:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8443"
  prometheus.io/path: "/metrics"
  prometheus.io/scheme: "https"
```

Because the metrics endpoint uses HTTPS with a self-signed certificate by default, the Prometheus scrape job must be configured to skip TLS verification or supply the CA certificate.

Example Prometheus scrape job using pod annotation discovery with `insecure_skip_verify`:

```yaml
scrape_configs:
  - job_name: bindplane-operator
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - bindplane-operator-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: "true"
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scheme]
        action: replace
        target_label: __scheme__
        regex: (.+)
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: (.+):(?:\d+);(\d+)
        replacement: $1:$2
    tls_config:
      insecure_skip_verify: true
```

## Metrics Reference

The operator exposes standard controller-runtime metrics. No custom metrics are registered.

### Reconciliation

| Metric | Type | Description |
|---|---|---|
| `controller_runtime_reconcile_total` | Counter | Total reconcile attempts, labeled by `controller` and `result` (`success`, `error`, `requeue`, `requeue_after`) |
| `controller_runtime_reconcile_errors_total` | Counter | Total reconcile errors, labeled by `controller` |
| `controller_runtime_reconcile_time_seconds` | Histogram | Reconcile duration in seconds, labeled by `controller` |
| `controller_runtime_reconcile_panics_total` | Counter | Total panics during reconciliation, labeled by `controller` |

### Workqueue

| Metric | Type | Description |
|---|---|---|
| `workqueue_adds_total` | Counter | Total items added to the workqueue |
| `workqueue_depth` | Gauge | Current number of items in the workqueue |
| `workqueue_queue_duration_seconds` | Histogram | Time items spend waiting in the workqueue before processing |
| `workqueue_work_duration_seconds` | Histogram | Time spent processing workqueue items |
| `workqueue_retries_total` | Counter | Total retries for workqueue items |
| `workqueue_longest_running_processor_seconds` | Gauge | Duration of the longest currently-running workqueue processor |

### Kubernetes API Client

| Metric | Type | Description |
|---|---|---|
| `rest_client_requests_total` | Counter | Total HTTP requests made to the Kubernetes API server, labeled by `method`, `code`, and `host` |
| `rest_client_request_duration_seconds` | Histogram | HTTP request latency to the Kubernetes API server |

### Leader Election

| Metric | Type | Description |
|---|---|---|
| `leader_election_master_status` | Gauge | `1` if this instance currently holds the leader lease, `0` otherwise |
