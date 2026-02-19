package exec_test

import (
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/exec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEx_success(t *testing.T) {
	t.Parallel()

	out, err := exec.Ex("", "echo", "hello")

	require.NoError(t, err)
	assert.Contains(t, out, "hello")
}

func TestEx_with_dir(t *testing.T) {
	t.Parallel()

	out, err := exec.Ex("/tmp", "pwd")

	require.NoError(t, err)
	assert.Contains(t, out, "/tmp")
}

func TestEx_failure(t *testing.T) {
	t.Parallel()

	_, err := exec.Ex("", "false")

	assert.Error(t, err)
}

func TestMustEx_panics_on_failure(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		exec.MustEx("", "false")
	})
}

func TestMustEx_success(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		exec.MustEx("", "echo", "ok")
	})
}
