# Documentation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add comprehensive documentation to the rules_gitops project — root README, per-package doc.go and README.md, Starlark docstrings, architecture doc, and examples guide.

**Architecture:** Distributed documentation. Each layer has its own docs living next to the code. Root README links to sub-docs. Go packages use doc.go for godoc + README.md for human browsing. Starlark rules get inline `doc=` strings plus a skylib/README.md reference.

**Tech Stack:** Markdown, Go doc comments, Starlark docstrings.

---

### Task 1: Root README.md

**Files:**
- Create: `README.md`

**Step 1: Write README.md**

```markdown
# rules_gitops

Modern Bazel rules for Kubernetes GitOps workflows — build manifests with Kustomize, push OCI images, and deploy via kubectl or gitops pull requests.

A port of [Adobe's rules_gitops](https://github.com/adobe/rules_gitops), updated for Bazel 8+, bzlmod, and rules_oci.

## Quick Start

Add to your `MODULE.bazel`:

\```python
bazel_dep(name = "rules_gitops", version = "0.1.0")

gitops = use_extension("@rules_gitops//skylib/kustomize:extensions.bzl", "gitops")
use_repo(gitops, "kustomize_bin")
\```

Use `k8s_deploy` in a BUILD file:

\```python
load("@rules_gitops//gitops:defs.bzl", "k8s_deploy")

k8s_deploy(
    name = "myapp",
    cluster = "my-cluster",
    namespace = "default",
    manifests = ["deployment.yaml", "service.yaml"],
    images = {"myapp-image": ":image"},
    image_registry = "docker.io",
)
\```

Build and inspect the rendered manifests:

\```bash
bazel run //path/to:myapp.show
\```

Apply directly:

\```bash
bazel run //path/to:myapp.apply
\```

Or generate gitops output for a PR-based workflow:

\```bash
bazel run //path/to:myapp.gitops
\```

## Architecture

The manifest pipeline has three stages:

\```
kustomize build ──► templating ({{VAR}} expansion) ──► resolver (image substitution)
\```

1. **Kustomize** assembles raw manifests from YAML files, patches, configmaps, and overlays.
2. **Templating** expands double-brace `{{VAR}}` placeholders with deployment-specific values (cluster, namespace, custom substitutions).
3. **Resolver** replaces `//label:image` references with fully-qualified `registry/repo@sha256:digest` strings.

A separate **stamper** handles single-brace `{VAR}` substitution from Bazel workspace status files.

## Documentation

- [Starlark Rules Reference](skylib/README.md) — all rules, macros, and providers
- [Architecture Guide](docs/architecture.md) — pipeline details, package map, design patterns
- [Examples](examples/README.md) — helloworld walkthrough with three deployment variants

## CLI Tools

| Binary | Package | Purpose |
|--------|---------|---------|
| `create_gitops_prs` | [`gitops/prer`](gitops/prer/README.md) | Orchestrate gitops PR creation across GitHub/GitLab/Bitbucket |
| `fast_template_engine` | [`templating`](templating/README.md) | Expand `{{VAR}}` templates from stamp files and key-value pairs |
| `stamper` | [`stamper`](stamper/README.md) | Substitute `{VAR}` placeholders from workspace status files |
| `resolver` | [`resolver`](resolver/README.md) | Replace image references in YAML with registry digests |
| `it_manifest_filter` | [`testing/it_manifest_filter`](testing/it_manifest_filter/README.md) | Transform manifests for integration testing |
| `it_sidecar` | [`testing/it_sidecar`](testing/it_sidecar/README.md) | Pod lifecycle helper for integration tests |

## Go Packages

| Package | Description |
|---------|-------------|
| [`gitops/bazel`](gitops/bazel/README.md) | Bazel target label to executable path conversion |
| [`gitops/commitmsg`](gitops/commitmsg/README.md) | Gitops target list encoding in commit messages |
| [`gitops/digester`](gitops/digester/README.md) | SHA256 file digest calculation and verification |
| [`gitops/exec`](gitops/exec/README.md) | Shell command execution helpers |
| [`gitops/git`](gitops/git/README.md) | Git repository operations and provider interface |
| [`gitops/git/github`](gitops/git/github/README.md) | GitHub PR creation provider |
| [`gitops/git/gitlab`](gitops/git/gitlab/README.md) | GitLab merge request provider |
| [`gitops/git/bitbucket`](gitops/git/bitbucket/README.md) | Bitbucket Server PR provider |
| [`gitops/prer`](gitops/prer/README.md) | Gitops PR orchestration engine |
| [`resolver`](resolver/README.md) | OCI image reference resolution in YAML |
| [`stamper`](stamper/README.md) | Workspace status variable substitution |
| [`templating`](templating/README.md) | Fast template engine with double-brace syntax |
| [`testing/it_manifest_filter`](testing/it_manifest_filter/README.md) | PVC-to-emptyDir manifest transformation |
| [`testing/it_sidecar`](testing/it_sidecar/README.md) | K8s integration test lifecycle helpers |
| [`testing/it_sidecar/client`](testing/it_sidecar/client/README.md) | Go test helper for it_sidecar orchestration |
| [`testing/it_sidecar/stern`](testing/it_sidecar/stern/README.md) | Pod log tailing (stern port) |

