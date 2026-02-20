# commitmsg

Encodes and decodes gitops target lists in git commit messages.

## API

| Function | Description |
|---|---|
| `Generate(targets []string) string` | Produces a commit message section with targets between begin/end markers. |
| `ExtractTargets(msg string) []string` | Parses target labels from a commit message. Returns nil if markers are missing or malformed. |

Targets are delimited by marker lines:

```
--- gitops targets begin ---
//app:deploy.gitops
--- gitops targets end ---
```

## Usage

```go
import "github.com/byte4ever/rules_gitops/gitops/commitmsg"

// Encode targets into a commit message block.
block := commitmsg.Generate([]string{
    "//app:deploy.gitops",
    "//svc:deploy.gitops",
})
commitMsg := "deploy: roll out v1.2.0" + block

// Later, extract the targets back out.
targets := commitmsg.ExtractTargets(commitMsg)
// ["//app:deploy.gitops", "//svc:deploy.gitops"]
```

If the end marker is missing, `ExtractTargets` returns nil.
