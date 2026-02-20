# bazel

Converts Bazel target labels to executable paths under `bazel-bin/`.

## API

| Function | Description |
|---|---|
| `TargetToExecutable(target string) string` | Converts a `//pkg:name` label to its `bazel-bin/pkg/name` path. Non-label inputs are returned unchanged. |

## Usage

```go
import "github.com/byte4ever/rules_gitops/gitops/bazel"

// //app/deploy:deploy-prod.gitops → bazel-bin/app/deploy/deploy-prod.gitops
path := bazel.TargetToExecutable("//app/deploy:deploy-prod.gitops")

// Inputs without the // prefix pass through untouched.
same := bazel.TargetToExecutable("some/local/path") // "some/local/path"
```

The conversion strips the leading `//`, replaces the first `:` with `/`, and prepends `bazel-bin/`.