## Development

\```bash
# Go tests
go test ./...
go test -race -cover ./...

# Lint (golangci-lint v2)
golangci-lint run

# Bazel build
bazel build //...
bazel test //...

# E2E tests (requires kind, docker, kubectl)
./e2e_test.sh
\```

## License

Apache License 2.0
```

**Step 2: Verify**

Run: `cat README.md | head -20` to confirm structure.

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add root README with quickstart and project overview"
```

---

### Task 2: Architecture doc

**Files:**
- Create: `docs/architecture.md`

**Step 1: Write docs/architecture.md**

Content must cover:

1. **Manifest Pipeline** — diagram of the 3-stage flow (kustomize → templating → resolver), with explanation of what each stage does, what inputs it takes, and what it outputs. Include the stamper as a separate path for `{VAR}` substitution.

2. **Starlark Rule Graph** — how `k8s_deploy` macro generates multiple targets (`.apply`, `.delete`, `.show`, `.gitops`), how it connects `kustomize` rule → `kubectl`/`gitops` rules, and how `K8sPushInfo` and `KustomizeInfo` providers flow between rules.

3. **Package Dependency Map** — which Go packages depend on which:
   - `prer` → `git`, `exec`, `bazel`, `commitmsg`, `digester`
   - `git` → `exec`
   - `git/github`, `git/gitlab`, `git/bitbucket` → `git` (implement `GitProvider`)
   - `resolver`, `stamper`, `templating` are standalone
   - `sidecar` → `stern`
   - `client` → (imports `sidecar` types)

4. **GitProvider Strategy Pattern** — the `GitProvider` interface, `GitProviderFunc` adapter, and the three implementations. Factory function in `prer/cmd/main.go` selects provider by `--git_server` flag.

5. **Worker Pool in prer** — how `prer.Run` uses `--push_parallelism` workers to push images concurrently, groups targets by deployment train, and creates one PR per branch.

**Step 2: Verify**

Visually scan the document for accuracy.

**Step 3: Commit**

```bash
git add docs/architecture.md
git commit -m "docs: add architecture guide with pipeline and package map"
```

---

### Task 3: All Go doc.go files

**Files:**
- Create: `gitops/bazel/doc.go`
- Create: `gitops/commitmsg/doc.go`
- Create: `gitops/digester/doc.go`
- Create: `gitops/exec/doc.go`
- Create: `gitops/git/doc.go`
- Create: `gitops/git/github/doc.go`
- Create: `gitops/git/gitlab/doc.go`
- Create: `gitops/git/bitbucket/doc.go`
- Create: `gitops/prer/doc.go`
- Create: `resolver/doc.go`
- Create: `stamper/doc.go`
- Create: `templating/doc.go`
- Create: `testing/it_manifest_filter/doc.go`
- Create: `testing/it_sidecar/doc.go`
- Create: `testing/it_sidecar/client/doc.go`
- Create: `testing/it_sidecar/stern/doc.go`

**Step 1: Create doc.go for each package**

Each doc.go contains the package doc comment (migrated from the source file where it currently lives) plus `package <name>`. The existing package comment in the source file should be replaced with just `package <name>` (no duplicate doc comment).

Template:

```go
// Package <name> <existing doc comment text>.
//
// <Optional additional paragraph with usage examples or key concepts.>
package <name>
```

Exact content for each:

`gitops/bazel/doc.go`:
```go
// Package bazel provides utilities for working with Bazel target labels and
// paths, converting target labels like //foo/bar:baz to their executable
// runfile paths.
package bazel
```

`gitops/commitmsg/doc.go`:
```go
// Package commitmsg generates and parses gitops target lists embedded in git
// commit messages. Targets are encoded between marker lines so that the prer
// package can detect which gitops deployments a commit carries.
package commitmsg
```

`gitops/digester/doc.go`:
```go
// Package digester calculates and verifies SHA256 file digests. It stores
// digests in companion .digest files alongside the original, enabling
// skip-if-unchanged optimizations for image pushes.
package digester
```

`gitops/exec/doc.go`:
```go
// Package exec provides shell command execution helpers. Ex returns combined
// output and an error; MustEx panics on failure for use in contexts where
// errors are unrecoverable.
package exec
```

`gitops/git/doc.go`:
```go
// Package git provides git repository operations and a strategy interface for
// creating pull requests across different git hosting platforms.
//
// The GitProvider interface abstracts PR creation. Implementations exist for
// GitHub, GitLab, and Bitbucket Server in sub-packages. GitProviderFunc is a
// convenience adapter that lets plain functions satisfy the interface.
//
// Repo wraps a local git clone with methods for branching, committing, and
// pushing. Clone creates a new Repo from a remote URL with optional mirror
// reference.
package git
```

`gitops/git/github/doc.go`:
```go
// Package github implements a git.GitProvider that creates pull requests on
// GitHub (cloud or enterprise). Configure with a Config containing the
// repository owner, name, and personal access token. Set EnterpriseHost for
// GitHub Enterprise installations.
package github
```

`gitops/git/gitlab/doc.go`:
```go
// Package gitlab implements a git.GitProvider that creates merge requests on
// GitLab. Configure with a Config containing the host URL, project path, and
// personal access token.
package gitlab
```

`gitops/git/bitbucket/doc.go`:
```go
// Package bitbucket implements a git.GitProvider that creates pull requests on
// Bitbucket Server (Stash). Configure with a Config containing the REST API
// endpoint, username, and password or token.
package bitbucket
```

`gitops/prer/doc.go`:
```go
// Package prer orchestrates the creation of gitops pull requests. It queries
// Bazel for gitops targets, groups them by deployment train, clones the git
// repository, runs each target, stamps files, commits changes, pushes images
// using a configurable worker pool, and creates PRs via a git.GitProvider.
//
// The main entry point is Run, which accepts a Config struct with all
// parameters for the workflow.
package prer
```

`resolver/doc.go`:
```go
// Package resolver walks multi-document YAML looking for container image
// references and substitutes them with fully-qualified registry URLs from an
// image map. It handles both single-document and multi-document YAML streams
// separated by "---" markers.
package resolver
```

`stamper/doc.go`:
```go
// Package stamper reads Bazel workspace status files and substitutes
// single-brace {VAR} placeholders in format strings. LoadStamps parses one or
// more status files into a variable map; Stamp combines loading and
// substitution in a single call.
package stamper
```

`templating/doc.go`:
```go
// Package templating provides a fast template engine that substitutes
// variables from stamp info files and explicit key-value pairs. It uses
// valyala/fasttemplate with configurable delimiters (default "{{" and "}}").
//
// The Engine type holds configuration (start/end tags, stamp info files) and
// expands templates via the Expand method, which reads a template file,
// applies variable substitution and import expansion, and writes the result.
package templating
```

`testing/it_manifest_filter/doc.go`:
```go
// Package filter transforms Kubernetes manifests for integration testing by
// replacing persistent storage references with ephemeral volumes and adjusting
// certificate issuers. It drops PersistentVolumeClaim and Ingress objects,
// converts StatefulSet volumeClaimTemplates to emptyDir volumes, and replaces
// letsencrypt-prod issuers with self-signed ones.
package filter
```

`testing/it_sidecar/doc.go`:
```go
// Package sidecar provides Kubernetes integration test lifecycle helpers that
// watch pods, set up port forwarding, tail logs via stern, and handle graceful
// cleanup. It is used by the it_sidecar CLI binary to manage the test
// environment lifecycle.
package sidecar
```

`testing/it_sidecar/client/doc.go`:
```go
// Package client provides a test-infrastructure helper that Go integration
// tests use to orchestrate the it_sidecar process from TestMain. K8STestSetup
// manages the sidecar subprocess lifecycle, waits for readiness, and exposes
// forwarded service ports to test code.
package client
```

`testing/it_sidecar/stern/doc.go`:
```go
// Package stern provides pod log tailing for Kubernetes integration tests.
// It watches pods matching a target pattern, follows container logs in real
// time, and handles container state transitions. Ported from the stern
// project.
package stern
```

**Step 2: Remove duplicate package comments from source files**

For each package, remove the `// Package ...` comment block from the original source file (e.g., `bazeltargets.go`, `commitmsg.go`, etc.) since doc.go now owns the package documentation. Leave just the bare `package <name>` declaration in those files.

