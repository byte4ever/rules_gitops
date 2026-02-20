# prer

Package `prer` orchestrates the creation of gitops deployment pull requests.
Given a Bazel workspace with `k8s_deploy` targets, it queries Bazel for gitops
targets, groups them into deployment trains, clones the target git repository,
runs each target to produce manifests, stamps files with build metadata, pushes
OCI images in parallel, and opens pull requests on the configured git hosting
platform (GitHub, GitLab, or Bitbucket).

The CLI binary is `create_gitops_prs`, located at `gitops/prer/cmd/main.go`.
The library entry point is the `Run` function, which accepts a `Config` struct.

## Config struct

| Field | Type | Description |
|---|---|---|
| `BazelCmd` | `string` | Bazel binary name or path. |
| `Workspace` | `string` | Bazel workspace root directory. |
| `Target` | `string` | Bazel query target pattern (e.g. `//...`). |
| `GitRepo` | `string` | Remote git repository URL to clone and push to. |
| `GitMirror` | `string` | Optional local git mirror path for faster reference clones. |
| `GitopsPath` | `string` | Subdirectory for sparse checkout. Empty means the repository root. |
| `TmpDir` | `string` | Directory for temporary clones. |
| `ReleaseBranch` | `string` | Release branch value used to filter targets by their `release_branch_prefix` attribute. |
| `PrimaryBranch` | `string` | Primary branch name (e.g. `main`). Deployment branches are created from this. |
| `DeploymentBranchPrefix` | `string` | Prefix prepended to deployment branch names. |
| `DeploymentBranchSuffix` | `string` | Suffix appended to deployment branch names. |
| `BranchName` | `string` | Source branch name injected into stamp context as `STABLE_GIT_BRANCH`. |
| `GitCommit` | `string` | Source commit SHA injected into stamp context as `STABLE_GIT_COMMIT`. |
| `PushParallelism` | `int` | Number of concurrent image push worker goroutines. |
| `GitopsKinds` | `[]string` | Bazel rule kinds to include in the gitops cquery (e.g. `gitops`, `k8s_deploy`). |
| `GitopsRuleNames` | `[]string` | Rule names used to build the push dependency query (e.g. `push_image`). |
| `GitopsRuleAttrs` | `[]string` | Rule attributes used when building push dependency queries. |
| `PRTitle` | `string` | Title for created pull requests. |
| `PRBody` | `string` | Body for created pull requests. When empty, the provider uses the title as the body. |
| `DryRun` | `bool` | When true, skip image push, git push, and PR creation. |
| `Stamp` | `bool` | When true, apply `{{VAR}}` template substitution to changed files using stamp context. |
| `Provider` | `git.GitProvider` | Strategy implementation that creates pull requests on the target platform. |

## CLI flags

The `create_gitops_prs` binary exposes every `Config` field as a CLI flag.
Flags are grouped by category below.

### Bazel

| Flag | Default | Description |
|---|---|---|
| `--bazel_cmd` | `bazel` | Bazel command name or path. |
| `--workspace` | | Bazel workspace root directory. |
| `--target` | | Bazel query target pattern. |

### Git repository

| Flag | Default | Description |
|---|---|---|
| `--git_repo` | | Remote git repository URL. |
| `--git_mirror` | | Local git mirror for reference clones. |
| `--gitops_path` | | Subdirectory for sparse checkout. |
| `--tmp_dir` | `os.TempDir()` | Temporary directory for clones. |

### Branch

| Flag | Default | Description |
|---|---|---|
| `--release_branch` | | Release branch to filter targets by. |
| `--primary_branch` | `main` | Primary branch name. |
| `--deployment_branch_prefix` | `deploy/` | Prefix for deployment branch names. |
| `--deployment_branch_suffix` | | Suffix for deployment branch names. |
| `--branch_name` | | Source branch name for stamp context. |
| `--git_commit` | | Source commit SHA for stamp context. |

### Push

| Flag | Default | Description |
|---|---|---|
| `--push_parallelism` | `4` | Number of concurrent image push workers. |

### Repeatable

These flags can be specified multiple times to build a list.

| Flag | Description |
|---|---|
| `--gitops_kind` | Rule kind to query (e.g. `--gitops_kind=gitops --gitops_kind=k8s_deploy`). |
| `--gitops_rule_name` | Rule name for push dependency query. |
| `--gitops_rule_attr` | Rule attribute for push dependency queries. |

### PR

| Flag | Default | Description |
|---|---|---|
| `--pr_title` | `GitOps deployment` | Title for created pull requests. |
| `--pr_body` | | Body for created pull requests. |
| `--dry_run` | `false` | Skip push and PR creation. |
| `--stamp` | `false` | Enable file stamping. |

### Provider selection

