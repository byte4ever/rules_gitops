# stamper

Package `stamper` reads Bazel workspace status files and substitutes `{VAR}`
placeholders in a format string. It uses single-brace `{VAR}` syntax,
distinct from the double-brace `{{VAR}}` syntax used by the `templating`
package.

## Library API

```go
import "github.com/byte4ever/rules_gitops/stamper"

func LoadStamps(infoFiles []string) (map[string]interface{}, error)
func Stamp(infoFiles []string, format string) (string, error)
```

### LoadStamps

Reads one or more workspace status files and merges them into a single map.
When multiple files define the same key, later files override earlier ones.

### Stamp

Loads stamps via `LoadStamps`, then substitutes every `{KEY}` in `format`
with the corresponding value. Unknown variables are preserved as-is in the
output (e.g., `{MISSING}` remains literal).

### Workspace status file format

Each file contains key-value pairs, one per line, with the first space as the
delimiter. Lines without a space are silently skipped. Values may contain
spaces.

```
BUILD_USER alice
GIT_SHA deadbeef
BUILD_MSG hello world from CI
```

## CLI

Binary at `stamper/cmd/`. Reads workspace status files and applies stamp
substitution to a format string or format file.

```
stamper [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--stamp-info-file PATH` | Workspace status file (repeatable) |
| `--output PATH` | Output file (default: stdout) |
| `--format STRING` | Format string containing `{VAR}` placeholders |
| `--format-file PATH` | File containing the format string |

Only one of `--format` or `--format-file` may be specified.

### Example

Given a workspace status file `status.txt`:

```
BUILD_USER alice
GIT_SHA deadbeef
```

```bash
stamper \
  --stamp-info-file status.txt \
  --format 'deployed by {BUILD_USER} at {GIT_SHA}'
```

**Output:**

```
deployed by alice at deadbeef
```
