# resolver

Package `resolver` walks multi-document YAML and substitutes container image
references using a provided image map. It is the final stage of the manifest
pipeline, replacing short image names with fully-qualified registry references
(typically `registry/repo@sha256:digest`).

## Library API

```go
import "github.com/byte4ever/rules_gitops/resolver"

func ResolveImages(in io.Reader, out io.Writer, imgMap map[string]string) error
```

`ResolveImages` reads multi-document YAML from `in`, replaces every container
`image` field whose value matches a key in `imgMap` with the corresponding
value, and writes the result to `out`.

### Image map format

Keys are image names exactly as they appear in the YAML `image:` fields. Values
are the fully-qualified replacements:

```go
imgMap := map[string]string{
    "myapp":       "gcr.io/project/myapp@sha256:abc123...",
    "filewatcher": "docker.io/kube/filewatcher/image:v1.2",
}
```

### Processing details

- Input is decoded as a stream of YAML documents (separated by `---`).
- Each document must have `metadata.name` and `kind` fields; missing either is
  an error.
- The resolver walks the document tree looking for `containers`,
  `initContainers`, `container`, and `spec` fields, then checks each `image`
  value against the map.
- Images prefixed with `//` (Bazel label syntax) that are not present in the
  map produce an error, catching unresolved build references.
- Unknown placeholders that do not start with `//` are left unchanged.

## CLI

Binary at `resolver/cmd/`. Reads YAML from stdin or a file, writes the
resolved output to stdout or a file.

```
resolver [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--infile PATH` | Input YAML file (default: stdin) |
| `--outfile PATH` | Output YAML file (default: stdout) |
| `--image NAME=VALUE` | Image substitution entry (repeatable) |

### Example

```bash
resolver \
  --infile manifests.yaml \
  --outfile resolved.yaml \
  --image "myapp=gcr.io/project/myapp@sha256:abc123..." \
  --image "sidecar=docker.io/team/sidecar:v2"
```

**Input YAML:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  containers:
  - name: app
    image: myapp
  - name: watcher
    image: sidecar
```

**Output YAML:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  containers:
  - name: app
    image: gcr.io/project/myapp@sha256:abc123...
  - name: watcher
    image: docker.io/team/sidecar:v2
```
