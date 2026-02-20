package digester

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// CalculateDigest computes the SHA256 hex digest of the file at
// path. Returns empty string with no error if the file does not
// exist.
func CalculateDigest(path string) (result string, retErr error) {
	const errCtx = "calculating digest"

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	fi, err := os.Open(path) //nolint:gosec // path is caller-provided by design
	if err != nil {
		return "", fmt.Errorf("%s: %w", errCtx, err)
	}

	defer func() {
		if closeErr := fi.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("%s: %w", errCtx, closeErr)
		}
	}()

	ha := sha256.New()

	if _, err := io.Copy(ha, fi); err != nil {
		return "", fmt.Errorf("%s: %w", errCtx, err)
	}

	return hex.EncodeToString(ha.Sum(nil)), nil
}

// GetDigest reads a stored digest from a sidecar .digest file.
// Returns empty string with no error if the sidecar file does
// not exist.
func GetDigest(path string) (string, error) {
	const errCtx = "getting stored digest"

	dp := path + ".digest"

	if _, err := os.Stat(dp); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	digest, err := os.ReadFile(dp) //nolint:gosec // path is caller-provided by design
	if err != nil {
		return "", fmt.Errorf("%s: %w", errCtx, err)
	}

	return string(digest), nil
}

// VerifyDigest compares the calculated digest of the file
// against its stored sidecar digest.
func VerifyDigest(path string) (bool, error) {
	const errCtx = "verifying digest"

	calc, err := CalculateDigest(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errCtx, err)
	}

	stored, err := GetDigest(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errCtx, err)
	}

	return calc == stored, nil
}

// SaveDigest calculates the digest of a file and writes it
// to a .digest sidecar file.
func SaveDigest(path string) error {
	const errCtx = "saving digest"

	digest, err := CalculateDigest(path)
	if err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	dp := path + ".digest"

	if err := os.WriteFile(dp, []byte(digest), 0o600); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}
