# templating

Package `templating` provides a fast template engine built on
`valyala/fasttemplate`. It expands `{{VAR}}` placeholders in template files
using explicit variables, file imports, and Bazel workspace status stamps.

## Library API

```go
import "github.com/byte4ever/rules_gitops/templating"

type Engine struct {
    StartTag       string   // default: "{{"
    EndTag         string   // default: "}}"
    StampInfoFiles []string // workspace status file paths
}

func (en *Engine) Expand(
    tplPath string,
    outPath string,
    vars []string,
    imports []string,
    executable bool,
) error
```

### Engine configuration

- `StartTag` / `EndTag` -- delimiters for template placeholders. Default to
  `{{` and `}}`. Can be set to any string pair (e.g., `<%` / `%>`).
- `StampInfoFiles` -- paths to Bazel workspace status files. Loaded as
  key-value pairs (same format as the `stamper` package).

### Expand

Reads the template at `tplPath` (or stdin if empty), substitutes all
placeholders, and writes the result to `outPath` (or stdout if empty). When
`executable` is true, the output file receives mode `0777`.

**Processing order:**

1. Load stamp files into a stamp map.
2. For each variable `NAME=VALUE`, expand `VALUE` against stamps using
   single-brace `{VAR}` syntax, then store the result as both `NAME` and
   `variables.NAME` in the context.
3. For each import `NAME=filename`, read the file, expand it against the
   context with the configured tags, then expand again against stamps with
   single-brace tags, and store as `imports.NAME` in the context.
4. Expand the template against the full context.

Variables override stamp values when they share the same key. Unknown
placeholders are preserved as-is in the output.

### Variable format

`NAME=VALUE` pairs. The value portion undergoes stamp expansion before
being stored, so `{STAMP_VAR}` references inside values are resolved.

```go
vars := []string{"APP=myapp", "AUTHOR={BUILD_USER}"}
```

### Import format

`NAME=filename` pairs. The file contents are read, expanded against the
current context, then stamp-expanded:

```go
imports := []string{"config=path/to/config.yaml"}
```

The result is available in the template as `{{imports.config}}`.

## CLI

Binary at `templating/cmd/` (`fast_template_engine`). Expands a template file
using stamp files, variables, and imports.

```
fast_template_engine [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--stamp_info_file PATH` | Workspace status file (repeatable) |
| `--variable NAME=VALUE` | Template variable (repeatable) |
| `--imports NAME=FILENAME` | File import (repeatable) |
| `--template PATH` | Input template file (default: stdin) |
| `--output PATH` | Output file (default: stdout) |
| `--executable` | Set executable bit on output file |
| `--start_tag TAG` | Start delimiter (default: `{{`) |
| `--end_tag TAG` | End delimiter (default: `}}`) |

### Example

Given a stamp file `status.txt`:

```
BUILD_USER alice
BUILD_HOST ci-01
```

And a template `deploy.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{APP}}
  annotations:
    built-by: {{BUILD_USER}}
    host: {{BUILD_HOST}}
data:
  version: {{variables.VERSION}}
```

```bash
fast_template_engine \
  --stamp_info_file status.txt \
  --variable APP=myservice \
  --variable VERSION=1.2.3 \
  --template deploy.yaml \
  --output resolved.yaml
```

**Output (`resolved.yaml`):**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: myservice
  annotations:
    built-by: alice
    host: ci-01
data:
  version: 1.2.3
```
