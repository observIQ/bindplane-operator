#!/usr/bin/env bash

set -ex

NAMESPACE="postgres"
OPERATOR_NAMESPACE="cnpg-system"

echo "Installing CloudNativePG operator..."

# Create operator namespace if it doesn't exist
if ! kubectl get namespace "${OPERATOR_NAMESPACE}" >/dev/null 2>&1; then
	echo "Creating operator namespace ${OPERATOR_NAMESPACE}..."
	kubectl create namespace "${OPERATOR_NAMESPACE}"
else
	echo "Operator namespace ${OPERATOR_NAMESPACE} already exists"
fi

# Create postgres namespace if it doesn't exist
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
	echo "Creating namespace ${NAMESPACE}..."
	kubectl create namespace "${NAMESPACE}"
else
	echo "Namespace ${NAMESPACE} already exists"
fi

# Install CloudNativePG operator
echo "Installing CloudNativePG operator..."
kubectl apply --server-side -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.27/releases/cnpg-1.27.1.yaml

# Wait for CloudNativePG operator to be ready
echo "Waiting for CloudNativePG operator to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/cnpg-controller-manager -n "${OPERATOR_NAMESPACE}" || {
	echo "Warning: CloudNativePG operator deployment may not be ready yet"
}

# Sometimes we need to keep waiting
sleep 10

# Deploy a simple test postgres cluster
echo "Deploying test PostgreSQL cluster..."
cat <<EOF | kubectl apply -n "${NAMESPACE}" -f -
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: test-postgres
spec:
  instances: 1
  imageName: ghcr.io/cloudnative-pg/postgresql:17
  storage:
    size: 1Gi
  bootstrap:
    initdb:
      database: testdb
      owner: testuser
      secret:
        name: test-postgres-superuser
EOF

# Create a secret for the superuser password (CloudNativePG will use this)
echo "Creating superuser secret..."
kubectl create secret generic test-postgres-superuser \
	--from-literal=username=testuser \
	--from-literal=password=testpass \
	-n "${NAMESPACE}" \
	--dry-run=client -o yaml | kubectl apply -f -

echo "Waiting for PostgreSQL cluster to be ready..."
kubectl wait --for=condition=Ready --timeout=600s cluster/test-postgres -n "${NAMESPACE}" || {
	echo "Warning: PostgreSQL cluster may not be ready yet. Check status with:"
	echo "  kubectl get cluster -n ${NAMESPACE}"
	echo "  kubectl get pods -n ${NAMESPACE}"
}

# Wait for the service to be created
CLUSTER_NAME="test-postgres"
SERVICE_NAME="${CLUSTER_NAME}-rw"
echo "Waiting for PostgreSQL service to be ready..."
for i in {1..30}; do
	if kubectl get service "${SERVICE_NAME}" -n "${NAMESPACE}" >/dev/null 2>&1; then
		break
	fi
	sleep 2
done

# Extract connection details
POSTGRES_HOST="${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
POSTGRES_PORT="5432"
POSTGRES_DATABASE="testdb"
POSTGRES_USERNAME="testuser"

# Get password from secret
POSTGRES_PASSWORD=$(kubectl get secret test-postgres-superuser -n "${NAMESPACE}" -o jsonpath='{.data.password}' 2>/dev/null | base64 -d 2>/dev/null || echo "testpass")

echo ""
echo "=========================================="
echo "CloudNativePG operator and test cluster have been deployed!"
echo "=========================================="
echo ""
echo "PostgreSQL Connection Details:"
echo "  Host: ${POSTGRES_HOST}"
echo "  Port: ${POSTGRES_PORT}"
echo "  Database: ${POSTGRES_DATABASE}"
echo "  Username: ${POSTGRES_USERNAME}"
echo "  Password: ${POSTGRES_PASSWORD}"
echo ""
echo "=========================================="
echo "Add this to your Bindplane CR spec:"
echo "=========================================="
echo ""
cat <<EOF
spec:
  bindplane:
    config:
      store:
        type: postgres
        postgres:
          host: ${POSTGRES_HOST}
          port: "${POSTGRES_PORT}"
          database: ${POSTGRES_DATABASE}
          username: ${POSTGRES_USERNAME}
          password: ${POSTGRES_PASSWORD}
EOF
echo ""
echo "=========================================="
echo ""
echo "Note: If your Bindplane CR is in the same namespace (${NAMESPACE}),"
echo "      you can use the shorter hostname: ${SERVICE_NAME}"
echo ""
echo "To check status:"
echo "  kubectl get cluster -n ${NAMESPACE}"
echo "  kubectl get pods -n ${NAMESPACE}"
echo "  kubectl get svc -n ${NAMESPACE}"
echo ""
echo "To connect to the database:"
echo "  kubectl exec -it -n ${NAMESPACE} ${CLUSTER_NAME}-1 -- psql -U ${POSTGRES_USERNAME} -d ${POSTGRES_DATABASE}"

