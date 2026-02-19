package digester_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/digester"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateDigest_returns_sha256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pa := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(pa, []byte("hello"), 0o600))

	got, err := digester.CalculateDigest(pa)

	require.NoError(t, err)
	// sha256("hello")
	assert.Equal(
		t,
		"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		got,
	)
}

func TestCalculateDigest_nonexistent_file(t *testing.T) {
	t.Parallel()

	got, err := digester.CalculateDigest("/nonexistent")

	assert.Empty(t, got)
	assert.NoError(t, err)
}

func TestSaveDigest_and_GetDigest_roundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pa := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(pa, []byte("content"), 0o600))

	require.NoError(t, digester.SaveDigest(pa))

	got, err := digester.GetDigest(pa)
	require.NoError(t, err)

	expected, err := digester.CalculateDigest(pa)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestVerifyDigest_valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pa := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(pa, []byte("content"), 0o600))
	require.NoError(t, digester.SaveDigest(pa))

	ok, err := digester.VerifyDigest(pa)

	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyDigest_tampered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pa := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(pa, []byte("content"), 0o600))
	require.NoError(t, digester.SaveDigest(pa))

	require.NoError(t, os.WriteFile(pa, []byte("tampered"), 0o600))

	ok, err := digester.VerifyDigest(pa)

	require.NoError(t, err)
	assert.False(t, ok)
}

func FuzzCalculateDigest(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Parallel()

		dir := t.TempDir()
		pa := filepath.Join(dir, "fuzz.bin")
		require.NoError(t, os.WriteFile(pa, data, 0o600))

		dg, err := digester.CalculateDigest(pa)

		require.NoError(t, err)
		assert.Len(t, dg, 64) // sha256 hex is always 64 chars
	})
}
