# Architecture

This document describes the internal architecture of `rules_gitops`: the
manifest pipeline, Starlark rule graph, Go package dependencies, the
GitProvider strategy pattern, and the worker pool used for image pushes.

---

## 1. Manifest Pipeline

Kubernetes manifests pass through a three-stage pipeline before they are
applied or committed to a gitops repository. A separate stamper path handles
Bazel workspace status substitution.

```
                            Bazel workspace status files
                                       |
                                       v
                                   stamper
                                 ({VAR} -> value)
                                       |
                                       v
                              kustomization.yaml
                                       |
                                       v
  *.yaml manifests ──> kustomize build ──> templating ──> resolver ──> final.yaml
                       (stage 1)          (stage 2)      (stage 3)
```

**Stage 1 -- kustomize build** (`@kustomize_bin//:kustomize`)

- Input: Raw YAML manifests, patches, configmaps, secrets, a generated
  `kustomization.yaml`.
- Action: Runs `kustomize build --load-restrictor LoadRestrictionsNone
  --reorder legacy` to produce a single multi-document YAML stream.
- Output: Merged, patched YAML piped to stdout.

If the kustomization contains single-brace `{VAR}` placeholders (e.g.
`{BUILD_USER}` in a namespace field), the generated `kustomization.yaml` is
first run through the **stamper** (`//stamper/cmd:stamper`) to replace those
placeholders with values from Bazel workspace status files
(`more_stable_status.txt`). This is a separate path that runs at build time
before kustomize executes.

**Stage 2 -- templating** (`//templating/cmd:templating`)

- Input: kustomize output piped on stdin, `--variable` flags, an optional
  `--stamp_info_file`, and `--imports` for file-based values.
- Action: Replaces double-brace `{{VAR}}` placeholders using
  `valyala/fasttemplate`. The start/end tags default to `{{`/`}}` but are
  configurable per rule.
- Output: Expanded YAML with template variables resolved.

**Stage 3 -- resolver** (`//resolver/cmd:resolver`)

- Input: Templated YAML piped on stdin, `--image` flags mapping Bazel labels
  to `registry/repo@sha256:digest` pairs.
- Action: Scans the YAML for image references matching `//label:image`
  patterns and replaces them with fully-qualified
  `registry/repository@sha256:...` references using digest files produced by
  the push rules.
- Output: Final deployment-ready YAML with pinned image digests.

The full shell pipeline assembled in `kustomize.bzl` looks like:

```
kustomize build ... | templating --template=... --variable=... | resolver --image ...=...@$(cat digest) > out.yaml
```

---

## 2. Starlark Rule Graph

The `k8s_deploy` macro (defined in `skylib/k8s.bzl`) is the main user-facing
entry point. It generates several concrete rule targets from a single macro
invocation.

### Generated targets

Given `k8s_deploy(name = "myapp", ...)`:

| Target            | Rule       | Purpose                                        |
|-------------------|------------|------------------------------------------------|
| `myapp`           | `kustomize`| Runs the 3-stage pipeline, produces `myapp.yaml` |
| `myapp.show`      | `show`     | Renders manifests and prints them to stdout      |
| `myapp.apply`     | `kubectl`  | Pushes images, then `kubectl apply -f -`         |
| `myapp.delete`    | `kubectl`  | `kubectl delete -f -` (no push)                  |
| `myapp.gitops`    | `gitops`   | Writes manifests to a gitops directory tree      |

When `gitops = False` (mynamespace mode), the `.gitops` target is not
generated. When `gitops = True`, the `.delete` target is not generated.

### Rule dependency flow

```
  k8s_container_push (push.bzl)
         |
         | K8sPushInfo provider
         v
     kustomize (kustomize.bzl)
         |
         | KustomizeInfo provider   +   DefaultInfo (yaml output)
         |
    +----+--------+----------+
    |              |          |
    v              v          v
  kubectl        gitops      show
  (.apply/       (.gitops)   (.show)
   .delete)
```

### Key providers

**`K8sPushInfo`** (defined in `skylib/push.bzl`)

Carries image push metadata from `k8s_container_push` to the `kustomize`
rule so the pipeline can resolve image references:

- `image_label` -- Bazel target label of the image
- `legacy_image_name` -- optional short alias
- `registry` -- target registry (e.g. `docker.io`)
- `repository` -- image repository path
- `digestfile` -- `File` containing `sha256:...` digest

**`KustomizeInfo`** (defined in `skylib/kustomize/kustomize.bzl`)

Carries the set of image push targets through the rule graph so that
`kubectl`, `gitops`, and `push_all` rules can locate and execute image
pushes:

- `image_pushes` -- `depset` of targets providing `K8sPushInfo`

The `imagePushStatements` function in `kustomize.bzl` iterates
`KustomizeInfo.image_pushes` to generate async push shell commands used by
both `kubectl` (for `.apply`) and `gitops` rules.

---

## 3. Package Dependency Map

```
gitops/prer ─────────┬──> gitops/git
     |               |         |
     |               |         └──> gitops/exec
     |               |
     ├──> gitops/exec
     ├──> gitops/bazel
     ├──> gitops/commitmsg
     └──> gitops/digester

gitops/git/github ──┐
gitops/git/gitlab ──┼── implement git.GitProvider interface
gitops/git/bitbucket┘

gitops/prer/cmd ──┬──> gitops/prer
                  ├──> gitops/git
                  ├──> gitops/git/github
                  ├──> gitops/git/gitlab
                  └──> gitops/git/bitbucket

testing/it_sidecar/cmd ──┬──> testing/it_sidecar (sidecar library)
                         └──> testing/it_sidecar/stern

testing/it_sidecar/client ──> (no internal deps; orchestrates sidecar subprocess via os/exec)

resolver/         ── standalone (no internal deps)
stamper/          ── standalone (no internal deps)
templating/       ── standalone (no internal deps)
testing/it_manifest_filter/ ── standalone (no internal deps)
```

