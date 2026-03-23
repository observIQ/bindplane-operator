#!/usr/bin/env bash
#
# Sign container images and attach SBOM attestations for a release.
#
# Called by the Release workflow (.github/workflows/release.yml) after
# GoReleaser pushes multi-arch images. For each image tag (version and
# "latest") the script:
#   1. Signs the manifest list with cosign (keyless / Sigstore OIDC).
#   2. Generates an SPDX JSON SBOM with syft.
#   3. Attaches the SBOM as a signed in-toto attestation with cosign.
#
# Prerequisites (installed by the workflow):
#   - cosign   (sigstore/cosign-installer)
#   - syft     (anchore/sbom-action/download-syft)
#
# Environment:
#   GITHUB_REF_NAME  – the git tag pushed (e.g. "0.0.19"), set automatically
#                      by GitHub Actions.

set -euo pipefail

if [ -z "${GITHUB_REF_NAME:-}" ]; then
  echo "ERROR: GITHUB_REF_NAME is not set"
  exit 1
fi

TAG="${GITHUB_REF_NAME}"
IMAGES=(
  "ghcr.io/observiq/bindplane-operator:${TAG}"
  "ghcr.io/observiq/bindplane-operator:latest"
)

for IMAGE in "${IMAGES[@]}"; do
  echo "Signing ${IMAGE}"
  cosign sign --yes "${IMAGE}"

  echo "Generating SBOM for ${IMAGE}"
  syft "${IMAGE}" -o spdx-json > sbom.spdx.json

  echo "Attaching SBOM attestation to ${IMAGE}"
  cosign attest --yes --predicate sbom.spdx.json --type spdxjson "${IMAGE}"

  rm -f sbom.spdx.json
done
