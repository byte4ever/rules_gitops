# bitbucket

Package `bitbucket` implements `git.GitProvider` for creating pull requests on
Bitbucket Server (Stash). This targets the Bitbucket Server REST API, not
Bitbucket Cloud.

```
import "github.com/byte4ever/rules_gitops/gitops/git/bitbucket"
```

Uses `net/http` with JSON payloads directly against the REST API. No external
Bitbucket client library is required.

## Config

| Field         | Type   | Required | Description |
|---------------|--------|----------|-------------|
| `APIEndpoint` | string | yes      | Full REST API URL for pull requests, including project and repo path (e.g. `https://bb.example.com/rest/api/1.0/projects/PROJ/repos/repo/pull-requests`). |
| `User`        | string | yes      | Bitbucket API username. |
| `Password`    | string | yes      | Bitbucket API password or personal access token. |

Authentication uses HTTP Basic Auth with the `User` and `Password` fields.

## CreatePR Behavior

`CreatePR` sends a POST to the configured `APIEndpoint` with the pull request
payload. The provider considers two status codes as success:

- **201 Created** -- the pull request was created.
- **409 Conflict** -- a pull request for that branch pair already exists (logs
  "reusing existing pull request" and returns nil).

Any other status code is returned as an error.

## Usage

```go
provider, err := bitbucket.NewProvider(bitbucket.Config{
    APIEndpoint: "https://bb.example.com/rest/api/1.0/projects/PROJ/repos/infra/pull-requests",
    User:        os.Getenv("BITBUCKET_USER"),
    Password:    os.Getenv("BITBUCKET_TOKEN"),
})
if err != nil {
    return err
}

err = provider.CreatePR(ctx, "feature/deploy", "main", "Deploy v1.2", "Release notes")
```
