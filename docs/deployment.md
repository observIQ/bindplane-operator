# Bindplane Deployment Sizing

This document describes the default deployment sizing and how to configure resources for different collector scales.

## Table of Contents

- [Default Deployment (0-10,000 Collectors)](#default-deployment-0-10000-collectors)
  - [Minimal Configuration](#minimal-configuration)
  - [Explicit Default Configuration](#explicit-default-configuration)
- [Small Deployment (0-1,000 Collectors)](#small-deployment-0-1000-collectors)
- [Large Deployment (100,000 Collectors)](#large-deployment-100000-collectors)

## Default Deployment (0-10,000 Collectors)

The default deployment configuration is suitable for managing 0 to 10,000 collectors.

### Minimal Configuration

The following is the minimum required configuration. The operator applies default resource allocations and replica counts automatically.

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane
spec:
  config:
    license: "your-license-key-here"
    store:
      type: postgres
      postgres:
        host: postgres.example.com
        port: "5432"
        database: bindplane
        username: bindplane
        password: bindplane
```

### Explicit Default Configuration

The following example shows all default resource allocations explicitly. These values are applied automatically by the operator — you only need to specify them in your CR if you want to override them.

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane
spec:
  config:
    license: "your-license-key-here"
    auth:
      type: system
      username: admin
      password: password
    network:
      host: 0.0.0.0
      port: "3001"
      remoteURL: http://bindplane:3001
    store:
      type: postgres
      postgres:
        host: postgres.example.com
        port: "5432"
        database: bindplane
        username: bindplane
        password: bindplane
  bindplane:
    replicas: 3
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 2000m
                memory: 2048Mi
              limits:
                memory: 2048Mi
  bindplaneJobs:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 1000m
                memory: 1024Mi
              limits:
                memory: 1024Mi
  bindplaneJobsMigrate:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 100m
                memory: 2048Mi
              limits:
                memory: 2048Mi
  nats:
    replicas: 2
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 250m
                memory: 500Mi
              limits:
                cpu: 500m
                memory: 500Mi
  transformAgent:
    replicas: 2
    podTemplate:
      spec:
        containers:
          - name: transform-agent
            resources:
              requests:
                cpu: 250m
                memory: 512Mi
              limits:
                memory: 512Mi
  tsdb:
    podTemplate:
      spec:
        containers:
          - name: tsdb
            resources:
              requests:
                cpu: 1000m
                memory: 2048Mi
              limits:
                memory: 2048Mi
```

## Small Deployment (0-1,000 Collectors)

For smaller deployments managing up to 1,000 collectors, reduce resource allocations and replica counts using the `replicas` and `podTemplate` fields:

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane
spec:
  config:
    license: "your-license-key-here"
    auth:
      type: system
      username: admin
      password: password
    network:
      host: 0.0.0.0
      port: "3001"
      remoteURL: http://bindplane:3001
    store:
      type: postgres
      postgres:
        host: postgres.example.com
        port: "5432"
        database: bindplane
        username: bindplane
        password: bindplane
  bindplane:
    replicas: 2
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 500m
                memory: 1024Mi
              limits:
                memory: 1024Mi
  bindplaneJobs:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 100m
                memory: 1024Mi
              limits:
                memory: 1024Mi
  bindplaneJobsMigrate:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 100m
                memory: 1024Mi
              limits:
                memory: 1024Mi
  nats:
    replicas: 1
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 100m
                memory: 256Mi
              limits:
                cpu: 500m
                memory: 256Mi
  transformAgent:
    replicas: 1
    podTemplate:
      spec:
        containers:
          - name: transform-agent
            resources:
              requests:
                cpu: 100m
                memory: 512Mi
              limits:
                memory: 512Mi
  tsdb:
    podTemplate:
      spec:
        containers:
          - name: tsdb
            resources:
              requests:
                cpu: 100m
                memory: 256Mi
              limits:
                memory: 256Mi
```

## Large Deployment (100,000 Collectors)

For large deployments managing up to 100,000 collectors, increase replica counts and resource allocations:

```yaml
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane
spec:
  config:
    license: "your-license-key-here"
    auth:
      type: system
      username: admin
      password: password
    network:
      host: 0.0.0.0
      port: "3001"
      remoteURL: http://bindplane:3001
    store:
      type: postgres
      postgres:
        host: postgres.example.com
        port: "5432"
        database: bindplane
        username: bindplane
        password: bindplane
  bindplane:
    replicas: 5
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 2000m
                memory: 2048Mi
              limits:
                memory: 2048Mi
  bindplaneJobs:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 1000m
                memory: 1024Mi
              limits:
                memory: 1024Mi
  bindplaneJobsMigrate:
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 100m
                memory: 2048Mi
              limits:
                memory: 2048Mi
  nats:
    replicas: 3
    podTemplate:
      spec:
        containers:
          - name: server
            resources:
              requests:
                cpu: 250m
                memory: 500Mi
              limits:
                cpu: 500m
                memory: 500Mi
  transformAgent:
    replicas: 2
    podTemplate:
      spec:
        containers:
          - name: transform-agent
            resources:
              requests:
                cpu: 250m
                memory: 512Mi
              limits:
                memory: 512Mi
  tsdb:
    podTemplate:
      spec:
        containers:
          - name: tsdb
            resources:
              requests:
                cpu: 2000m
                memory: 4096Mi
              limits:
                memory: 4096Mi
```

> **Note**: With 3 NATS replicas, all three cluster route hostnames are automatically included.