**Step 3: Verify**

Run: `go vet ./...` to ensure packages still compile cleanly.

**Step 4: Commit**

```bash
git add -A '*/doc.go' gitops/ resolver/ stamper/ templating/ testing/
git commit -m "docs: add doc.go files for all Go packages

Migrate package-level documentation comments into dedicated doc.go
files and expand them with additional context."
```

---

### Task 4: Go README.md — small utility packages

**Files:**
- Create: `gitops/bazel/README.md`
- Create: `gitops/commitmsg/README.md`
- Create: `gitops/digester/README.md`
- Create: `gitops/exec/README.md`

**Step 1: Write READMEs**

Each follows this template:

```markdown
# <package name>

<One-line description from doc.go.>

## API

<Table of exported functions/types with one-line descriptions.>

## Usage

<Short Go code example showing typical usage.>
```

Content guidance per package:

**`gitops/bazel/README.md`**: Show `TargetToExecutable` converting `//foo/bar:baz` to its runfile path. Explain the Bazel label convention.

**`gitops/commitmsg/README.md`**: Show `Generate(targets)` producing a commit message block with the begin/end markers, and `ExtractTargets(msg)` parsing them back out.

**`gitops/digester/README.md`**: Show the digest lifecycle: `SaveDigest` writes a `.digest` companion file, `VerifyDigest` checks if the file has changed, `GetDigest` reads it back, `CalculateDigest` computes a fresh SHA256.

