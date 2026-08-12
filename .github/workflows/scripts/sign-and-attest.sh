#!/usr/bin/env bash
#
# Sign a release's container image and attach its SBOM attestation.
#
# Called by the Sign workflow (.github/workflows/sign.yml), which the Release
# workflow invokes once GoReleaser has pushed the multi-arch manifest lists, and
# which can also be dispatched on its own to sign a release published earlier.
# For the requested release tag the script resolves the registry-side digest and:
#   1. Signs the manifest list by digest with cosign (keyless / Sigstore OIDC).
#   2. Generates an SPDX JSON SBOM with syft against the digest reference.
#   3. Attaches the SBOM as a signed in-toto attestation with cosign.
#
# Signing by digest (name@sha256:...) is the cosign-recommended model;
# signatures are discoverable from any tag pointing at that digest, so signing
# the release tag's manifest list also covers :latest whenever :latest points at
# that same manifest list. Signing is always scoped to the requested release
# tag: a dispatch that re-signs an older release must never sign whatever
# :latest happens to point at now.
#
# Every registry operation is retried with exponential backoff, and each step
# is skipped when its artifact is already present. GHCR's token service
# intermittently rejects push-scoped token requests with "DENIED: denied" even
# when the credentials are valid and were granted the same scope seconds
# earlier — release run 31604663860 published 0.2.0 unsigned that way, after
# the images and the GitHub release had already gone out. A transient registry
# failure must not cost a release its signature, and re-running this script
# against an already-signed digest must be a cheap no-op so that recovery is
# just "run it again".
#
# Prerequisites (installed by the workflow):
#   - docker buildx  (docker/setup-buildx-action)
#   - cosign         (sigstore/cosign-installer)
#   - syft           (anchore/sbom-action/download-syft)
#
# Environment:
#   RELEASE_TAG          – the release tag to sign (e.g. "0.0.20"), passed by the
#                          Sign workflow. Falls back to GITHUB_REF_NAME, which
#                          GitHub Actions sets to the tag on a tag-triggered run.
#   RETRY_MAX_ATTEMPTS   – attempts per registry operation (default 5).
#   RETRY_INITIAL_DELAY  – seconds before the first retry, doubling on each
#                          subsequent attempt (default 5).

set -euo pipefail

RELEASE_TAG="${RELEASE_TAG:-${GITHUB_REF_NAME:-}}"
if [ -z "${RELEASE_TAG}" ]; then
  echo "ERROR: neither RELEASE_TAG nor GITHUB_REF_NAME is set"
  exit 1
fi

IMAGE="ghcr.io/observiq/bindplane-operator"

RETRY_MAX_ATTEMPTS="${RETRY_MAX_ATTEMPTS:-5}"
RETRY_INITIAL_DELAY="${RETRY_INITIAL_DELAY:-5}"

# retry <command> [args...]
# Runs the command, retrying with exponential backoff until it succeeds or
# RETRY_MAX_ATTEMPTS is exhausted. Diagnostics go to stderr so that callers can
# safely capture the command's stdout via command substitution.
retry() {
  local attempt=1
  local delay="${RETRY_INITIAL_DELAY}"
  until "$@"; do
    if [ "${attempt}" -ge "${RETRY_MAX_ATTEMPTS}" ]; then
      echo "ERROR: '$*' failed after ${RETRY_MAX_ATTEMPTS} attempts" >&2
      return 1
    fi
    echo "  attempt ${attempt}/${RETRY_MAX_ATTEMPTS} of '$*' failed, retrying in ${delay}s" >&2
    sleep "${delay}"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

# inspect_digest <tag>
# Prints the manifest digest for a tag. Fails on anything that is not a
# well-formed digest so that a garbled or partial registry response is retried
# rather than propagated into a signing reference.
inspect_digest() {
  local digest
  if ! digest=$(docker buildx imagetools inspect "${IMAGE}:${1}" --format '{{ .Manifest.Digest }}'); then
    return 1
  fi
  if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]; then
    echo "ERROR: unexpected digest '${digest}' for ${IMAGE}:${1}" >&2
    return 1
  fi
  printf '%s\n' "${digest}"
}

# artifact_exists <tag>
# True when the tag already resolves in the registry. Deliberately not retried:
# a missing artifact is the expected answer for a fresh release, and a false
# negative only costs a redundant signing attempt.
artifact_exists() {
  docker buildx imagetools inspect "${IMAGE}:${1}" >/dev/null 2>&1
}

echo "Resolving registry digest for ${IMAGE}:"
DIGEST=$(retry inspect_digest "${RELEASE_TAG}")
echo "  ${RELEASE_TAG} -> ${DIGEST}"

# Reported for operator visibility only — :latest needs no separate signature
# when it points at the same manifest list, and must not be signed when it does
# not (see the note on tag scoping above). Best effort: a failure to resolve
# :latest is not a reason to fail signing the release tag.
LATEST_DIGEST=$(inspect_digest "latest" 2>/dev/null || true)
if [ "${LATEST_DIGEST}" = "${DIGEST}" ]; then
  echo "  latest -> ${DIGEST} (same manifest list, covered by this signature)"
elif [ -n "${LATEST_DIGEST}" ]; then
  echo "  latest -> ${LATEST_DIGEST} (different manifest list, not signed by this run)"
fi

REF="${IMAGE}@${DIGEST}"
# cosign derives its artifact tags from the digest, replacing the algorithm
# separator: sha256:abc... -> sha256-abc....sig / sha256-abc....att
COSIGN_TAG="${DIGEST/:/-}"

if artifact_exists "${COSIGN_TAG}.sig"; then
  echo "Signature already present for ${REF}, skipping"
else
  echo "Signing ${REF}"
  retry cosign sign --yes "${REF}"
fi

if artifact_exists "${COSIGN_TAG}.att"; then
  echo "SBOM attestation already present for ${REF}, skipping"
else
  echo "Generating SBOM for ${REF}"
  retry syft "${REF}" -o spdx-json=sbom.spdx.json

  echo "Attaching SBOM attestation to ${REF}"
  retry cosign attest --yes --predicate sbom.spdx.json --type spdxjson "${REF}"
fi

rm -f sbom.spdx.json
