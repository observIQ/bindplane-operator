# Bindplane Deployment Sizing

This document describes the default deployment sizing and how to configure resources for different collector scales.

## Default Deployment (0-10,000 Collectors)

The default deployment configuration is suitable for managing 0 to 10,000 collectors. The default resource allocations are:

### Bindplane Node
- **Replicas**: 3
- **CPU Request**: 2000m
- **Memory Request**: 2048Mi
- **Memory Limit**: 2048Mi

### Bindplane Jobs
- **Replicas**: 1
- **CPU Request**: 1000m
- **Memory Request**: 1024Mi
- **Memory Limit**: 1024Mi

### Bindplane Jobs Migrate
- **Replicas**: 1
- **CPU Request**: 100m
- **Memory Request**: 2048Mi
- **Memory Limit**: 2048Mi

### Bindplane NATS
- **Replicas**: 1
- **CPU Request**: 250m
- **CPU Limit**: 500m
- **Memory Request**: 500Mi
- **Memory Limit**: 500Mi

**Note**: NATS cluster routes are automatically configured based on the replica count. With 1 replica, only the first hostname is included. With 3 replicas, all three hostnames are included.

### Bindplane Transform Agent
- **Replicas**: 2
- **CPU Request**: 250m
- **Memory Request**: 1024Mi
- **Memory Limit**: 1024Mi

### Bindplane TSDB (Prometheus by default)
- **Replicas**: 1 (scales vertically)
- **CPU Request**: 250m
- **Memory Request**: 500Mi
- **Memory Limit**: 500Mi

## Small Deployment (0-1,000 Collectors)

For smaller deployments managing up to 1,000 collectors, you can reduce resource allocations and replica counts using the `replicas` and `podTemplate` fields in the CRD spec. See the example configuration in `config/samples/bindplane_v1alpha1_bindplane_small.yaml` for a complete example.

### Recommended Small Deployment Configuration

- **Bindplane Node**: 2 replicas, 500m CPU, 1024Mi memory
- **Bindplane Jobs**: 1 replica, 100m CPU, 1024Mi memory
- **Bindplane Jobs Migrate**: 1 replica, 100m CPU, 1024Mi memory
- **Bindplane NATS**: 1 replica, 100m CPU, 256Mi memory
- **Bindplane Transform Agent**: 1 replica, 100m CPU, 512Mi memory
- **Bindplane TSDB (Prometheus default)**: 1 replica, 100m CPU, 256Mi memory

**Note**: 
- The `bindplane.podTemplate` applies to Bindplane Node, Jobs, and Jobs Migrate (all use container name "server")
- When configuring NATS with 1 replica, the cluster routes will automatically include only the first hostname. With 3 replicas, all three hostnames are included.
- Resources can be overridden via `podTemplate` by specifying the container name and resources in the `containers` array.
