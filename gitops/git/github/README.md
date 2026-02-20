# github

Package `github` implements `git.GitProvider` for creating pull requests on
GitHub, supporting both github.com and GitHub Enterprise installations.

```
import "github.com/byte4ever/rules_gitops/gitops/git/github"
```

Uses [google/go-github/v68](https://github.com/google/go-github) as the
underlying client.

## Config

| Field            | Type   | Required | Description |
|------------------|--------|----------|-------------|
| `RepoOwner`      | string | yes      | GitHub user or organisation that owns the repository. |
| `Repo`           | string | yes      | Repository name (without owner prefix). |
| `AccessToken`    | string | yes      | Personal access token or GitHub App token. |
| `EnterpriseHost` | string | no       | GitHub Enterprise hostname (e.g. `git.corp.example.com`). Leave empty for github.com. |

When `EnterpriseHost` is set, the provider constructs the API base URL as
`https://<host>/api/v3/` and the upload URL as `https://<host>/api/uploads/`.

## CreatePR Behavior

`CreatePR` opens a pull request from branch `from` into branch `to`. If a PR
for that head/base pair already exists, GitHub returns HTTP 422 and the provider
treats this as success (logs "reusing existing pull request" and returns nil).

## Usage

```go
provider, err := github.NewProvider(github.Config{
    RepoOwner:   "my-org",
    Repo:        "my-repo",
    AccessToken: os.Getenv("GITHUB_TOKEN"),
})
if err != nil {
    return err
}

err = provider.CreatePR(ctx, "feature/deploy", "main", "Deploy v1.2", "Release notes")
```
