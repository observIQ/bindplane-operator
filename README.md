# Bindplane Operator

A Kubernetes operator for managing [Bindplane](https://bindplane.com/) server deployments in Kubernetes clusters.

## About

Bindplane is a telemetry pipeline built on [OpenTelemetry](https://opentelemetry.io/) that enables teams to collect, refine, and export metrics, logs, and traces from any source to any destination. The Bindplane Operator automates the deployment and management of Bindplane server in Kubernetes, including:

The operator uses Kubernetes Custom Resource Definitions (CRDs) to provide a declarative API for managing Bindplane deployments, making it easy to configure, deploy, and maintain Bindplane infrastructure in your Kubernetes cluster.

## Documentation

- **[Architecture](docs/architecture.md)** — Operator design and components
- **[Configuration](docs/configuration.md)** — CRD spec and configuration options
- **[Deployment](docs/deployment.md)** — Deployment sizing guidance
- **[Minikube](docs/minikube.md)** — Running the operator on Minikube

## Learn More

- **[Bindplane](https://bindplane.com/)**: Learn more about Bindplane and its capabilities
- **[Bindplane Documentation](https://docs.bindplane.com/)**: Documentation for using Bindplane
- **[OpenTelemetry](https://opentelemetry.io/)**: Learn about the OpenTelemetry project that powers Bindplane
