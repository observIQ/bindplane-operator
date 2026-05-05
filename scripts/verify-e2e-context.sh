#!/usr/bin/env bash
# Verifies that the current kubectl context matches the expected Kind cluster
# for e2e tests. Invoked by Makefile e2e targets before running go test.
#
# Usage: verify-e2e-context.sh [kind-cluster-name]
#   kind-cluster-name defaults to "bindplane-operator-test-e2e"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/context-guard.sh
source "${SCRIPT_DIR}/lib/context-guard.sh"

context_guard::require_kind "${1:-bindplane-operator-test-e2e}"