**`gitops/exec/README.md`**: Show `Ex` running a command and capturing output, and `MustEx` for fire-and-forget commands that panic on failure.

**Step 2: Commit**

```bash
git add gitops/bazel/README.md gitops/commitmsg/README.md gitops/digester/README.md gitops/exec/README.md
git commit -m "docs: add READMEs for utility packages (bazel, commitmsg, digester, exec)"
```

---

### Task 5: Go README.md — core libraries

**Files:**
- Create: `resolver/README.md`
- Create: `stamper/README.md`
- Create: `templating/README.md`

**Step 1: Write READMEs**

**`resolver/README.md`**: Describe the image map format (`imagename=registry/repo@sha256:digest`), show `ResolveImages` reading from stdin-like reader and writing substituted YAML. Mention the CLI binary in `resolver/cmd/`. Show example YAML before/after resolution.

**`stamper/README.md`**: Explain the workspace status file format (key-value lines), show `LoadStamps` parsing files, `Stamp` doing substitution. Mention the CLI binary in `stamper/cmd/`. Distinguish single-brace `{VAR}` (stamper) from double-brace `{{VAR}}` (templating).

**`templating/README.md`**: Explain the Engine configuration (StartTag, EndTag, StampInfoFiles), show `Expand` expanding a template with variables and imports. Mention the CLI binary in `templating/cmd/`. Cover the `--variable NAME=VALUE` and `--imports NAME=filename` patterns.

**Step 2: Commit**

```bash
git add resolver/README.md stamper/README.md templating/README.md
git commit -m "docs: add READMEs for core libraries (resolver, stamper, templating)"
```

---

### Task 6: Go README.md — git package and providers

**Files:**
- Create: `gitops/git/README.md`
- Create: `gitops/git/github/README.md`
- Create: `gitops/git/gitlab/README.md`
- Create: `gitops/git/bitbucket/README.md`

**Step 1: Write READMEs**

**`gitops/git/README.md`**: Document the `GitProvider` interface and its single `CreatePR` method. Document `GitProviderFunc` adapter. Document the `Repo` type with all its methods. Show `Clone` usage. Explain the mirror optimization. List the three provider sub-packages.

**`gitops/git/github/README.md`**: Document `Config` fields (RepoOwner, Repo, AccessToken, EnterpriseHost). Show `NewProvider` + `CreatePR` usage. Note: set `EnterpriseHost` for GHE.

**`gitops/git/gitlab/README.md`**: Document `Config` fields (Host, Repo, AccessToken). Show `NewProvider` + `CreatePR` usage. Note: `Repo` is the full project path like `org/project`.

