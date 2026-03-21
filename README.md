# Bindplane Operator

A Kubernetes operator for managing [Bindplane](https://bindplane.com/) server deployments in Kubernetes clusters.

## About Bindplane

Bindplane is a telemetry pipeline built on [OpenTelemetry](https://opentelemetry.io/) that enables teams to collect,
refine, and export metrics, logs, and traces from any source to any destination. The Bindplane Operator automates the
deployment and management of Bindplane server in Kubernetes.

## Getting Started

See **[Getting Started](docs/getting-started.md)** for instructions on deploying Postgres and Bindplane.

## Documentation

- **[Getting Started](docs/getting-started.md)** — Deploy Postgres and Bindplane
- **[Architecture](docs/architecture.md)** — Operator design and components
- **[Configuration](docs/configuration/configuration.md)** — Bindplane configuration (license, auth, store, etc.)
- **[API Reference (CRD)](docs/configuration/api.md)** — Full list of Bindplane custom resource options
- **[Deployment](docs/deployment.md)** — Deployment sizing guidance
- **[Minikube Development](docs/development/minikube.md)** — Running the operator on Minikube

## Learn More

- **[Bindplane](https://bindplane.com/)**: Learn more about Bindplane and its capabilities
- **[Bindplane Documentation](https://docs.bindplane.com/)**: Documentation for using Bindplane
- **[Bindplane OTEL Collector](https://github.com/observIQ/bindplane-otel-collector)**: The Bindplane Distro for OpenTelemetry Collector (BDOT Collector)
- **[OpenTelemetry](https://opentelemetry.io/)**: Learn about the OpenTelemetry project that powers Bindplane
