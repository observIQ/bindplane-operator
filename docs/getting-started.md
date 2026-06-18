# Getting Started

> ⚠️ **Beta:** The Bindplane Operator is currently in beta and is **not recommended for production use**. If you are
> interested in using the operator, please reach out to your Bindplane contact or [Bindplane support](https://bindplane.com/contact).

## Prerequisites

Before following this guide, ensure you have the following:

- A running Kubernetes cluster with `kubectl` configured to access it
- A Bindplane license key
- Sufficient cluster permissions to install CRDs and deploy workloads
- [cert-manager](https://cert-manager.io/) installed in your cluster (required for the default install, which includes the validating admission webhook — see [Validating Admission Webhook](configuration/security.md#validating-admission-webhook) to deploy without it)
- Postgres database reachable from the cluster

## Install Cert Manager

```bash
kubectl apply \
  --server-side \
  -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

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

Once deployed, the `default` namespace will have the following workloads:
- Bindplane Node: The primary Bindplane deployment that handles all user and collector connections
- Bindplane Jobs: Periodic jobs
- Bindplane Jobs Migrate: One-shot database migration Job (completes and exits)
- Bindplane NATS: Bindplane with the embedded NATS server, forming the distributed event bus for Bindplane communication
- Bindplane TSDB: The Bindplane metrics storage backend, typically Prometheus
- Bindplane Transform Agent: The transform agent powers Bindplane's [Live Preview](https://docs.bindplane.com/feature-guides/live-preview)

```
NAME                                         READY   STATUS    RESTARTS   AGE
bindplane-jobs-b4d59c95b-vnn9z               1/1     Running   0          60s
bindplane-jobs-migrate-<hash>                 0/1     Completed 0          60s  # runs to completion; absent after TTL expires
bindplane-nats-0                             1/1     Running   0          88s
bindplane-nats-1                             1/1     Running   0          60s
bindplane-node-6cd57b4977-8k7zr              1/1     Running   0          52s
bindplane-node-6cd57b4977-b4ct9              1/1     Running   0          60s
bindplane-node-6cd57b4977-xzj9p              1/1     Running   0          60s
bindplane-postgres-1                         1/1     Running   0          8m38s
bindplane-transform-agent-79b648557d-nhj98   1/1     Running   0          60s
bindplane-transform-agent-79b648557d-scwhk   1/1     Running   0          60s
bindplane-tsdb-0                             1/1     Running   0          7m29s
```

## Further Reading

- [Configuration Reference](configuration/configuration.md) — all available `spec.config` fields with examples
- [API Reference](configuration/api.md) — full CRD type definitions
- [Architecture Overview](architecture.md) — how Bindplane's components interact
- [Deployment Sizing](deployment.md) — recommended resource allocations for different scales
