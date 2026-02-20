# gitlab

Package `gitlab` implements `git.GitProvider` for creating merge requests on
GitLab.

```
import "github.com/byte4ever/rules_gitops/gitops/git/gitlab"
```

Uses [gitlab.com/gitlab-org/api/client-go](https://gitlab.com/gitlab-org/api/client-go)
as the underlying client.

## Config

| Field         | Type   | Required | Description |
|---------------|--------|----------|-------------|
| `Host`        | string | no       | Base URL of the GitLab instance (e.g. `https://gitlab.example.com`). Defaults to `https://gitlab.com` when empty. |
| `Repo`        | string | yes      | Full project path including namespace (e.g. `org/project` or `group/subgroup/project`). |
| `AccessToken` | string | yes      | Personal or project access token. |

## CreatePR Behavior

`CreatePR` opens a merge request from branch `from` into branch `to`. If a
merge request for that source branch already exists, GitLab returns HTTP 409
and the provider treats this as success (logs "reusing existing merge request"
and returns nil).

The `body` parameter is ignored -- only `title` is sent to the GitLab API.

## Usage

```go
provider, err := gitlab.NewProvider(gitlab.Config{
    Host:        "https://gitlab.example.com",
    Repo:        "platform/infrastructure",
    AccessToken: os.Getenv("GITLAB_TOKEN"),
})
if err != nil {
    return err
}

err = provider.CreatePR(ctx, "feature/deploy", "main", "Deploy v1.2", "")
```