**`gitops/git/bitbucket/README.md`**: Document `Config` fields (APIEndpoint, User, Password). Show `NewProvider` + `CreatePR` usage. Note: this targets Bitbucket Server (Stash), not Bitbucket Cloud.

**Step 2: Commit**

```bash
git add gitops/git/README.md gitops/git/github/README.md gitops/git/gitlab/README.md gitops/git/bitbucket/README.md
git commit -m "docs: add READMEs for git package and provider sub-packages"
```

---

### Task 7: Go README.md — prer (create_gitops_prs)

**Files:**
- Create: `gitops/prer/README.md`

**Step 1: Write README**

This is the most complex package. The README must cover:

1. **Overview** — what create_gitops_prs does end-to-end.
2. **Config struct** — table of all fields with types, defaults, and descriptions.
3. **CLI flags** — full table of all flags (from the exploration data above), grouped by category: Bazel, Git repository, Branch, Push, Gitops kinds, PR, Provider selection, GitHub-specific, GitLab-specific, Bitbucket-specific.
4. **Workflow** — numbered steps of what `Run` does:
   1. Query Bazel for gitops targets
   2. Group targets by deployment branch
   3. Clone the git repository (with optional mirror)
   4. For each deployment train: switch branch, run targets, stamp, commit
   5. Push images (worker pool with `PushParallelism` goroutines)
   6. Create PRs via the configured GitProvider
5. **Usage example** — a full CLI invocation with GitHub provider.

**Step 2: Commit**

```bash
git add gitops/prer/README.md
git commit -m "docs: add README for prer package with CLI reference and workflow"
```

---

### Task 8: Go README.md — testing packages

**Files:**
- Create: `testing/it_manifest_filter/README.md`
- Create: `testing/it_sidecar/README.md`
- Create: `testing/it_sidecar/client/README.md`
- Create: `testing/it_sidecar/stern/README.md`

**Step 1: Write READMEs**