Key observations:

- **`prer`** is the most connected package. It depends on `git` for repository
  operations, `exec` for shell commands, `bazel` for target-to-executable
  path conversion, `commitmsg` for encoding target lists in commit messages,
  and `digester` for SHA256 verification during stamping.
- **`git`** depends only on `exec` for running git shell commands.
- **Platform providers** (`github`, `gitlab`, `bitbucket`) have no internal
  dependencies -- they only import their respective API client libraries.
- **Pipeline tools** (`resolver`, `stamper`, `templating`) are fully
  standalone and communicate only via stdin/stdout pipes.
- **`sidecar`** and **`stern`** are both independent libraries. The
  `it_sidecar/cmd` binary wires them together. The `client` package
  orchestrates the sidecar as a subprocess.

---

## 4. GitProvider Strategy Pattern

The `gitops/git` package defines a strategy interface that decouples PR
creation logic from any specific hosting platform.

### Interface

```go
// gitops/git/provider.go

type GitProvider interface {
    CreatePR(ctx context.Context, from, to, title, body string) error
}
```

`from` is the source (deployment) branch, `to` is the target (primary)
branch, and `title`/`body` are the PR description.

### Function adapter

```go
type GitProviderFunc func(ctx context.Context, from, to, title, body string) error
```

`GitProviderFunc` satisfies `GitProvider`. If `body` is empty, the adapter
substitutes `title` as the body. This allows passing a plain function where a
full struct implementation is unnecessary (e.g. in tests).

### Implementations

| Package                   | Type                  | Platform                  | API client                        |
|---------------------------|-----------------------|---------------------------|-----------------------------------|
| `gitops/git/github`       | `github.Provider`     | GitHub / GitHub Enterprise| `google/go-github/v68`            |
| `gitops/git/gitlab`       | `gitlab.Provider`     | GitLab (cloud or self-hosted) | `gitlab.com/gitlab-org/api/client-go` |
| `gitops/git/bitbucket`    | `bitbucket.Provider`  | Bitbucket Server (Stash)  | Raw `net/http` + JSON             |

Each provider:

1. Accepts a `Config` struct via `NewProvider(cfg Config) (*Provider, error)`.
2. Validates required fields (token, endpoint, etc.) at construction time.
3. Treats "already exists" responses as success (GitHub 422, GitLab 409,
   Bitbucket 409).

### Factory

The `newGitProvider` function in `gitops/prer/cmd/main.go` selects the
implementation at runtime based on the `--git_server` flag:

```
--git_server=github    --> github.NewProvider(github.Config{...})
--git_server=gitlab    --> gitlab.NewProvider(gitlab.Config{...})
--git_server=bitbucket --> bitbucket.NewProvider(bitbucket.Config{...})
```

An unknown value returns an error.

---

## 5. Worker Pool in prer

The `prer.Run` function orchestrates the full gitops PR creation workflow.
Image pushing is the most expensive step and is parallelized with a
bounded worker pool.

### Workflow overview

```
1. bazel cquery          -- find gitops targets
2. groupByTrain          -- group targets by deployment_branch attribute
3. git.Clone             -- clone the gitops repository
4. for each train:
   a. SwitchToBranch     -- checkout or create the deployment branch
   b. run target exes    -- execute each .gitops target (writes manifests)
   c. stampChangedFiles  -- verify digests, apply {{VAR}} stamps, save digests
   d. Commit             -- commit changes with encoded target list in message
5. pushImages            -- push all container images (worker pool)
6. repo.Push             -- push all updated branches
7. CreatePR              -- create a PR per updated branch
```

### Image push worker pool

The `pushImages` function (in `prer.go`) implements bounded concurrency:

- **Parallelism**: Controlled by `Config.PushParallelism` (default 4, set via
  `--push_parallelism` flag). Falls back to 1 if set to zero or negative.
- **Mechanism**: A buffered channel of size `PushParallelism` acts as a
  semaphore. Each push target is dispatched as a goroutine that acquires a
  semaphore slot before running `bazel.TargetToExecutable` + `exec.Ex`.
- **Error collection**: Errors from individual pushes are collected under a
  mutex. After all goroutines complete (`sync.WaitGroup`), if any errors
  occurred the first is returned.
- **Cancellation**: Before dispatching each goroutine, `ctx.Err()` is checked.
  If the context is cancelled, the loop breaks early.

### Deployment train processing

`processTrain` handles branch lifecycle for a single train:

1. **Branch selection**: `SwitchToBranch` creates the deployment branch from
   the primary branch if it does not exist, or checks it out if it does.
2. **Stale branch detection**: If the branch already existed, the last commit
   message is decoded (via `commitmsg.ExtractTargets`) to find previously
   deployed targets. If any were removed, the branch is recreated from the
   primary branch to avoid stale manifests.
3. **Target execution**: Each gitops target executable is run via `exec.MustEx`
   in the workspace directory.
4. **Stamping**: Changed files are compared by SHA256 digest
   (`digester.VerifyDigest`). Unchanged files are restored; changed files are
   stamped with `{{VAR}}` replacement and their new digest is saved.
5. **Commit**: A commit message encoding the current target list (via
   `commitmsg.Generate`) is created, allowing future runs to detect removals.
