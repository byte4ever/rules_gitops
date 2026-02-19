# rules_gitops Modern Port — Design Document

Date: 2026-02-19

## Overview

Port Adobe's rules_gitops to modern Bazel as `github.com/byte4ever/rules_gitops`.
Target: Bazel 8+ with bzlmod (MODULE.bazel, no WORKSPACE), Go 1.24+, rules_oci
replacing rules_docker.

## Project Structure

Flat module layout (single MODULE.bazel at repo root):

```
MODULE.bazel / BUILD.bazel / .bazelrc / go.mod
gitops/          Public API (defs.bzl, extensions.bzl) + PR creation CLI + git backends
skylib/          Starlark rules (k8s.bzl, push.bzl, kustomize/, stamp, toolchain)
resolver/        Go binary: image ref resolution (OCI-aware)
stamper/         Go binary: stamp variable substitution
templating/      Go binary: fast template engine
examples/        Separate module with local override
inspiration/     Original Adobe code (reference only, not built)
```

## bzlmod Migration

- `WORKSPACE` replaced by `MODULE.bazel` with `bazel_dep()` for rules_go,
  rules_oci, gazelle, skylib.
- `http_archive` calls for external deps become `bazel_dep` declarations.
- Kustomize binary setup moves to a module extension in
  `gitops/extensions.bzl` (replaces the `repositories.bzl` pattern).
- `rules_docker` references replaced by `rules_oci` equivalents throughout.

## rules_oci Migration

Provider `K8sPushInfo` stays but its implementation switches from `docker_push`
to `oci_push`.

| Original (rules_docker)    | New (rules_oci)                  |
|----------------------------|----------------------------------|
| `container_image`          | `oci_image`                      |
| `container_push`           | `oci_push` + `oci_image_index`   |
| `docker_pull` (WORKSPACE)  | `oci.pull` (module extension)    |
| `ImageInfo` provider       | `OciImageInfo` provider          |

`push.bzl` wraps `oci_push` instead of `docker_push`. The resolver binary
needs updated logic to extract digests from `oci_push` outputs (OCI layout
directories with `index.json` instead of Docker tarball metadata).

## Kustomize Module Extension

`gitops/extensions.bzl` defines a module extension that:

1. Detects host platform (linux/darwin, amd64/arm64).
2. Downloads kustomize v5.x binary (upgraded from v4.5.3).
3. Registers it as a toolchain for Starlark rule access via `ctx.toolchains`.

Platform support: Linux (amd64, arm64), macOS (amd64, arm64) — adds arm64
macOS which the original lacked.

Consumer usage:

```starlark
bazel_dep(name = "rules_gitops", version = "...")
gitops = use_extension("@rules_gitops//gitops:extensions.bzl", "gitops")
use_repo(gitops, "kustomize_bin")
```

## Go Binaries

Built with rules_go via gazelle-managed BUILD files. Standard Go modules
(no vendor), resolved via gazelle `go_deps` extension:

```starlark
go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")
```

### resolver

Resolves image references in YAML manifests to registry URLs with digests.
Updated to understand rules_oci output format (OCI layout directories).

### stamper

Performs `{{VARIABLE}}` substitutions using Bazel volatile/stable status files.
No major changes from original.

### templating

Fast template engine. The vendored `fasttemplate` library replaced by a proper
`go.mod` dependency (or rewritten if unmaintained).

### create_gitops_prs CLI

Queries Bazel for `.gitops` targets, groups by deployment branch, runs targets,
creates PRs. Supports GitHub, GitLab, and Bitbucket.

Updated client libraries:

- `google/go-github` v32 upgraded to latest (v60+)
- `xanzy/go-gitlab` upgraded to latest
- Bitbucket client updated accordingly

## Design Patterns

| Component                          | Pattern                   | Rationale                                                    |
|------------------------------------|---------------------------|--------------------------------------------------------------|
| Git backends (GH, GL, BB)          | Strategy                  | Swap git platform without changing PR creation logic         |
| `create_gitops_prs` configuration  | Config struct             | CLI has many params, exceeds 4-arg limit                     |
| Manifest pipeline (resolve/stamp/template) | Chain of Responsibility | Each step transforms manifest YAML and passes it along |
| Kustomize binary selection         | Factory                   | Hides platform-specific binary selection behind interface    |

All pattern usages annotated with `// Pattern: <Name> — <reason>` comments.

## Go Conventions

Code follows the `golang` and `go-design-patterns` skills:

- Declaration order: `type` -> `const` -> `var` -> `func`
- File naming: `lowercase_with_underscores.go`
- Max 4 function args, max 3 return values, max 5 interface methods
- 80 char line length enforced by `golines`
- 12% minimum comment density
- Receiver names max 2 chars, consistent across methods
- Enums implement `fmt.Stringer` via `go:generate stringer -linecomment`
- Errors: `errCtx` pattern, `%w` wrapping, `Err` prefix sentinels, `Error`
  suffix types
- Tests: `package foo_test`, `t.Parallel()`, `testify` assertions, `goleak`
- Blocked packages: `goccy/go-json` not `encoding/json`,
  `goccy/go-yaml` not `gopkg.in/yaml.v3`, `log/slog` not logrus,
  `clock` for time abstraction
- Interfaces defined where consumed, not where implemented

Post-change workflow:

```bash
go generate ./...
golangci-lint run --fix
betteralign -apply ./...
golines --shorten-comments --chain-split-dots --max-len=80 \
  --base-formatter=gofumpt -w ./...
golangci-lint run
bazel test //...
```

## Testing Strategy

### Unit tests (Go)

- Standard `go_test` targets for all Go packages.
- Coverage threshold: 80% minimum line coverage per package, enforced via
  `bazel coverage` with `--instrumentation_filter` and CI parse step.
- Fuzz testing: Go 1.24 native `FuzzXxx` targets for resolver, stamper, and
  templating. Run with 30s time budget per target in CI.

### Starlark rule tests

- Bazel skylib `analysistest` and `unittest` for rules in `skylib/`.
- Test `k8s_deploy` target generation, kustomize output for various
  configurations, image substitution, and stamp variable handling.

### Integration / E2E tests

- Kind cluster-based tests using `k8s_test_setup` for ephemeral namespaces.
- `e2e_test.sh` orchestrates the full pipeline.

## CI Pipeline (GitHub Actions)

Triggers: push to `main` and `feature/*`, PRs to `main`.
Bazel version: 8.x only (via bazelisk), no matrix.
Caching: `actions/cache` for `~/.cache/bazel`.

### Jobs

1. **lint-go** — `golangci-lint run`, `betteralign` check, `golines` dry-run.
2. **test-go** — `bazel coverage //...` with 80% threshold, fuzz targets with
   60s timeout, `goleak` for goroutine leak detection.
3. **test-starlark** — `bazel test //skylib/...`.
4. **buildifier** — `bazel run //:buildifier-check`.
5. **e2e** (depends on test-go + test-starlark) — create kind cluster, build
   and test examples, run `e2e-test.sh`.
