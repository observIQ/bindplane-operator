#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/context-guard.sh
source "${SCRIPT_DIR}/lib/context-guard.sh"
context_guard::require_minikube || exit 1

set -x

ARGO_ROLLOUTS_VERSION="${ARGO_ROLLOUTS_VERSION:-v1.9.0}"
NAMESPACE="argo-rollouts"
URL="https://github.com/argoproj/argo-rollouts/releases/download/${ARGO_ROLLOUTS_VERSION}/install.yaml"

echo "Installing Argo Rollouts ${ARGO_ROLLOUTS_VERSION}..."
kubectl apply \
	--server-side \
	-f "${URL}"

echo "Waiting for Argo Rollouts controller to be ready..."
kubectl wait \
	--namespace "${NAMESPACE}" \
	--for=condition=Available \
	--timeout=300s \
	deployment/argo-rollouts

echo "Argo Rollouts ${ARGO_ROLLOUTS_VERSION} is installed and ready."
