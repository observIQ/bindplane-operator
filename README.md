<a href="https://bindplane.com">
  <p align="center">
    <picture>
      <source media="(prefers-color-scheme: light)" srcset="https://res.cloudinary.com/du4nxa27k/image/upload/v1734001913/bindplane-logo_czndai.svg" width="auto" height="50">
      <source media="(prefers-color-scheme: dark)" srcset="https://res.cloudinary.com/du4nxa27k/image/upload/v1734001913/bindplane-logo-dark_lkmoxd.svg" width="auto" height="50">
      <img alt="Bindplane Logo" src="https://res.cloudinary.com/du4nxa27k/image/upload/v1734001913/bindplane-logo_czndai.svg" width="auto" height="50">
    </picture>
  </p>
</a>

# Bindplane Operator

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A Kubernetes operator for managing [Bindplane](https://bindplane.com/) server deployments in Kubernetes clusters.

> ⚠️ **Beta:** The Bindplane Operator is currently in beta and is **not recommended for production use**. If you are
> interested in using the operator, please reach out to your Bindplane contact or [Bindplane support](https://bindplane.com/contact).

## About

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
- **[Security](docs/configuration/security.md)** — TLS, Secrets, cert-manager, and webhook configuration
- **[Autoscaling](docs/configuration/autoscaling.md)** — Node HPA behavior and recommendations
- **[Deployment](docs/deployment.md)** — Deployment sizing guidance
- **[Monitoring](docs/monitoring.md)** — Operator metrics and Prometheus integration

## Contributing

Please read the [Contributing Guide](docs/CONTRIBUTING.md) before submitting a pull request.

This project follows the [CNCF Code of Conduct](docs/CODE_OF_CONDUCT.md).

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md).

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Learn More

- **[Bindplane](https://bindplane.com/)**: Learn more about Bindplane and its capabilities
- **[Bindplane Documentation](https://docs.bindplane.com/)**: Documentation for using Bindplane
- **[Bindplane OTEL Collector](https://github.com/observIQ/bindplane-otel-collector)**: The Bindplane Distro for OpenTelemetry Collector (BDOT Collector)
- **[OpenTelemetry](https://opentelemetry.io/)**: Learn about the OpenTelemetry project that powers Bindplane