**`testing/it_manifest_filter/README.md`**: Explain what transformations `ReplacePDWithEmptyDirs` performs (drop PVCs/Ingresses, convert StatefulSet volumes, fix issuers). Show CLI usage with `--infile`/`--outfile`. Explain why this is needed (integration tests don't have persistent storage).

**`testing/it_sidecar/README.md`**: Document the sidecar lifecycle (watch pods → port forward → tail logs → signal READY → wait for shutdown). Document all exported functions. Show CLI flags. Explain interaction with the client package.

**`testing/it_sidecar/client/README.md`**: Show how to use `K8STestSetup` from `TestMain` in a Go integration test. Document `WaitForPods`, `PortForwardServices`, `ReadyCallback` fields. Show `GetServiceLocalPort` usage for connecting to forwarded services.

**`testing/it_sidecar/stern/README.md`**: Brief description of the stern port. Document `Target`, `Tail`, `Run`, `Watch` types/functions. Note the Apache 2.0 license from the original stern project.

**Step 2: Commit**

```bash
git add testing/it_manifest_filter/README.md testing/it_sidecar/README.md testing/it_sidecar/client/README.md testing/it_sidecar/stern/README.md
git commit -m "docs: add READMEs for testing packages (filter, sidecar, client, stern)"
```

---

### Task 9: Starlark docstrings — gitops/defs.bzl

**Files:**
- Modify: `gitops/defs.bzl`

**Step 1: Add module docstring**

Add at the top of `gitops/defs.bzl`:

```python
"""Public API for rules_gitops.

Provides three rules for Kubernetes GitOps workflows:

- `k8s_deploy`: Main deployment macro that generates kustomize, kubectl, show,
  and gitops targets for a set of Kubernetes manifests.
- `k8s_test_setup`: Rule for setting up Kubernetes integration test environments
  with image pushing, manifest application, pod waiting, and port forwarding.
- `external_image`: Rule for declaring externally-hosted container images (not
  built by Bazel) that should be injected into manifests.
"""
```

**Step 2: Commit**

```bash
git add gitops/defs.bzl
git commit -m "docs: add module docstring to gitops/defs.bzl"
```

---

### Task 10: Starlark docstrings — skylib/k8s.bzl

**Files:**
- Modify: `skylib/k8s.bzl`

**Step 1: Add doc= to show rule**

Find the `show = rule(` definition and add a `doc` parameter:

```python
show = rule(
    implementation = _show_impl,
    doc = """Generates an executable that renders and displays the final Kubernetes manifests.

Runs the template engine on kustomize output, expanding {{VAR}} placeholders
with stamp values, and prints each document separated by "---" delimiters.
Use with `bazel run` to inspect the fully rendered manifests.""",
    attrs = { ... },
)
```

**Step 2: Add docstring to _show_impl**

```python
def _show_impl(ctx):
    """Build a shell script that renders and prints kustomize output through the template engine."""
```

**Step 3: Add doc= to kubeconfig repository_rule**

```python
kubeconfig = repository_rule(
    implementation = _kubeconfig_impl,
    doc = """Finds local kubectl and Kubernetes certificates to create a kubeconfig file.

Reads the cluster configuration from the host system and exports kubeconfig,
kubectl binary, and cluster name as files in the external repository.""",
    attrs = { ... },
    local = True,
    environ = [...],
)
```

**Step 4: Add doc= to k8s_test_namespace rule**

```python
k8s_test_namespace = rule(
    implementation = _k8s_test_namespace_impl,
    doc = """Creates a namespace reservation script for integration tests.

Generates an executable shell script from a template that creates or reuses
a Kubernetes namespace for test isolation.""",
    attrs = { ... },
)
```

**Step 5: Add doc= to k8s_test_setup rule**

```python
k8s_test_setup = rule(
    implementation = _k8s_test_setup_impl,
    doc = """Creates a complete integration test setup executable.

Generates a shell script that pushes container images, applies Kubernetes
manifests through the manifest filter (replacing PVCs with emptyDir), waits
for pods and services to become ready, and sets up port forwarding. Used
with the it_sidecar binary to manage the test lifecycle.""",
    attrs = { ... },
)
```

**Step 6: Add docstrings to _kubeconfig_impl, _k8s_test_namespace_impl, _k8s_test_setup_impl**

```python
def _k8s_test_namespace_impl(ctx):
    """Build a namespace reservation script from the template."""

def _k8s_test_setup_impl(ctx):
    """Build the integration test setup script that pushes images, applies manifests, and configures port forwarding."""
```

**Step 7: Verify**

Run: `bazel build //skylib:k8s.bzl` or just check syntax with a quick `python3 -c "compile(open('skylib/k8s.bzl').read(), 'k8s.bzl', 'exec')"` — note: Starlark isn't Python, so just visually verify quotes are balanced.

**Step 8: Commit**

```bash
git add skylib/k8s.bzl
git commit -m "docs: add docstrings to all rules and implementations in k8s.bzl"
```

---

### Task 11: Starlark docstrings — skylib/push.bzl, external_image.bzl, stamp.bzl

**Files:**
- Modify: `skylib/external_image.bzl`
- Modify: `skylib/stamp.bzl`

(push.bzl already has good docs — skip it)

**Step 1: Add doc= to external_image rule**

```python
external_image = rule(
    implementation = _external_image_impl,
    doc = """Declares an externally-hosted container image for injection into manifests.

Provides K8sPushInfo for images not built by Bazel. The image reference and
digest are specified as attributes, and a digest file is written so the
resolver can substitute the reference in manifests.""",
    attrs = { ... },
)
```

**Step 2: Add doc= to stamp_value rule**

```python
stamp_value = rule(
    implementation = _stamp_value_impl,
    doc = """Stamps a string by substituting {VAR} placeholders from workspace status files.

Runs the stamper CLI on the `str` attribute and writes the result to a .txt
output file. Commonly used to resolve {BUILD_USER} at analysis time.""",
    attrs = { ... },
)
```

**Step 3: Add doc= to more_stable_status rule**

```python
more_stable_status = rule(
    implementation = _more_stable_status_impl,
    doc = """Extracts selected variables from stable-status.txt into a reduced status file.

Produces a subset of the Bazel stable status file containing only the
variables listed in the `vars` attribute. This reduces unnecessary rebuilds
when unrelated status variables change.""",
    attrs = { ... },
)
```

**Step 4: Add docstrings to _impl functions**

```python
def _external_image_impl(ctx):
    """Write the image digest to a file and return K8sPushInfo."""

def _stamp_value_impl(ctx):
    """Run the stamper CLI on the str attribute and write the result."""

def _more_stable_status_impl(ctx):
    """Filter stable-status.txt to include only the requested variables."""
```

**Step 5: Commit**

```bash
git add skylib/external_image.bzl skylib/stamp.bzl
git commit -m "docs: add docstrings to external_image, stamp_value, and more_stable_status rules"
```

---

### Task 12: Starlark docstrings — skylib/kustomize/kustomize.bzl and related

**Files:**
- Modify: `skylib/kustomize/kustomize.bzl`
- Modify: `skylib/run_in_workspace.bzl`

**Step 1: Add doc to KustomizeInfo provider**

```python
KustomizeInfo = provider(
    doc = """Carries image push statements from kustomize rules to kubectl/gitops rules.

Contains shell script fragments that push container images. Rules that
consume KustomizeInfo (kubectl, gitops) prepend these push statements
to their generated scripts.""",
    fields = {
        "image_pushes": "Depset of Files containing image push shell script fragments.",
    },
)
```

**Step 2: Add doc= to kustomize rule**

```python
kustomize = rule(
    implementation = _kustomize_impl,
    doc = """Builds Kubernetes manifests using kustomize with template expansion and image resolution.

Generates a kustomization.yaml from the provided manifests, patches, configmaps,
and overlays, runs `kustomize build`, pipes the output through the template engine
for {{VAR}} expansion, and then through the resolver for image reference substitution.
Outputs a single rendered .yaml file.""",
    attrs = { ... },
)
```

**Step 3: Add doc= to kubectl rule**

```python
kubectl = rule(
    implementation = _kubectl_impl,
    doc = """Generates a kubectl apply/delete executable for rendered manifests.

Creates a shell script that optionally pushes images, then pipes kustomize
output through the template engine and applies or deletes it via kubectl.
Use with `bazel run //target.apply` or `bazel run //target.delete`.""",
    attrs = { ... },
)
```

**Step 4: Add doc= to gitops rule**

```python
gitops = rule(
    implementation = _gitops_impl,
    doc = """Generates a gitops output executable that writes rendered manifests to a directory tree.

Creates a shell script that optionally pushes images and writes the final
kustomize-built, template-expanded manifests to a gitops path structure:
$TARGET_DIR/{gitops_path}/{namespace}/{cluster}/{file}. Used by the
create_gitops_prs CLI to generate PR content.""",
    attrs = { ... },
)
```

**Step 5: Add doc= to undocumented kustomize rule attributes**

Add `doc=` strings to: `configmaps_srcs`, `secrets_srcs`, `deps_aliases`, `disable_name_suffix_hash`, `end_tag`, `manifests`, `name_prefix`, `name_suffix`, `namespace`, `patches`, `start_tag`, `substitutions`, `deps`, `configurations`, `common_labels`, `common_annotations`.

Use concise descriptions matching those in `k8s_deploy`'s docstring.

**Step 6: Add docstring to imagePushStatements**

```python
def imagePushStatements(ctx, kustomize_objs, files=[]):
    """Collect image push shell script fragments from kustomize objects.

    Gathers push statements from KustomizeInfo providers on the given
    kustomize objects and any additional files, returning the combined
    statements, file list, and runfile depset.

    Args:
        ctx: Rule context.
        kustomize_objs: List of targets providing KustomizeInfo.
        files: Additional files to include.

    Returns:
        Tuple of (statements string, files list, dep_runfiles depset).
    """
