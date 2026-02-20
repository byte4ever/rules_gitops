# git

Package `git` provides local git repository operations and a strategy interface
for creating pull requests across different git hosting platforms.

```
import "github.com/byte4ever/rules_gitops/gitops/git"
```

## GitProvider Interface

`GitProvider` is the strategy interface that abstracts PR creation. Each hosting
platform implements this interface in a sub-package:

| Sub-package                | Platform                  |
|----------------------------|---------------------------|
| `gitops/git/github`        | GitHub (cloud and enterprise) |
| `gitops/git/gitlab`        | GitLab                    |
| `gitops/git/bitbucket`     | Bitbucket Server (Stash)  |

```go
type GitProvider interface {
    CreatePR(ctx context.Context, from, to, title, body string) error
}
```

### GitProviderFunc

`GitProviderFunc` is a function adapter that satisfies the `GitProvider`
interface, following the same pattern as `http.HandlerFunc`. When `body` is
empty, `title` is substituted automatically.

```go
provider := git.GitProviderFunc(
    func(ctx context.Context, from, to, title, body string) error {
        fmt.Printf("PR: %s -> %s\n", from, to)
        return nil
    },
)

err := provider.CreatePR(ctx, "feature/x", "main", "Add feature", "")
// body will be set to "Add feature" since it was empty
```

## Repo

`Repo` represents a local clone of a git repository. Create one with `Clone`
and call `Clean` when done.

```go
type Repo struct {
    Dir        string // filesystem location of the clone
    RemoteName string // name of the upstream remote
}
```

### Clone

```go
func Clone(repo, dir, mirrorDir, primaryBranch, gitopsPath string) (*Repo, error)
```

Clones a repository into `dir` from the full URL in `repo`. The clone uses
`--no-checkout`, `--single-branch`, `--filter=blob:none`, and `--no-tags` for
speed.

**Mirror optimization**: when `mirrorDir` is non-empty, it is passed as
`--reference` to `git clone`. This lets git reuse objects from an existing local
mirror instead of downloading them from the remote, which significantly reduces
clone time and network traffic in CI environments where multiple clones of the
same repository are common.

**Sparse checkout**: when `gitopsPath` is not empty or `"."`, sparse-checkout is
enabled so only the specified subtree is materialized.

### Methods

| Method | Description |
|--------|-------------|
| `Clean() error` | Removes the local clone directory. |
| `Fetch(pattern string)` | Adds `pattern` to tracked remote branches and fetches. |
| `SwitchToBranch(branch, primaryBranch string) bool` | Checks out `branch`, creating it from `primaryBranch` if it does not exist. Returns `true` when the branch was newly created. |
| `RecreateBranch(branch, primaryBranch string)` | Discards the content of `branch` and resets it from `primaryBranch`. |
| `GetLastCommitMessage() string` | Returns the most recent commit message on the current branch. Returns an empty string on error. |
| `Commit(message, gitopsPath string) bool` | Stages changes under `gitopsPath` and commits. Returns `true` when changes were committed, `false` when the tree was clean. |
| `RestoreFile(fileName string)` | Restores the specified file to its last-committed state. |
| `GetChangedFiles() []string` | Returns file paths with unstaged changes. |
| `IsClean() bool` | Reports whether the working tree has no uncommitted changes. |
| `Push(branches []string)` | Force-pushes the given branches to the remote. |

### Usage

```go
repo, err := git.Clone(
    "https://github.com/org/repo.git",
    "/tmp/work",
    "/var/cache/mirrors/repo.git", // mirror dir (optional, "" to skip)
    "main",
    "deploy/production",           // sparse-checkout path
)
if err != nil {
    return err
}
defer repo.Clean()

created := repo.SwitchToBranch("gitops/deploy-prod", "main")
// ... write manifests ...
if repo.Commit("deploy: update production manifests", "deploy/production") {
    repo.Push([]string{"gitops/deploy-prod"})
}
```
