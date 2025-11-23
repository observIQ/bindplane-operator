# Bindplane Operator Architecture

This document describes the architecture of the Bindplane Operator and the services it manages.

## Overview

The Bindplane Operator manages a distributed Bindplane deployment consisting of multiple services that work together to provide telemetry pipeline management capabilities.

## Services

### Bindplane Node

**Type**: Deployment

The primary service that users and managed collectors connect to. Exposes the Bindplane UI, API, and OpAMP endpoints for managing collectors and configurations.

**Scaling**: Scales horizontally with number of replicas.

### Bindplane Jobs

**Type**: Deployment

Manages periodic background jobs, such as agent cleanup and maintenance tasks.

**Scaling**: Always one pod.

### Bindplane Jobs Migrate

**Type**: Deployment

Manages Postgres setup and migrations for the Bindplane database schema.

**Scaling**: Always one pod.

### Bindplane NATS

**Type**: StatefulSet

Operates the Bindplane distributed event bus, allowing Bindplane nodes to communicate with each other.

**Scaling**: Scales horizontally, 3 pods recommended for all deployment sizes.

### Bindplane Transform Agent

**Type**: Deployment

Powers Bindplane's [Live Preview](https://docs.bindplane.com/feature-guides/live-preview) feature, which provides real-time preview of changes to telemetry configurations.

**Scaling**: Scales horizontally, 2 pods generally fine for medium to large environments.

### Bindplane Prometheus

**Type**: StatefulSet

Stores short-term Bindplane metrics for agent health and agent throughput. Rollouts are stored in Postgres. **Note**: This Prometheus instance is NOT for ingesting metrics from managed collectors. It is exclusively for storing metrics pertaining to collector health and other internal workings of Bindplane. It is not intended for user use.

**Scaling**: Scales vertically, one pod only.

## Architecture Diagram

```
               ┌─────────────────────────┐
               │         Ingress         │
               │ (Users and collectors)  │
               └───────────────┬─────────┘
                               │
        ┌──────────────────────▼──────────────────────┐
        │            Bindplane Node                   │
        │      (UI, API, OpAMP endpoints)             │
        └───┬──────────┬──────────┬──────────┬────────┘
            │          │          │          │
            │          │          │          │
    ┌───────▼───┐  ┌───▼────┐  ┌─▼────────┐  ┌──────▼────────┐
    │   NATS    │  │Postgres│  │Prometheus│  │ Transform     │
    │(Event Bus)│  │        │  │          │  │ Agent         │
    └──────┬────┘  └───┬────┘  └──────────┘  └───────────────┘
           │           │
           │           │
           └───────────┘
              │
    ┌─────────┴─────────┐
    │                   │
┌───▼───────────────────▼───┐
│ Bindplane Jobs and        │
│ Jobs Migrate              │
└───────────────────────────┘
```
