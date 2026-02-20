# exec

Shell command execution helpers with logging via `log/slog`.

## API

| Function | Description |
|---|---|
| `Ex(dir, name string, arg ...string) (string, error)` | Runs a command and returns combined stdout+stderr output. Pass empty `dir` to use the current working directory. |
| `MustEx(dir, name string, arg ...string)` | Same as `Ex` but panics on failure. Use in contexts where errors are unrecoverable. |

## Usage

```go
import "github.com/byte4ever/rules_gitops/gitops/exec"

// Run a command and capture output.
out, err := exec.Ex("/path/to/repo", "git", "status", "--short")
if err != nil {
    // err wraps the underlying exec error; out still contains
    // any output produced before the failure.
}

// Fire-and-forget when failure is unrecoverable.
exec.MustEx("", "mkdir", "-p", "/tmp/workspace")
```

Both functions log the command and its output at Info level via `slog`.
