#!/usr/bin/env bash

set -e

NAMESPACE="openldap"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPENLDAP_MANIFEST="${SCRIPT_DIR}/../test/helper/openldap/openldap.yaml"

echo "Deploying OpenLDAP..."

kubectl apply -f "${OPENLDAP_MANIFEST}"

echo "Waiting for OpenLDAP deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/openldap -n "${NAMESPACE}" || {
	echo "Warning: OpenLDAP deployment may not be ready yet. Check status with:"
	echo "  kubectl get pods -n ${NAMESPACE}"
	echo "  kubectl get svc -n ${NAMESPACE}"
}

LDAP_SERVER="openldap.${NAMESPACE}.svc.cluster.local"
LDAP_PORT="1636"
LDAP_BASE_DN="dc=stage,dc=net"
LDAP_BIND_USER="cn=admin,dc=stage,dc=net"
LDAP_BIND_PASSWORD="stageadmin"

echo ""
echo "=========================================="
echo "OpenLDAP has been deployed!"
echo "=========================================="
echo ""
echo "LDAP connection details (LDAPS with TLS):"
echo "  Server:  ${LDAP_SERVER}"
echo "  Port:    ${LDAP_PORT}"
echo "  Base DN: ${LDAP_BASE_DN}"
echo "  Bind DN: ${LDAP_BIND_USER}"
echo "  Bind password: ${LDAP_BIND_PASSWORD}"
echo ""
echo "Test users (from LDAP_USERS / LDAP_PASSWORDS): user / password"
echo ""
echo "=========================================="
echo "Add this to your Bindplane CR spec (LDAP auth with TLS insecure verify):"
echo "=========================================="
echo ""
cat <<EOF
spec:
  config:
    auth:
      type: ldap
      ldap:
        protocol: ldaps
        server: ${LDAP_SERVER}
        port: "${LDAP_PORT}"
        baseDN: ${LDAP_BASE_DN}
        bindUser: ${LDAP_BIND_USER}
        bindPassword: ${LDAP_BIND_PASSWORD}
        tlsSkipVerify: true
EOF
echo ""
echo "=========================================="
echo ""
echo "Note: If your Bindplane CR is in the same namespace (${NAMESPACE}),"
echo "      you can use the shorter hostname: openldap"
echo ""
echo "To check status:"
echo "  kubectl get pods -n ${NAMESPACE}"
echo "  kubectl get svc -n ${NAMESPACE}"
echo ""
