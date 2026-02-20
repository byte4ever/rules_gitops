# rules_gitops

Modern Bazel rules for Kubernetes GitOps workflows.

## Overview

`rules_gitops` builds Kubernetes manifests with Kustomize, pushes OCI images,
and deploys via kubectl or gitops pull requests. It is a port of
[Adobe's rules_gitops](https://github.com/adobe/rules_gitops), updated for
Bazel 8+ with bzlmod and rules_oci (replacing rules_docker).

Key changes from the upstream Adobe project:

- Bazel 8+ with bzlmod (`MODULE.bazel`, no WORKSPACE file)
- `rules_oci` replaces `rules_docker` for image handling
- Kustomize downloaded via module extension instead of repository rules
- Go tooling rewritten with modern conventions (`log/slog`, `goccy/go-json`,
  `goccy/go-yaml`)
- Git provider support for GitHub, GitLab, and Bitbucket

## Quick Start

Add the module dependency to your `MODULE.bazel`:

```starlark
bazel_dep(name = "rules_gitops", version = "0.1.0")

gitops = use_extension(
    "@rules_gitops//skylib/kustomize:extensions.bzl",
    "gitops",
)
use_repo(gitops, "kustomize_bin")
```

Define a deployment in a `BUILD` file:

```starlark
load("@rules_gitops//gitops:defs.bzl", "k8s_deploy")

k8s_deploy(
    name = "myapp",
    cluster = "my-cluster",
    user = "my-cluster",
    namespace = "default",
    image_registry = "ghcr.io/myorg",
    images = {
        "myapp-image": ":image",
    },
    manifests = [
        "deployment.yaml",
        "service.yaml",
    ],
)
```

Run the generated targets:

```bash
bazel run //path/to:myapp.show     # print rendered manifests
bazel run //path/to:myapp.apply    # apply to cluster via kubectl
bazel run //path/to:myapp.delete   # delete from cluster
bazel run //path/to:myapp.gitops   # write manifests for gitops workflow
```

## Architecture

Manifest generation uses a three-stage pipeline:

```
kustomize build --> templating ({{VAR}} expansion) --> resolver (image substitution)
```

1. **Kustomize** assembles raw Kubernetes YAML from manifests and overlays.
2. **Templating** expands double-brace `{{VAR}}` placeholders with deployment
   variables (namespace, name, etc.).
3. **Resolver** replaces image references like `//label:image` with fully
   qualified `registry/repo@sha256:digest` values.

The **stamper** operates on a separate path, substituting single-brace `{VAR}`
placeholders with values from Bazel workspace status files.

## Documentation

- [Starlark Rules Reference](skylib/README.md) -- rule APIs and providers
- [Architecture Guide](docs/architecture.md) -- design and internals
- [Kustomize Extension](skylib/kustomize/README.md) -- module extension setup
- [Examples](examples/README.md) -- working BUILD examples

## CLI Tools

| Binary | Package | Purpose |
|--------|---------|---------|
| `create_gitops_prs` | [gitops/prer](gitops/prer/) | Orchestrate gitops PR creation across git providers |
| `fast_template_engine` | [templating](templating/) | Expand `{{VAR}}` templates with stamp info and variables |
| `stamper` | [stamper](stamper/) | Substitute `{VAR}` from Bazel workspace status files |
| `resolver` | [resolver](resolver/) | Replace image references in YAML with resolved digests |
| `it_manifest_filter` | [testing/it_manifest_filter](testing/it_manifest_filter/) | Transform manifests for integration testing (e.g. PVC to emptyDir) |
| `it_sidecar` | [testing/it_sidecar](testing/it_sidecar/) | Pod lifecycle management for integration tests |

## Go Packages

| Package | Description |
|---------|-------------|
| [gitops/bazel](gitops/bazel/) | Bazel target label to executable path conversion |
| [gitops/commitmsg](gitops/commitmsg/) | Gitops target list encoding in commit messages |
| [gitops/digester](gitops/digester/) | SHA256 file digest calculation and verification |
| [gitops/exec](gitops/exec/) | Shell command execution helpers |
| [gitops/git](gitops/git/) | Git repository operations and `GitProvider` interface |
| [gitops/git/bitbucket](gitops/git/bitbucket/) | Bitbucket PR creation provider |
| [gitops/git/github](gitops/git/github/) | GitHub PR creation provider |
| [gitops/git/gitlab](gitops/git/gitlab/) | GitLab PR creation provider |
| [gitops/prer](gitops/prer/) | PR creation orchestrator (worker pool, bazel query, image push) |
| [resolver](resolver/) | OCI-aware image reference resolution in K8s manifests |
| [stamper](stamper/) | Workspace status file substitution engine |
| [templating](templating/) | Fast template engine using `valyala/fasttemplate` |
| [testing/it_manifest_filter](testing/it_manifest_filter/) | Manifest transformation for integration tests |
| [testing/it_sidecar](testing/it_sidecar/) | K8s integration test lifecycle (pod wait, port forward, log tail) |
| [testing/it_sidecar/client](testing/it_sidecar/client/) | Go test helper for it_sidecar subprocess orchestration |
| [testing/it_sidecar/stern](testing/it_sidecar/stern/) | Log tailing for Kubernetes pods |

## Development

```bash
# Run all Go tests
go test ./...

# Run tests with race detection and coverage
go test -race -cover ./...

# Lint (requires golangci-lint v2)
golangci-lint run

# Build all Bazel targets
bazel build //...

# Run all Bazel tests
bazel test //...

# Build examples
cd examples && bazel build //...

# Run end-to-end tests (requires kind, docker, kubectl)
./e2e_test.sh
```

## License

Apache 2.0
