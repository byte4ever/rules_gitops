# digester

Calculates and verifies SHA256 file digests using companion `.digest` sidecar files.

## API

| Function | Description |
|---|---|
| `CalculateDigest(path string) (string, error)` | Computes the SHA256 hex digest of a file. Returns `""` with no error if the file does not exist. |
| `SaveDigest(path string) error` | Calculates the digest and writes it to `<path>.digest`. |
| `GetDigest(path string) (string, error)` | Reads the stored digest from `<path>.digest`. Returns `""` with no error if the sidecar does not exist. |
| `VerifyDigest(path string) (bool, error)` | Returns true if the file's current SHA256 matches its stored `.digest` sidecar. |

## Usage

```go
import "github.com/byte4ever/rules_gitops/gitops/digester"

// Save a digest alongside the file.
err := digester.SaveDigest("/tmp/image.tar")
// Creates /tmp/image.tar.digest containing the SHA256 hex string.

// Verify the file hasn't changed since the digest was saved.
ok, err := digester.VerifyDigest("/tmp/image.tar")
if !ok {
    // File content has diverged from stored digest.
}

// Read the stored digest directly.
stored, err := digester.GetDigest("/tmp/image.tar")

// Compute a fresh digest without touching the sidecar file.
fresh, err := digester.CalculateDigest("/tmp/image.tar")
```

This is used for skip-if-unchanged optimizations during image pushes.
