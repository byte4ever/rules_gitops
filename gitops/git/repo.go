package git

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	oe "os/exec"
	"path/filepath"
	"strings"

	"github.com/byte4ever/rules_gitops/gitops/exec"
)

// Repo is a local clone of a git repository. Create
// with Clone, and call Clean when done.
type Repo struct {
	// Dir is the filesystem location of the clone.
	Dir string
	// RemoteName is the name of the upstream remote.
	RemoteName string
}

// Clone clones a repository into dir. Pass the full
// repository URL as repo (e.g.
// "https://github.com/org/repo.git"). mirrorDir is an
// optional local mirror used as a reference clone. When
// gitopsPath is non-root only that subtree is checked
// out via sparse-checkout.
//
//nolint:gosec // file paths originate from CLI flags
func Clone(
	repo string,
	dir string,
	mirrorDir string,
	primaryBranch string,
	gitopsPath string,
) (*Repo, error) {
	const errCtx = "cloning repository"

	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf(
			"%s: remove dir: %w", errCtx, err,
		)
	}

	remoteName := "origin"

	args := []string{
		"clone",
		"--no-checkout",
		"--single-branch",
		"--branch", primaryBranch,
		"--filter=blob:none",
		"--no-tags",
		"--origin", remoteName,
	}

	if mirrorDir != "" {
		args = append(args, "--reference", mirrorDir)
	}

	args = append(args, repo, dir)
	exec.MustEx("", "git", args...)

	// Enable sparse-checkout when restricting to a
	// subdirectory.
	if !isRootPath(gitopsPath) {
		exec.MustEx(
			dir, "git",
			"config", "--local",
			"core.sparsecheckout", "true",
		)

		genPath := fmt.Sprintf("%s/\n", gitopsPath)
		sparsePath := filepath.Join(
			dir, ".git", "info", "sparse-checkout",
		)

		//nolint:gosec // mode 0644 is intentional
		err := os.WriteFile(
			sparsePath,
			[]byte(genPath),
			0o644,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: write sparse-checkout: %w",
				errCtx, err,
			)
		}
	}

	exec.MustEx(dir, "git", "checkout", primaryBranch)

	return &Repo{
		Dir:        dir,
		RemoteName: remoteName,
	}, nil
}

// Clean removes the local clone directory.
func (r *Repo) Clean() error {
	const errCtx = "cleaning repository"

	if err := os.RemoveAll(r.Dir); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}

// Fetch adds the given pattern to tracked remote
// branches and fetches them.
func (r *Repo) Fetch(pattern string) {
	exec.MustEx(
		r.Dir, "git",
		"remote", "set-branches", "--add",
		r.RemoteName, pattern,
	)
	exec.MustEx(
		r.Dir, "git",
		"fetch", "--force",
		"--filter=blob:none", "--no-tags",
		r.RemoteName,
	)
}

// SwitchToBranch switches to branch, creating it from
// primaryBranch if it does not exist. Returns true when
// the branch was newly created.
func (r *Repo) SwitchToBranch(
	branch string,
	primaryBranch string,
) bool {
	if _, err := exec.Ex(
		r.Dir, "git", "checkout", branch,
	); err != nil {
		// Branch does not exist yet: create and
		// check out.
		exec.MustEx(
			r.Dir, "git",
			"branch", branch, primaryBranch,
		)
		exec.MustEx(
			r.Dir, "git", "checkout", branch,
		)

		return true
	}

	return false
}

// RecreateBranch discards the content of branch and
// resets it from primaryBranch.
func (r *Repo) RecreateBranch(
	branch string,
	primaryBranch string,
) {
	exec.MustEx(
		r.Dir, "git", "checkout", primaryBranch,
	)
	exec.MustEx(
		r.Dir, "git",
		"branch", "-f", branch, primaryBranch,
	)
	exec.MustEx(r.Dir, "git", "checkout", branch)
}

// GetLastCommitMessage returns the most recent commit
// message on the current branch. Returns empty string
// on error.
func (r *Repo) GetLastCommitMessage() string {
	msg, err := exec.Ex(
		r.Dir, "git", "log", "-1", "--pretty=%B",
	)
	if err != nil {
		return ""
	}

	return msg
}

// Commit stages all changes under gitopsPath and
// commits them. Returns true when changes were
// committed, false when the tree was clean.
func (r *Repo) Commit(
	message string,
	gitopsPath string,
) bool {
	if isRootPath(gitopsPath) {
		exec.MustEx(r.Dir, "git", "add", ".")
	} else {
		exec.MustEx(r.Dir, "git", "add", gitopsPath)
	}

	if r.IsClean() {
		return false
	}

	exec.MustEx(
		r.Dir, "git", "commit", "-a", "-m", message,
	)

	return true
}

// RestoreFile restores the specified file to its
// last-committed state.
func (r *Repo) RestoreFile(fileName string) {
	exec.MustEx(
		r.Dir, "git", "checkout", "--", fileName,
	)
}

// GetChangedFiles returns file paths that differ from
// the index (unstaged changes).
func (r *Repo) GetChangedFiles() []string {
	out, err := exec.Ex(
		r.Dir, "git", "diff", "--name-only",
	)
	if err != nil {
		slog.Error(
			"failed to get changed files",
			"error", err,
		)

		return nil
	}

	var files []string

	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		files = append(files, sc.Text())
	}

	if err := sc.Err(); err != nil {
		slog.Error(
			"failed to scan changed files",
			"error", err,
		)

		return nil
	}

	return files
}

// IsClean reports whether the working tree has no
// uncommitted changes.
func (r *Repo) IsClean() bool {
	//nolint:gosec // args are constants
	cmd := oe.CommandContext(
		context.Background(),
		"git", "status", "--porcelain",
	)
	cmd.Dir = r.Dir

	by, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(
			"failed to check repo status",
			"error", err,
		)

		return false
	}

	return len(by) == 0
}

// Push force-pushes the given branches to the remote.
// All changes should be committed before calling Push.
func (r *Repo) Push(branches []string) {
	args := append(
		[]string{
			"push", r.RemoteName,
			"-f", "--set-upstream",
		},
		branches...,
	)
	exec.MustEx(r.Dir, "git", args...)
}

// isRootPath reports whether gitopsPath refers to the
// repository root.
func isRootPath(gitopsPath string) bool {
	return gitopsPath == "" || gitopsPath == "."
}