```

**Step 7: Add docstring to workspace_binary macro**

In `skylib/run_in_workspace.bzl`:

```python
def workspace_binary(name, cmd, args=None, visibility=None, data=None, root_file="//:MODULE.bazel"):
    """Creates an executable target that runs a binary from the workspace root.

    Wraps a binary tool so it executes with the working directory set to the
    workspace root (located by finding root_file in runfiles). Useful for
    tools like kustomize that need to run relative to the project root.

    Args:
        name: Target name.
        cmd: Label of the binary to wrap.
        args: Optional arguments to pass to the binary.
        visibility: Bazel visibility.
        data: Additional data dependencies.
        root_file: Label used to locate the workspace root in runfiles.
    """
```

**Step 8: Add docstrings to _impl functions**

```python
def _kustomize_impl(ctx):
    """Generate kustomization.yaml, run kustomize build, expand templates, and resolve images."""

def _kubectl_impl(ctx):
    """Generate a shell script that pushes images and runs kubectl apply/delete."""

def _gitops_impl(ctx):
    """Generate a shell script that pushes images and writes manifests to a gitops directory tree."""
```

**Step 9: Commit**

```bash
git add skylib/kustomize/kustomize.bzl skylib/run_in_workspace.bzl
git commit -m "docs: add docstrings to kustomize rules, providers, and workspace_binary"
```

---

### Task 13: skylib/README.md — Starlark rules reference

**Files:**
- Create: `skylib/README.md`

**Step 1: Write skylib/README.md**

Structure:

```markdown
# Starlark Rules Reference