| Flag | Default | Description |
|---|---|---|
| `--git_server` | `github` | Git hosting platform: `github`, `gitlab`, or `bitbucket`. |

### GitHub-specific

| Flag | Description |
|---|---|
| `--github_repo_owner` | GitHub repository owner. |
| `--github_repo` | GitHub repository name. |
| `--github_access_token` | GitHub personal access token. |
| `--github_enterprise_host` | GitHub Enterprise hostname (omit for github.com). |

### GitLab-specific

| Flag | Description |
|---|---|
| `--gitlab_host` | GitLab instance URL. |
| `--gitlab_repo` | GitLab project path (`org/project`). |
| `--gitlab_access_token` | GitLab personal access token. |

### Bitbucket-specific

| Flag | Description |
|---|---|
| `--bitbucket_api_endpoint` | Bitbucket Server REST API URL. |
| `--bitbucket_user` | Bitbucket API username. |
| `--bitbucket_password` | Bitbucket API password or token. |

## Workflow

The `Run` function executes the following steps in order:

1. **Query Bazel for gitops targets.** Builds a `kind()` cquery expression from
   the configured `GitopsKinds` and `Target` pattern, then runs
   `bazel cquery --output=jsonproto` to retrieve all matching configured targets
   with their rule attributes.

2. **Group targets by deployment train.** Parses the cquery results and groups
   targets by their `deployment_branch` attribute. Only targets whose
   `release_branch_prefix` matches `ReleaseBranch` are included. If no targets
   match, the run exits early.

3. **Clone the git repository.** Clones `GitRepo` into a temporary directory
   under `TmpDir`. When `GitMirror` is set, the clone uses it as a local
   reference to reduce network transfer. If `GitopsPath` is set, a sparse
   checkout restricts the working tree to that subdirectory. The clone is cleaned
   up on return. Deployment branch patterns are fetched after the initial clone.

4. **Process each deployment train.** For each group of targets sharing a
   deployment branch:
   - Switches to (or creates) the deployment branch, named
     `{DeploymentBranchPrefix}{branch}{DeploymentBranchSuffix}`, based off the
     primary branch.
   - Checks the previous commit message for the list of previously deployed
     targets. If any targets were removed since the last deployment, the branch
     is recreated from the primary branch to avoid stale manifests.
   - Runs each target executable (converted from Bazel label to binary path)
     in the workspace directory, producing manifest files in the clone.
   - When `Stamp` is enabled, iterates changed files, verifies SHA256 digests
     (restoring files whose content has not changed), and applies `{{VAR}}`
     template substitution using `STABLE_GIT_COMMIT`, `STABLE_GIT_BRANCH`,
     `BUILD_TIMESTAMP`, `BUILD_EMBED_LABEL`, `RANDOM_SEED`, and
     `STABLE_BUILD_LABEL`.
   - Commits the changes with a message encoding the target list (used for
     deletion detection on the next run).

5. **Push images.** Builds a dependency query from `GitopsRuleNames` across all
   targets to discover push targets. Runs the push target executables in a
   worker pool bounded by `PushParallelism` goroutines. Context cancellation
   stops scheduling new work; errors from individual pushes are collected and the
   first is returned.

6. **Push git branches.** Pushes all updated deployment branches to the remote
   in a single operation. Skipped when `DryRun` is true.

7. **Create pull requests.** Calls `Provider.CreatePR` for each updated
   deployment branch, opening a PR from the deployment branch into the primary
   branch with the configured title and body. Skipped when `DryRun` is true.

## Usage example

Full invocation with the GitHub provider, deploying targets from a release
branch to a gitops repository:

```sh
create_gitops_prs \
  --bazel_cmd=bazel \
  --workspace=/src/workspace \
  --target="//deploy/..." \
  --git_repo=https://github.com/myorg/gitops-config.git \
  --gitops_path=clusters/production \
  --release_branch=release/v2.1 \
  --primary_branch=main \
  --deployment_branch_prefix=deploy/ \
  --branch_name=release/v2.1 \
  --git_commit=abc123def456 \
  --push_parallelism=8 \
  --gitops_kind=gitops \
  --gitops_rule_name=push_image \
  --pr_title="Deploy release/v2.1" \
  --pr_body="Automated deployment from CI pipeline" \
  --stamp \
  --git_server=github \
  --github_repo_owner=myorg \
  --github_repo=gitops-config \
  --github_access_token="$GITHUB_TOKEN"
```

Dry-run mode previews which branches would be updated without pushing or
creating PRs:

```sh
create_gitops_prs \
  --workspace=/src/workspace \
  --target="//deploy/..." \
  --git_repo=https://github.com/myorg/gitops-config.git \
  --release_branch=release/v2.1 \
  --gitops_kind=gitops \
  --dry_run
```
