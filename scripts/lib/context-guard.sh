#!/usr/bin/env bash
# Sourced helper — do not execute directly.
# Provides kubectl-context safety guards for test and install workflows.

context_guard::current() {
	kubectl config current-context 2>/dev/null || echo '<none>'
}

# context_guard::require_minikube
# Fails if the current kubectl context is not "minikube".
# Honors BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT=1 so the e2e suite can invoke
# Makefile targets (make install, make deploy, etc.) on a Kind cluster after the
# suite has already validated the Kind context via context_guard::require_kind.
context_guard::require_minikube() {
	if [[ "${BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT:-}" == "1" ]]; then
		return 0
	fi
	local actual
	actual="$(context_guard::current)"
	if [[ "${actual}" != "minikube" ]]; then
		echo "ERROR: current kubectl context is '${actual}' but this script requires 'minikube'." >&2
		echo "       Run: kubectl config use-context minikube" >&2
		return 1
	fi
}

# context_guard::require_kind "<kind-cluster-name>"
# Fails if the current kubectl context is not "kind-<kind-cluster-name>".
# Honors BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT=1 to bypass for power users.
context_guard::require_kind() {
	local kind_cluster="${1:?kind cluster name required}"
	local expected="kind-${kind_cluster}"
	if [[ "${BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT:-}" == "1" ]]; then
		return 0
	fi
	local actual
	actual="$(context_guard::current)"
	if [[ "${actual}" != "${expected}" ]]; then
		echo "ERROR: current kubectl context is '${actual}' but e2e suite requires '${expected}'." >&2
		echo "       Run: kubectl config use-context ${expected}" >&2
		echo "       Or set: BINDPLANE_OPERATOR_E2E_ALLOW_ANY_CONTEXT=1" >&2
		return 1
	fi
}
