#!/usr/bin/env bash
#
# Sign container images and attach SBOM attestations for a release.
#
# Called by the Release workflow (.github/workflows/release.yml) after
# GoReleaser pushes multi-arch manifest lists. For each release tag the
# script resolves the registry-side digest, deduplicates, and then:
#   1. Signs the manifest list by digest with cosign (keyless / Sigstore OIDC).
#   2. Generates an SPDX JSON SBOM with syft against the digest reference.
#   3. Attaches the SBOM as a signed in-toto attestation with cosign.
#
# Signing by digest (name@sha256:...) is the cosign-recommended model;
# signatures are discoverable from any tag pointing at that digest, so
# signing once covers both the version tag and :latest.
#
# Prerequisites (installed by the workflow):
#   - docker buildx  (docker/setup-buildx-action)
#   - cosign         (sigstore/cosign-installer)
#   - syft           (anchore/sbom-action/download-syft)
#
# Environment:
#   GITHUB_REF_NAME  – the git tag pushed (e.g. "0.0.20"), set automatically
#                      by GitHub Actions.

set -euo pipefail

if [ -z "${GITHUB_REF_NAME:-}" ]; then
  echo "ERROR: GITHUB_REF_NAME is not set"
  exit 1
fi

IMAGE="ghcr.io/observiq/bindplane-operator"
TAGS=(
  "${GITHUB_REF_NAME}"
  "latest"
)

declare -A TAG_TO_DIGEST
declare -A SEEN_DIGESTS

echo "Resolving registry digests for ${IMAGE} tags:"
for TAG in "${TAGS[@]}"; do
  DIGEST=$(docker buildx imagetools inspect "${IMAGE}:${TAG}" --format '{{ .Manifest.Digest }}')
  if [ -z "${DIGEST}" ]; then
    echo "ERROR: failed to resolve digest for ${IMAGE}:${TAG}"
    exit 1
  fi
  TAG_TO_DIGEST["${TAG}"]="${DIGEST}"
  echo "  ${TAG} -> ${DIGEST}"
done

for TAG in "${TAGS[@]}"; do
  DIGEST="${TAG_TO_DIGEST[${TAG}]}"
  if [ -n "${SEEN_DIGESTS[${DIGEST}]:-}" ]; then
    echo "Digest ${DIGEST} already signed and attested (covered by tag ${SEEN_DIGESTS[${DIGEST}]}), skipping ${TAG}"
    continue
  fi
  SEEN_DIGESTS["${DIGEST}"]="${TAG}"

  REF="${IMAGE}@${DIGEST}"

  echo "Signing ${REF}"
  cosign sign --yes "${REF}"

  echo "Generating SBOM for ${REF}"
  syft "${REF}" -o spdx-json=sbom.spdx.json

  echo "Attaching SBOM attestation to ${REF}"
  cosign attest --yes --predicate sbom.spdx.json --type spdxjson "${REF}"

  rm -f sbom.spdx.json
done
