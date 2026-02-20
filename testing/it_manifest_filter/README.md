# it_manifest_filter

Package `filter` transforms Kubernetes manifests for integration testing
environments that lack persistent storage, ingress controllers, and
production certificate infrastructure.

## Transformations

`ReplacePDWithEmptyDirs` processes a multi-document YAML stream and applies
the following changes:

- **Drops PersistentVolumeClaim objects** entirely.
- **Drops Ingress objects** entirely.
- **Converts StatefulSet volumeClaimTemplates** to `emptyDir` volumes. When
  the claim template specifies a `storage` request, the corresponding
  `emptyDir` gets a matching `sizeLimit`. The `volumeClaimTemplates` and
  `status` fields are removed from the StatefulSet.
- **Replaces `persistentVolumeClaim` volume sources** with `emptyDir` across
  all remaining objects (Deployments, DaemonSets, etc.).
- **Replaces `letsencrypt-prod` issuer references** in Certificate objects
  with `letsencrypt-staging`.

## API

```go
func ReplacePDWithEmptyDirs(in io.Reader, out io.Writer) error
```

Reads multi-document YAML from `in`, applies transformations, and writes
the filtered result to `out`. Documents are separated by `---` in the
output.

## CLI

The binary at `testing/it_manifest_filter/cmd/` wraps `ReplacePDWithEmptyDirs`
for command-line use.

```
it_manifest_filter [--infile <path>] [--outfile <path>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--infile` | stdin | Input YAML file path |
| `--outfile` | stdout | Output YAML file path |

Both flags are optional. When omitted, the tool reads from stdin and writes
to stdout, supporting pipeline usage:

```sh
kustomize build . | it_manifest_filter > filtered.yaml
```
