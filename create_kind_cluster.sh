#!/usr/bin/env bash
set -o errexit

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

# Install kind.
go install sigs.k8s.io/kind@v0.29.0
kind="$(go env GOPATH)/bin/kind"

# Create registry container unless it already exists.
reg_name='kind-registry'
reg_port='15000'
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker container rm "${reg_name}" 2>/dev/null || true
  docker run \
    -d --restart=always \
    -e "REGISTRY_HTTP_ADDR=0.0.0.0:${reg_port}" \
    -p "${reg_port}:${reg_port}" \
    --name "${reg_name}" \
    registry:3
fi

# Create a cluster with the local registry enabled.
cat <<EOF | "${kind}" create cluster --name "${KIND_CLUSTER_NAME}" --image "kindest/node:v1.30.13" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
EOF

# Connect the registry to the cluster network.
docker network connect "${KIND_CLUSTER_NAME}" "${reg_name}" || true

# Document the local registry.
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
