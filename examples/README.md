# Examples

## Prerequisites

- Bazel 8+ with bzlmod
- Docker (for building container images)
- kind + kubectl (for e2e tests with a local cluster)

## Helloworld

The `helloworld/` directory builds a simple HTTP server as an OCI image and
deploys it using three `k8s_deploy` configurations.

### Build Chain

```starlark
go_library → go_binary → pkg_tar → oci_image → k8s_deploy
```

The Go binary is cross-compiled for linux/amd64, packaged into a tar layer,
and assembled into a distroless OCI image.

### Development (mynamespace)

Personal namespace deployment using `{BUILD_USER}` as the namespace name.
Applies directly via kubectl — gitops mode is not used.

```bash
bazel run //helloworld:mynamespace.apply
bazel run //helloworld:mynamespace.show
bazel run //helloworld:mynamespace.delete
```

### Canary (canary)

Gitops deployment to a shared `hwteam` namespace with:
- `-canary` name suffix to distinguish from production
- `prefix_suffix_app_labels = True` to modify app labels in Deployments
- Digest-based image tagging (`image_digest_tag = True`)
- Repository prefix `k8s` for image paths
- Deployment branch: `helloworld-canary`

```bash
bazel run //helloworld:canary.show
bazel run //helloworld:canary.gitops
```

### Production (release)

Gitops deployment to the same `hwteam` namespace:
- Digest-based image tagging
- Repository prefix `k8s`
- Deployment branch: `helloworld-prod`
- Tagged as `release` for Bazel query filtering

```bash
bazel run //helloworld:release.show
bazel run //helloworld:release.gitops
```

## Building

```bash
cd examples
bazel build //...
```

## E2E Testing

The repository includes scripts for end-to-end testing with a kind cluster:

```bash
# Create a kind cluster with a local registry
./create_kind_cluster.sh

# Run the full e2e test suite
./e2e_test.sh
```

The e2e test creates a kind cluster, configures a local registry at
`localhost:15000`, builds and pushes images, applies deployments, and
verifies the results.
