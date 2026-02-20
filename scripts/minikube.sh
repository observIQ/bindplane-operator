#!/usr/bin/env bash

set -ex

minikube delete
minikube start
eval $(minikube docker-env)

# Enable ingress-nginx addon
minikube addons enable ingress

# Wait for ingress-nginx to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

make install-postgres-operator
make docker-build IMG=bindplane-operator:local
make install
make deploy

# Wait for bindplane-sample-node service to be created
kubectl wait --for=condition=available --timeout=300s deployment/bindplane-sample-node || true

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
