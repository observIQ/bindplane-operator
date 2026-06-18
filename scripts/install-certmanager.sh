#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/context-guard.sh
source "${SCRIPT_DIR}/lib/context-guard.sh"
context_guard::require_minikube || exit 1

set -x

NAMESPACE="cert-manager"

echo "Installing cert-manager CRDs..."
kubectl apply \
	--server-side \
	-f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.crds.yaml

echo "Waiting for cert-manager CRDs to be established..."
kubectl wait \
	--for=condition=Established \
	--timeout=120s \
	crd/certificates.cert-manager.io \
	crd/clusterissuers.cert-manager.io \
	crd/issuers.cert-manager.io

echo "Installing cert-manager controllers..."
kubectl apply \
	-f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

echo "Waiting for cert-manager deployments to be ready..."
kubectl wait \
	--namespace "${NAMESPACE}" \
	--for=condition=Available \
	--timeout=300s \
	deployment/cert-manager \
	deployment/cert-manager-cainjector \
	deployment/cert-manager-webhook

echo "cert-manager is installed and ready."