## Public API

The public API is exported from `@rules_gitops//gitops:defs.bzl`:

\```python
load("@rules_gitops//gitops:defs.bzl", "k8s_deploy", "k8s_test_setup", "external_image")
\```

### k8s_deploy

<Full macro documentation. Copy/expand from the existing Starlark docstring.
Include the full attributes table with: name, type, default, description.
All 32 parameters.>

### k8s_test_setup

<Rule documentation with attributes table.>

### external_image

<Rule documentation with attributes table.>

## Image Pushing

### k8s_container_push

<Rule documentation from push.bzl. Full attributes table.>

### K8sPushInfo Provider

<Field descriptions table.>

## Kustomize

### kustomize (rule)

<Rule documentation with attributes table.>

### KustomizeInfo Provider

<Field descriptions.>

## Supporting Rules

### show
### kubectl
### gitops
### expand_template
### merge_files
### stamp / stamp_value / more_stable_status
### workspace_binary

## Module Extension

### gitops (module extension)

<How to set up the kustomize binary download in MODULE.bazel.>
```

For each rule, include: brief description, attributes table, example usage, and outputs where applicable.

**Step 2: Commit**

```bash
git add skylib/README.md
git commit -m "docs: add Starlark rules reference in skylib/README.md"
```

---

### Task 14: skylib/kustomize/README.md

**Files:**
- Create: `skylib/kustomize/README.md`

**Step 1: Write README**

Content:
- What the kustomize subsystem does (downloads kustomize binary, provides build rules)
- The module extension: how `gitops` extension in `extensions.bzl` registers `kustomize_bin`
- Supported platforms (darwin_amd64, darwin_arm64, linux_amd64, linux_arm64)
- Kustomize version (5.4.3)
- How to use: `use_extension` + `use_repo` in MODULE.bazel
- The `kustomize`, `kubectl`, `gitops` rules and what they do (brief, link to skylib/README.md for full reference)

**Step 2: Commit**

```bash
git add skylib/kustomize/README.md
git commit -m "docs: add kustomize subsystem README"
```

---

### Task 15: examples/README.md

**Files:**
- Create: `examples/README.md`

**Step 1: Write README**

Content:

```markdown
# Examples

## Prerequisites

- Bazel 8+ with bzlmod
- Docker (for building and pushing container images)
- kind (for e2e tests with a local Kubernetes cluster)
- kubectl

## Helloworld

The `helloworld/` directory demonstrates three deployment patterns using
`k8s_deploy`:

### Development (mynamespace)

Personal namespace deployment using `{BUILD_USER}` as the namespace.
Direct kubectl apply — no gitops PR workflow.

\```bash
bazel run //helloworld:mynamespace.apply
\```

### Canary (canary)

Gitops deployment to a shared namespace with:
- `-canary` name suffix and modified app labels
- Digest-based image tagging
- Deployment branch: `helloworld-canary`

\```bash
bazel run //helloworld:canary.show    # inspect rendered manifests
bazel run //helloworld:canary.gitops  # generate gitops output
\```

### Production (release)

Gitops deployment to production with:
- Digest-based image tagging
- Deployment branch: `helloworld-prod`
- Tagged as `release` for filtering

\```bash
bazel run //helloworld:release.show
bazel run //helloworld:release.gitops
\```

## Building

\```bash
cd examples
bazel build //...
\```

## E2E Testing

The e2e test creates a kind cluster, pushes images to a local registry, and
runs the full deployment lifecycle:

\```bash
./e2e_test.sh
\```
```

**Step 2: Commit**

```bash
git add examples/README.md
git commit -m "docs: add examples README with helloworld walkthrough"
```

---

### Task 16: Final review and verification

**Step 1: Verify all links in root README**

Check that every relative link in README.md points to a file that exists:

```bash
grep -oP '\[.*?\]\((.*?)\)' README.md | grep -oP '\(.*?\)' | tr -d '()' | while read f; do [ -f "$f" ] || echo "BROKEN: $f"; done
```

**Step 2: Verify Go builds**

```bash
go vet ./...
```

**Step 3: Verify Bazel loads**

```bash
bazel build //skylib:k8s.bzl --nobuild 2>&1 | head -20
```

Or simply: `bazel query //...` to ensure all files parse.

**Step 4: Final commit if any fixes needed**

```bash
git add -A
git commit -m "docs: fix any broken links or syntax issues from review"
```
