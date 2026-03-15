#!/usr/bin/env bash

set -ex

if [ -z "${BINDPLANE_LICENSE:-}" ]; then
  echo "BINDPLANE_LICENSE is not set. Set it in your shell and re-run." >&2
  exit 1
fi

minikube delete
minikube start
eval $(minikube docker-env)

goreleaser release --snapshot --clean
GOARCH=$(go env GOARCH)
docker tag \
  ghcr.io/observiq/bindplane-operator:latest-${GOARCH} \
  bindplane-operator:local

make install
make deploy

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.4/cert-manager.yaml
minikube addons enable ingress
make install-postgres-operator

# Wait for ingress-nginx to be ready after
# deploying nginx and postgres
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

# Create secrets for Bindplane license and system auth (script expects BINDPLANE_LICENSE; auth defaults to admin/password)
BINDPLANE_AUTH_USERNAME="${BINDPLANE_AUTH_USERNAME:-admin}"
BINDPLANE_AUTH_PASSWORD="${BINDPLANE_AUTH_PASSWORD:-password}"
kubectl create namespace default --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic bindplane-license \
  --from-literal=license="${BINDPLANE_LICENSE}" \
  --namespace=default \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic bindplane-system-auth \
  --from-literal=username="${BINDPLANE_AUTH_USERNAME}" \
  --from-literal=password="${BINDPLANE_AUTH_PASSWORD}" \
  --namespace=default \
  --dry-run=client -o yaml | kubectl apply -f -

# Apply Bindplane CR and cert-manager resources (CR references the secrets above)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl apply -f "${SCRIPT_DIR}/../bindplane_v1alpha1_bindplane.yaml"

# Create ingress for bindplane node service
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bindplane-node
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: bindplane-node.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: bindplane-sample-node
            port:
              number: 3001
EOF

set +x

echo ""
echo "Ingress created! To access the bindplane node service:"
echo "1. Add 'bindplane-node.local' to /etc/hosts pointing to $(minikube ip)"
echo "2. Access via: http://bindplane-node.local"
echo ""
echo "Or use port-forward: kubectl port-forward service/bindplane-sample-node 3001:3001"
echo ""
