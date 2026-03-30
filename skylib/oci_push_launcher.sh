#!/usr/bin/env bash
set -euo pipefail
# OCI push launcher -- pushes an OCI image layout to a registry via crane.
#
# Invoked by push-tag.sh.tpl with arguments built by push.bzl:
#   --crane=<path>            crane binary (rules_oci toolchain)
#   -stamp-info-file <path>   stamp info file (repeated)
#   --dst=REG/REPO:TAG        destination reference
#   --image-dir=<path>        OCI image layout directory
#   -skip-unchanged-digest    skip push if digest unchanged (optional)
#   --insecure                allow HTTP registry (optional)

CRANE=""
DST=""
IMAGE_DIR=""
STAMP_FILES=()
SKIP_UNCHANGED=""
INSECURE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --crane=*)
            CRANE="${1#--crane=}"
            shift
            ;;
        -stamp-info-file)
            STAMP_FILES+=("$2")
            shift 2
            ;;
        --dst=*)
            DST="${1#--dst=}"
            shift
            ;;
        --image-dir=*)
            IMAGE_DIR="${1#--image-dir=}"
            shift
            ;;
        -skip-unchanged-digest)
            SKIP_UNCHANGED="1"
            shift
            ;;
        --insecure)
            INSECURE="--insecure"
            shift
            ;;
        *)
            echo "WARN: oci_push_launcher: unknown argument: $1" >&2
            shift
            ;;
    esac
done

if [[ -z "$CRANE" ]]; then
    echo "ERROR: --crane argument is required" >&2
    exit 1
fi

if [[ -z "$DST" ]]; then
    echo "ERROR: --dst argument is required" >&2
    exit 1
fi

if [[ -z "$IMAGE_DIR" ]]; then
    echo "ERROR: --image-dir argument is required" >&2
    exit 1
fi

# Resolve stamp variables in DST (e.g. {BUILD_USER}).
for sf in "${STAMP_FILES[@]+"${STAMP_FILES[@]}"}"; do
    if [[ -f "$sf" ]]; then
        while IFS=' ' read -r key value; do
            DST="${DST//\{$key\}/$value}"
        done < "$sf"
    fi
done

# Parse registry/repo and tag from DST.
REPO="${DST%:*}"
TAG="${DST##*:}"

echo "oci_push_launcher: pushing ${IMAGE_DIR} -> ${REPO}:${TAG}"

# Extract digest from the OCI layout index.json.
DIGEST=$(grep -o '"sha256:[a-f0-9]*"' "${IMAGE_DIR}/index.json" | head -1 | tr -d '"')

if [[ -z "$DIGEST" ]]; then
    echo "ERROR: could not extract digest from ${IMAGE_DIR}/index.json" >&2
    exit 1
fi

# Push by digest, then tag -- same pattern as rules_oci push.sh.tpl.
REFS=$(mktemp)
trap 'rm -f "${REFS}"' EXIT

"${CRANE}" push ${INSECURE} "${IMAGE_DIR}" "${REPO}@${DIGEST}" --image-refs "${REFS}"
"${CRANE}" tag ${INSECURE} $(cat "${REFS}") "${TAG}"

echo "oci_push_launcher: push complete -> ${REPO}:${TAG} (${DIGEST})"
