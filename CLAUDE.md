# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Modern port of Adobe's `rules_gitops` for Bazel 8+ with bzlmod. Provides Starlark rules for Kubernetes GitOps workflows: building manifests via Kustomize, pushing OCI images, and deploying via kubectl or gitops PRs. Go tooling handles template expansion, stamping, image resolution, and PR creation across GitHub/GitLab/Bitbucket.

Module: `github.com/byte4ever/rules_gitops`

## Build & Test Commands

```bash
# Go
go test ./...                          # all tests
go test ./gitops/prer/...              # single package
go test -race -cover ./...             # race + coverage
go vet ./...                           # vet

# Lint (golangci-lint v2)
golangci-lint run                      # lint all
golangci-lint run ./gitops/git/...     # lint specific package

# Bazel (requires bazel 8+)
bazel build //...                      # build all
bazel test //...                       # test all
cd examples && bazel build //...       # build examples

# E2E (requires kind, docker, kubectl)
./e2e_test.sh
```

## Architecture

### Starlark Rules (skylib/)

`gitops/defs.bzl` is the public API, exporting `k8s_deploy`, `k8s_test_setup`, `external_image`.

- `skylib/k8s.bzl` — Central `k8s_deploy` macro. Generates `.apply`, `.delete`, `.show`, `.gitops` targets per deployment. Orchestrates kustomize build, template expansion, and image resolution.
- `skylib/kustomize/kustomize.bzl` — `KustomizeInfo` provider, `imagePushStatements` rule, kustomize build actions. Depends on `K8sPushInfo` from `push.bzl`.
- `skylib/kustomize/extensions.bzl` — Module extension that downloads kustomize binary (replaces repository_rule for bzlmod).
- `skylib/push.bzl` — `K8sPushInfo` provider and `k8s_container_push` rule for rules_oci image pushing.
- `skylib/stamp.bzl`, `skylib/expand_template.bzl`, `skylib/merge_files.bzl` — Supporting rules.

### Go Packages

**CLI tools** (each has `cmd/main.go` entry point):
- `templating/` — Fast template engine using `valyala/fasttemplate`. Double-brace `{{VAR}}` for template vars, single-brace `{VAR}` for stamp vars.
- `stamper/` — Workspace status file substitution.
- `resolver/` — OCI-aware image reference resolution in K8s manifests. Replaces `//label:image` references with `registry/repo@sha256:digest`.
- `gitops/prer/` — `create_gitops_prs` CLI. Queries bazel for gitops targets, clones repo, runs deployments, pushes images (worker pool), creates PRs via git provider factory.
- `testing/it_manifest_filter/` — Replaces PVCs with emptyDir for integration tests.
- `testing/it_sidecar/` — K8s integration test lifecycle (pod waiting, port forwarding, log tailing).

**Libraries:**
- `gitops/exec/` — Shell command execution helpers (`Ex`, `MustEx`).
- `gitops/commitmsg/` — Gitops target list encoding in commit messages.
- `gitops/digester/` — SHA256 file digest calculation and verification.
- `gitops/bazel/` — Bazel target label to executable path conversion.
- `gitops/git/` — `GitProvider` strategy interface, `Repo` operations (clone, branch, commit, push).
- `gitops/git/github/`, `gitlab/`, `bitbucket/` — Platform-specific PR creation providers.

### Three-Tool Manifest Pipeline

```
kustomize build → templating ({{VAR}} expansion) → resolver (image substitution)
```

Stamper uses single-brace `{VAR}` separately for workspace status values.

## Key Conventions

### Go Standards

- **Blocked packages**: Use `goccy/go-json` not `encoding/json`, `goccy/go-yaml` not `gopkg.in/yaml.v3`, `log/slog` not logrus.
- **Error handling**: `errCtx` const pattern, `%w` wrapping, lowercase error strings, no `log.Fatal` (return errors), no bare panic (only `Must*` functions).
- **Testing**: `package foo_test`, `t.Parallel()`, `testify` assertions, `export_test.go` bridge for unexported symbols.
- **CLI pattern**: `run(ctx) error` function + `main()` wrapper.
- **Config structs** for >4 parameters. `context.Context` as first parameter.
- **80-char line length**, 12% comment density, max 2-char receiver names.

### Starlark

- Labels use `//skylib:` paths (e.g., `//skylib:k8s.bzl`).
- Module extension in `skylib/kustomize/extensions.bzl` replaces old repository_rules.
- `K8sPushInfo` and `KustomizeInfo` are the key providers connecting push and deploy rules.

### Dependencies

- Bazel 8+ with bzlmod (`MODULE.bazel`, no WORKSPACE)
- rules_oci (replaces rules_docker)
- rules_go, gazelle, rules_pkg, bazel_skylib
- Kustomize v5.4.3 via module extension
- Go 1.24+ (`go.mod` specifies 1.25.0)
- golangci-lint v2 (`.golangci.yml`)
