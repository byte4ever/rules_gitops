#!/usr/bin/env bash
set -euo pipefail
# OCI push launcher - wraps crane/oci_push for pushing images.
# Accepts --dst=REGISTRY/REPO:TAG and --image-dir=PATH arguments.
echo "oci_push_launcher: pushing image $*"
echo "ERROR: oci_push_launcher is not yet fully implemented." >&2
echo "This is a placeholder for rules_oci crane-based push." >&2
exit 1
