# Getting Started

## Prerequisites

Before following this guide, ensure you have the following:

- A running Kubernetes cluster with `kubectl` configured to access it
- A Bindplane license key
- Sufficient cluster permissions to install CRDs and deploy workloads

## Install Postgres

Bindplane requires Postgres for persisting resources such as configurations. Provisioning, operating, and maintaining
Postgres is the responsibility of the user.

For production environments, a managed cloud database service is strongly recommended, such as [Google Cloud SQL](https://cloud.google.com/sql),
[Amazon RDS](https://aws.amazon.com/rds/postgresql/), or [Azure Database for PostgreSQL](https://azure.microsoft.com/en-us/products/postgresql).
These services handle backups, high availability, and maintenance automatically.

The following is a basic example using [Cloud Native Postgres](https://github.com/cloudnative-pg/cloudnative-pg) for non-production or local
cluster deployments only. It uses a superuser account for simplicity. For a least-privileged user setup, see the
[Postgres Configuration guide](https://docs.bindplane.com/deployment/virtual-machine/bindplane/postgresql/postgres-configuration).

**Install Cloudnative PG operator**:

```bash
kubectl apply \
  --server-side \
  -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.27/releases/cnpg-1.27.1.yaml
```

**Create Postgres user secret**:

```bash
kubectl create secret generic bindplane-postgres-superuser \
	--from-literal=username=bindplane \
	--from-literal=password=bindplanepass \
	--dry-run=client -o yaml | kubectl apply -f -
```

**Deploy a basic Postgres server**:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: bindplane-postgres
spec:
  instances: 1
  storage:
    size: 10Gi
  bootstrap:
    initdb:
      database: bindplane
      owner: bindplane
      secret:
        name: bindplane-postgres-superuser
EOF
```

Wait for Postgres to be ready:

```bash
kubectl wait --for=condition=Ready --timeout=600s cluster/bindplane-postgres
```

## Install Bindplane

**Install the operator and CRDs**:

```bash
kubectl apply \
  --server-side \
  -f https://github.com/observiq/bindplane-operator/releases/latest/download/install.yaml
```

**Create license secret**:

```bash
kubectl create secret generic bindplane-license \
  --from-literal=license="$BINDPLANE_LICENSE"
```

**Deploy Bindplane**:

```bash
kubectl apply -f - <<EOF
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane
spec:
  config:
    licenseSecretRef:
      name: bindplane-license
      key: license
    store:
      postgres:
        host: bindplane-postgres-rw
        usernameSecretRef:
          name: bindplane-postgres-superuser
          key: username
        passwordSecretRef:
          name: bindplane-postgres-superuser
          key: password
EOF
```

Once deployed, the `default` namespace will have the following pods:

```

```

## Further Reading

- [Configuration Reference](configuration/configuration.md) — all available `spec.config` fields with examples
- [API Reference](configuration/api.md) — full CRD type definitions
- [Architecture Overview](architecture.md) — how Bindplane's components interact
- [Deployment Sizing](deployment.md) — recommended resource allocations for different scales
