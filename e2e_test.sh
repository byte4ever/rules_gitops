#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o xtrace

bindir=$(cd "$(dirname "$0")" && pwd)
cd "${bindir}"

# Check prerequisites.
bazel version
docker version
which kubectl
go version

cluster_running="$(docker inspect -f '{{.State.Running}}' kind-control-plane 2>/dev/null || true)"
if [ "${cluster_running}" != 'true' ]; then
  ./create_kind_cluster.sh
fi

cleanup() {
    echo "Cleanup..."
}

set +o xtrace
trap "echo FAILED ; cleanup" EXIT
set -o xtrace

./examples/e2e-test.sh

cleanup

trap "echo PASS" EXIT
