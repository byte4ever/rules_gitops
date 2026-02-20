package git_test

import (
	"context"
	"os"
	oe "os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/gitops/git"
)

func TestIsRootPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "empty string is root",
			path: "",
			want: true,
		},
		{
			name: "dot is root",
			path: ".",
			want: true,
		},
		{
			name: "subdir is not root",
			path: "deploy/k8s",
			want: false,
		},
		{
			name: "single dir is not root",
			path: "gitops",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := git.IsRootPathForTest(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepo_IsClean(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	initGitRepo(t, dir)

	rp := &git.Repo{
		Dir:        dir,
		RemoteName: "origin",
	}

	// A freshly initialised repo with one commit
	// should be clean.
	assert.True(t, rp.IsClean())
}

func TestRepo_IsClean_dirty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	initGitRepo(t, dir)

	rp := &git.Repo{
		Dir:        dir,
		RemoteName: "origin",
	}

	// Create a new file to make the tree dirty.
	fp := filepath.Join(dir, "new.txt")

	//nolint:gosec // test file
	err := os.WriteFile(
		fp, []byte("hello\n"), 0o600,
	)
	require.NoError(t, err)

	assert.False(t, rp.IsClean())
}

func TestRepo_GetLastCommitMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	initGitRepo(t, dir)

	rp := &git.Repo{
		Dir:        dir,
		RemoteName: "origin",
	}

	msg := rp.GetLastCommitMessage()
	assert.Contains(t, msg, "initial")
}

func TestRepo_GetChangedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	initGitRepo(t, dir)

	// Create, add, and commit a tracked file.
	fp := filepath.Join(dir, "tracked.txt")

	err := os.WriteFile(
		fp, []byte("v1\n"), 0o600,
	)
	require.NoError(t, err)

	gitCmd(t, dir, "add", "tracked.txt")
	gitCmd(
		t, dir, "commit", "-m", "add tracked",
	)

	// Modify the tracked file so it shows as
	// changed.
	err = os.WriteFile(
		fp, []byte("v2\n"), 0o600,
	)
	require.NoError(t, err)

	rp := &git.Repo{
		Dir:        dir,
		RemoteName: "origin",
	}

	changed := rp.GetChangedFiles()
	assert.Contains(t, changed, "tracked.txt")
}

func TestRepo_GetChangedFiles_empty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	initGitRepo(t, dir)

	rp := &git.Repo{
		Dir:        dir,
		RemoteName: "origin",
	}

	changed := rp.GetChangedFiles()
	assert.Empty(t, changed)
}

func TestRepo_Clean(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sub := filepath.Join(dir, "repo")

	err := os.MkdirAll(sub, 0o750)
	require.NoError(t, err)

	rp := &git.Repo{Dir: sub, RemoteName: "origin"}

	err = rp.Clean()
	require.NoError(t, err)

	_, statErr := os.Stat(sub)
	assert.True(t, os.IsNotExist(statErr))
}

// initGitRepo creates a git repository with one
// initial commit. Git hooks are disabled to avoid
// interference from pre-commit hooks.
func initGitRepo(tb testing.TB, dir string) {
	tb.Helper()

	cmds := [][]string{
		{"init", "-b", "main"},
		{
			"config",
			"user.email", "test@test.com",
		},
		{"config", "user.name", "Test"},
		// Disable hooks so pre-commit scanners do
		// not interfere with tests.
		{
			"config", "core.hooksPath",
			"/dev/null",
		},
		{
			"commit", "--allow-empty",
			"-m", "initial",
		},
	}

	for _, args := range cmds {
		gitCmd(tb, dir, args...)
	}
}

// gitCmd runs a git command in the given directory.
func gitCmd(
	tb testing.TB,
	dir string,
	args ...string,
) {
	tb.Helper()

	//nolint:gosec // test helper
	cmd := oe.CommandContext(
		context.Background(), "git", args...,
	)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf(
			"git %v failed: %s: %v",
			args, string(out), err,
		)
	}
}
