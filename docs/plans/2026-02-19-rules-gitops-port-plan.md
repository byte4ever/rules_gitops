# rules_gitops Modern Port — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Port Adobe's rules_gitops to Bazel 8+ with bzlmod, rules_oci, and Go 1.24+.

**Architecture:** Flat module layout with single MODULE.bazel. Go helper binaries (stamper, resolver, templating) built with rules_go. Starlark rules in skylib/ generate Kubernetes manifests via kustomize, with image substitution and stamp variable support. GitOps PR creation CLI supports GitHub, GitLab, and Bitbucket.

**Tech Stack:** Bazel 8+, bzlmod, rules_go, rules_oci, gazelle, Go 1.24+, kustomize v5.x, golangci-lint, testify, goleak

**Reference code:** `inspiration/rules_gitops/` contains the original Adobe codebase being ported.

**Go conventions:** Follow the `golang` and `go-design-patterns` skills strictly. See `docs/plans/2026-02-19-rules-gitops-port-design.md` for full details.

---

## Phase 1: Project Scaffolding

### Task 1.1: Initialize Go module and Bazel module

**Files:**
- Create: `go.mod`
- Create: `MODULE.bazel`
- Create: `BUILD.bazel`
- Create: `.bazelrc`
- Create: `.bazelversion`
- Create: `.gitignore`

**Step 1: Create go.mod**

```go
module github.com/byte4ever/rules_gitops

go 1.24
```

**Step 2: Create .bazelversion**

```
8.1.0
```

**Step 3: Create MODULE.bazel**

```starlark
module(
    name = "rules_gitops",
    version = "0.1.0",
)

bazel_dep(name = "bazel_skylib", version = "1.7.1")
bazel_dep(name = "rules_go", version = "0.50.1")
bazel_dep(name = "gazelle", version = "0.38.0")
bazel_dep(name = "rules_oci", version = "2.0.0")
bazel_dep(name = "rules_pkg", version = "1.0.1")

go_sdk = use_extension("@rules_go//go:extensions.bzl", "go_sdk")
go_sdk.download(version = "1.24.0")

go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")
```

**Step 4: Create BUILD.bazel**

```starlark
load("@gazelle//:def.bzl", "gazelle")
load("@rules_go//go:def.bzl", "TOOLS_NOGO", "nogo")
load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")

# gazelle:prefix github.com/byte4ever/rules_gitops
# gazelle:exclude examples
# gazelle:exclude inspiration
# gazelle:proto disable_global
# gazelle:build_tags darwin,linux

gazelle(
    name = "gazelle",
    command = "fix",
    extra_args = [
        "-build_file_name",
        "BUILD,BUILD.bazel",
    ],
)

buildifier(
    name = "buildifier",
    lint_mode = "warn",
    lint_warnings = [
        "-module-docstring",
        "-function-docstring",
        "-function-docstring-header",
        "-function-docstring-args",
        "-function-docstring-return",
    ],
)

buildifier(
    name = "buildifier-fix",
    lint_mode = "fix",
)

buildifier(
    name = "buildifier-check",
    lint_mode = "warn",
    lint_warnings = [
        "-module-docstring",
        "-function-docstring",
        "-function-docstring-header",
        "-function-docstring-args",
        "-function-docstring-return",
    ],
    mode = "check",
)
```

**Step 5: Create .bazelrc**

```
build --nolegacy_external_runfiles
build --verbose_failures

test --test_output=errors
```

**Step 6: Create .gitignore**

```
bazel-*
/vendor/
```

**Step 7: Verify build**

Run: `bazel build //...`
Expected: SUCCESS (empty build, no targets yet)

**Step 8: Commit**

```bash
git add go.mod MODULE.bazel BUILD.bazel .bazelrc .bazelversion .gitignore
git commit -m "feat: initialize project scaffolding with Bazel 8 bzlmod"
```

---

### Task 1.2: Configure golangci-lint

**Files:**
- Create: `.golangci.yml`

**Step 1: Create .golangci.yml**

Configure per the `golang` skill: blocked packages, decorder, errcheck, wrapcheck, varnamelen, nilnil, noctx, gochecknoglobals, gosec, errname, golines (80 char), comment density.

```yaml
run:
  timeout: 5m

linters:
  enable:
    - decorder
    - errcheck
    - errname
    - gochecknoglobals
    - gocritic
    - gofumpt
    - gosec
    - nilnil
    - noctx
    - paralleltest
    - testpackage
    - varnamelen
    - wrapcheck

linters-settings:
  decorder:
    dec-order:
      - type
      - const
      - var
      - func
    disable-dec-order-check: false
  varnamelen:
    min-name-length: 2
    ignore-names:
      - i
      - j
      - k
      - n
      - ok
      - t
      - w
      - r
    ignore-decls:
      - w http.ResponseWriter
      - r *http.Request
      - t *testing.T
      - b *testing.B
  gosec:
    excludes:
      - G101
  govet:
    enable-all: true
  gochecknoglobals:
    allow:
      - Err*

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - wrapcheck
        - varnamelen
        - gochecknoglobals
```

**Step 2: Verify lint passes (no Go files yet, should be clean)**

Run: `golangci-lint run ./...`
Expected: No issues (no Go source files yet)

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "feat: configure golangci-lint with project coding standards"
```

---

## Phase 2: Utility Go Packages

Port the small, dependency-free Go packages first to establish patterns.

### Task 2.1: Port commitmsg package

**Files:**
- Create: `gitops/commitmsg/commitmsg.go`
- Create: `gitops/commitmsg/commitmsg_test.go`

**Reference:** `inspiration/rules_gitops/gitops/commitmsg/commitmsg.go`

**Step 1: Write the failing test**

```go
package commitmsg_test

import (
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/commitmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_produces_markers(t *testing.T) {
	t.Parallel()

	targets := []string{"//app:deploy.gitops", "//svc:deploy.gitops"}
	msg := commitmsg.Generate(targets)

	assert.Contains(t, msg, "--- gitops targets begin ---")
	assert.Contains(t, msg, "--- gitops targets end ---")
	assert.Contains(t, msg, "//app:deploy.gitops")
	assert.Contains(t, msg, "//svc:deploy.gitops")
}

func TestExtractTargets_roundtrip(t *testing.T) {
	t.Parallel()

	targets := []string{"target1", "target2"}
	msg := commitmsg.Generate(targets)
	got := commitmsg.ExtractTargets(msg)

	require.Equal(t, targets, got)
}

func TestExtractTargets_no_markers(t *testing.T) {
	t.Parallel()

	got := commitmsg.ExtractTargets("just a regular commit message")

	assert.Empty(t, got)
}

func TestExtractTargets_missing_end_marker(t *testing.T) {
	t.Parallel()

	msg := "--- gitops targets begin ---\ntarget1\n"
	got := commitmsg.ExtractTargets(msg)

	assert.Empty(t, got)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./gitops/commitmsg/... -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write implementation**

```go
// Package commitmsg generates and parses gitops target lists
// embedded in git commit messages.
package commitmsg

import (
	"log"
	"strings"
)

const (
	begin = "--- gitops targets begin ---"
	end   = "--- gitops targets end ---"
)

// ExtractTargets extracts the list of gitops targets from
// a commit message delimited by begin/end markers.
func ExtractTargets(msg string) []string {
	var targets []string

	betweenMarkers := false

	for _, s := range strings.Split(msg, "\n") {
		switch s {
		case begin:
			betweenMarkers = true
		case end:
			betweenMarkers = false
		default:
			if betweenMarkers {
				targets = append(targets, s)
			}
		}
	}

	if betweenMarkers {
		log.Print("unable to find end marker in commit message")

		return nil
	}

	return targets
}

// Generate produces a commit message section containing the
// given list of gitops targets between begin/end markers.
func Generate(targets []string) string {
	var sb strings.Builder

	sb.WriteByte('\n')
	sb.WriteString(begin)
	sb.WriteByte('\n')

	for _, t := range targets {
		sb.WriteString(t)
		sb.WriteByte('\n')
	}

	sb.WriteString(end)
	sb.WriteByte('\n')

	return sb.String()
}
```

**Step 4: Run tests and lint**

Run: `go test ./gitops/commitmsg/... -v && golangci-lint run ./gitops/commitmsg/...`
Expected: PASS, no lint issues

**Step 5: Run gazelle to generate BUILD files**

Run: `bazel run //:gazelle`

**Step 6: Run bazel test**

Run: `bazel test //gitops/commitmsg:...`
Expected: PASS

**Step 7: Commit**

```bash
git add gitops/commitmsg/
git commit -m "feat: port commitmsg package for gitops target extraction"
```

---

### Task 2.2: Port digester package

**Files:**
- Create: `gitops/digester/digester.go`
- Create: `gitops/digester/digester_test.go`

**Reference:** `inspiration/rules_gitops/gitops/digester/digester.go`

**Step 1: Write the failing test**

```go
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
	p := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(p, []byte("hello"), 0o644))

	got, err := digester.CalculateDigest(p)

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
	p := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(p, []byte("content"), 0o644))

	require.NoError(t, digester.SaveDigest(p))

	got, err := digester.GetDigest(p)
	require.NoError(t, err)

	expected, err := digester.CalculateDigest(p)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestVerifyDigest_valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(p, []byte("content"), 0o644))
	require.NoError(t, digester.SaveDigest(p))

	ok, err := digester.VerifyDigest(p)

	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyDigest_tampered(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(p, []byte("content"), 0o644))
	require.NoError(t, digester.SaveDigest(p))

	// tamper with file
	require.NoError(t, os.WriteFile(p, []byte("tampered"), 0o644))

	ok, err := digester.VerifyDigest(p)

	require.NoError(t, err)
	assert.False(t, ok)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./gitops/digester/... -v`
Expected: FAIL

**Step 3: Write implementation**

Port from inspiration but fix error handling (original uses `log.Fatal`, we return errors per the `golang` skill).

```go
// Package digester calculates and verifies SHA256 file digests.
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
func CalculateDigest(path string) (string, error) {
	const errCtx = "calculating digest"

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	fi, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errCtx, err)
	}

	defer fi.Close()

	h := sha256.New()

	if _, err := io.Copy(h, fi); err != nil {
		return "", fmt.Errorf("%s: %w", errCtx, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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

	digest, err := os.ReadFile(dp)
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

	if err := os.WriteFile(dp, []byte(digest), 0o666); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}
```

**Step 4: Run tests and lint**

Run: `go test ./gitops/digester/... -v && golangci-lint run ./gitops/digester/...`
Expected: PASS

**Step 5: Add fuzz test**

```go
func FuzzCalculateDigest(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Parallel()

		dir := t.TempDir()
		p := filepath.Join(dir, "fuzz.bin")
		require.NoError(t, os.WriteFile(p, data, 0o644))

		d, err := digester.CalculateDigest(p)

		require.NoError(t, err)
		assert.Len(t, d, 64) // sha256 hex is always 64 chars
	})
}
```

**Step 6: Run fuzz test briefly**

Run: `go test ./gitops/digester/... -fuzz=FuzzCalculateDigest -fuzztime=10s`
Expected: PASS

**Step 7: Run gazelle and bazel test**

Run: `bazel run //:gazelle && bazel test //gitops/digester:...`
Expected: PASS

**Step 8: Commit**

```bash
git add gitops/digester/
git commit -m "feat: port digester package with proper error handling"
```

---

### Task 2.3: Port exec package

**Files:**
- Create: `gitops/exec/exec.go`
- Create: `gitops/exec/exec_test.go`

**Reference:** `inspiration/rules_gitops/gitops/exec/exec.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./gitops/exec/... -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// Package exec provides shell command execution helpers.
package exec

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// Ex executes the named command in the given directory and
// returns combined stdout+stderr output. Pass empty dir to
// use the current working directory.
func Ex(
	dir string,
	name string,
	arg ...string,
) (string, error) {
	const errCtx = "executing command"

	slog.Info(
		"executing",
		"cmd", name,
		"args", strings.Join(arg, " "),
	)

	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}

	b, err := cmd.CombinedOutput()

	slog.Info("output", "result", string(b))

	if err != nil {
		return string(b), fmt.Errorf(
			"%s: %s %s: %w",
			errCtx, name, strings.Join(arg, " "), err,
		)
	}

	return string(b), nil
}

// MustEx executes the command and panics on failure.
func MustEx(dir string, name string, arg ...string) {
	if _, err := Ex(dir, name, arg...); err != nil {
		panic(fmt.Sprintf("command failed: %v", err))
	}
}
```

**Step 4: Run tests and lint**

Run: `go test ./gitops/exec/... -v && golangci-lint run ./gitops/exec/...`
Expected: PASS

**Step 5: Run gazelle and bazel test**

Run: `bazel run //:gazelle && bazel test //gitops/exec:...`
Expected: PASS

**Step 6: Commit**

```bash
git add gitops/exec/
git commit -m "feat: port exec package with slog logging"
```

---

### Task 2.4: Port bazel target utility package

**Files:**
- Create: `gitops/bazel/bazeltargets.go`
- Create: `gitops/bazel/bazeltargets_test.go`

**Reference:** `inspiration/rules_gitops/gitops/bazel/bazeltargets.go`

**Step 1: Write the failing test**

```go
package bazel_test

import (
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/bazel"
	"github.com/stretchr/testify/assert"
)

func TestTargetToExecutable_full_target(t *testing.T) {
	t.Parallel()

	got := bazel.TargetToExecutable(
		"//app/deploy:deploy-prod.gitops",
	)

	assert.Equal(
		t,
		"bazel-bin/app/deploy/deploy-prod.gitops",
		got,
	)
}

func TestTargetToExecutable_no_prefix(t *testing.T) {
	t.Parallel()

	got := bazel.TargetToExecutable("some/path")

	assert.Equal(t, "some/path", got)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./gitops/bazel/... -v`
Expected: FAIL

**Step 3: Write implementation**

```go
// Package bazel provides utilities for working with Bazel
// target labels and paths.
package bazel

import "strings"

// TargetToExecutable converts a Bazel target label like
// //pkg:name to the corresponding bazel-bin executable
// path. Non-label inputs are returned unchanged.
func TargetToExecutable(target string) string {
	if !strings.HasPrefix(target, "//") {
		return target
	}

	t := "bazel-bin/" + target[2:]
	t = strings.Replace(t, ":", "/", 1)

	return t
}
```

**Step 4: Run tests and lint**

Run: `go test ./gitops/bazel/... -v && golangci-lint run ./gitops/bazel/...`
Expected: PASS

**Step 5: Run gazelle and bazel test**

Run: `bazel run //:gazelle && bazel test //gitops/bazel:...`
Expected: PASS

**Step 6: Commit**

```bash
git add gitops/bazel/
git commit -m "feat: port bazel target-to-executable utility"
```

---

## Phase 3: Template Engine

### Task 3.1: Evaluate fasttemplate dependency

**Decision point:** The original vendors `valyala/fasttemplate`. Check if it's available as a Go module dependency.

**Step 1: Check if fasttemplate is maintained**

Run: `go list -m -versions github.com/valyala/fasttemplate`

If available, add it to go.mod. If not, find an alternative or inline the minimal subset needed (the library is ~200 lines).

**Step 2: Add dependency**

Run: `go get github.com/valyala/fasttemplate@latest`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add fasttemplate dependency"
```

---

### Task 3.2: Port templating binary

**Files:**
- Create: `templating/main.go`
- Create: `templating/main_test.go`

**Reference:** `inspiration/rules_gitops/templating/main.go`

The templating binary reads stamp info files and variables, then expands a template file. Key flags: `--stamp_info_file`, `--variable`, `--imports`, `--template`, `--output`, `--start_tag`, `--end_tag`, `--executable`.

**Step 1: Write the failing test**

Test the core template expansion logic as an exported function (not just main). Create a `templating` package with an `Engine` type.

```go
package templating_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/byte4ever/rules_gitops/templating"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Expand_variables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tpl := filepath.Join(dir, "input.tpl")
	out := filepath.Join(dir, "output.txt")

	require.NoError(t, os.WriteFile(
		tpl,
		[]byte("hello {{NAME}}"),
		0o644,
	))

	e := templating.Engine{
		StartTag: "{{",
		EndTag:   "}}",
	}

	err := e.Expand(tpl, out, map[string]string{
		"NAME": "world",
	})

	require.NoError(t, err)

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(got))
}

func TestEngine_Expand_stamps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// stamp file
	sf := filepath.Join(dir, "stamp.txt")
	require.NoError(t, os.WriteFile(
		sf,
		[]byte("BUILD_USER testuser\nGIT_SHA abc123\n"),
		0o644,
	))

	tpl := filepath.Join(dir, "input.tpl")
	out := filepath.Join(dir, "output.txt")

	require.NoError(t, os.WriteFile(
		tpl,
		[]byte("user={{BUILD_USER}} sha={{GIT_SHA}}"),
		0o644,
	))

	e := templating.Engine{
		StartTag:       "{{",
		EndTag:         "}}",
		StampInfoFiles: []string{sf},
	}

	err := e.Expand(tpl, out, nil)
	require.NoError(t, err)

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "user=testuser sha=abc123", string(got))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./templating/... -v`
Expected: FAIL

**Step 3: Write implementation**

Create `templating/engine.go` with the `Engine` type and `Expand` method. Create `templating/main.go` as the CLI entry point that parses flags and calls the engine.

**Step 4: Add fuzz test**

Fuzz the template engine with arbitrary template strings and variable maps.

**Step 5: Run tests and lint**

Run: `go test ./templating/... -v && golangci-lint run ./templating/...`
Expected: PASS

**Step 6: Run gazelle and bazel test**

Run: `bazel run //:gazelle && bazel test //templating:...`
Expected: PASS

**Step 7: Commit**

```bash
git add templating/
git commit -m "feat: port template engine with Engine type"
```

---

### Task 3.3: Port stamper binary

**Files:**
- Create: `stamper/main.go`
- Create: `stamper/stamper.go`
- Create: `stamper/stamper_test.go`

**Reference:** `inspiration/rules_gitops/stamper/main.go`

The stamper reads Bazel workspace status files and substitutes `{VAR}` (single-brace) placeholders. Key flags: `--stamp-info-file`, `--format`, `--format-file`, `--output`.

**Step 1: Write the failing test**

```go
package stamper_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/byte4ever/rules_gitops/stamper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStamp_substitutes_variables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := filepath.Join(dir, "status.txt")
	require.NoError(t, os.WriteFile(
		sf,
		[]byte("BUILD_USER alice\nGIT_SHA deadbeef\n"),
		0o644,
	))

	got, err := stamper.Stamp(
		[]string{sf},
		"deployed by {BUILD_USER} at {GIT_SHA}",
	)

	require.NoError(t, err)
	assert.Equal(t, "deployed by alice at deadbeef", got)
}

func TestStamp_missing_variable_preserved(t *testing.T) {
	t.Parallel()

	got, err := stamper.Stamp(nil, "no {SUCH_VAR} here")

	require.NoError(t, err)
	assert.Equal(t, "no {SUCH_VAR} here", got)
}
```

**Step 2-6:** Same TDD cycle as prior tasks.

**Step 7: Add fuzz test for stamper**

Fuzz with arbitrary status file content and format strings.

**Step 8: Commit**

```bash
git add stamper/
git commit -m "feat: port stamper binary for workspace status substitution"
```

---

## Phase 4: Resolver Binary

### Task 4.1: Port resolver package

**Files:**
- Create: `resolver/pkg/resolver.go`
- Create: `resolver/pkg/resolver_test.go`
- Create: `resolver/main.go`

**Reference:** `inspiration/rules_gitops/resolver/pkg/resolver.go`

The resolver walks YAML documents looking for container image references and substitutes them with registry URLs. Uses `goccy/go-yaml` (not `ghodss/yaml` per blocked packages) and `k8s.io/apimachinery`.

**Step 1: Write the failing test**

Test with a minimal Deployment YAML containing an image reference that should be substituted.

**Step 2: Write implementation**

Port the resolver logic, replacing `github.com/ghodss/yaml` with `github.com/goccy/go-yaml`.

**Step 3: Add fuzz test**

Fuzz the YAML parsing with arbitrary inputs to ensure no panics.

**Step 4-7:** TDD cycle, gazelle, bazel test, commit.

```bash
git add resolver/
git commit -m "feat: port resolver binary with OCI-aware image substitution"
```

---

## Phase 5: Starlark Rules Foundation

### Task 5.1: Create K8sPushInfo provider and push.bzl skeleton

**Files:**
- Create: `skylib/push.bzl`

**Reference:** `inspiration/rules_gitops/skylib/push.bzl`

Port the `K8sPushInfo` provider definition and `k8s_container_push` rule, replacing `rules_docker` references with `rules_oci`.

**Step 1: Create push.bzl with provider and rule skeleton**

The `K8sPushInfo` provider fields stay identical. The rule attrs change: `_digester` and `_pusher` from `@io_bazel_rules_docker` become `@rules_oci` equivalents.

**Step 2: Commit**

```bash
git add skylib/push.bzl
git commit -m "feat: define K8sPushInfo provider and push rule for rules_oci"
```

---

### Task 5.2: Port stamp.bzl

**Files:**
- Create: `skylib/stamp.bzl`

**Reference:** `inspiration/rules_gitops/skylib/stamp.bzl`

Replace `load("@io_bazel_rules_docker//skylib:path.bzl", "runfile")` with an inline `_runfile_path` implementation.

**Step 1: Create stamp.bzl**

Inline the `runfile` function:
```starlark
def _runfile_path(ctx, f):
    if f.owner and f.owner.workspace_name:
        return f.owner.workspace_name + "/" + f.short_path
    return ctx.workspace_name + "/" + f.short_path
```

Port `stamp`, `stamp_value`, and `more_stable_status` rules.

**Step 2: Commit**

```bash
git add skylib/stamp.bzl
git commit -m "feat: port stamp.bzl with inline runfile path helper"
```

---

### Task 5.3: Port external_image.bzl

**Files:**
- Create: `skylib/external_image.bzl`

**Reference:** `inspiration/rules_gitops/skylib/external_image.bzl`

Straightforward port — only depends on `K8sPushInfo` from `push.bzl`.

**Step 1: Create external_image.bzl**

**Step 2: Commit**

```bash
git add skylib/external_image.bzl
git commit -m "feat: port external_image rule"
```

---

### Task 5.4: Port templates.bzl

**Files:**
- Create: `skylib/templates.bzl`

**Reference:** `inspiration/rules_gitops/skylib/templates.bzl`

Port `expand_template` and `merge_files` rules. Update label references from `@com_adobe_rules_gitops//` to `@rules_gitops//`.

**Step 1: Create templates.bzl**

**Step 2: Commit**

```bash
git add skylib/templates.bzl
git commit -m "feat: port expand_template and merge_files rules"
```

---

### Task 5.5: Port shell templates

**Files:**
- Create: `skylib/cmd.sh.tpl`
- Create: `skylib/k8s_cmd.sh.tpl`
- Create: `skylib/k8s_gitops.sh.tpl`
- Create: `skylib/k8s_test_namespace.sh.tpl`
- Create: `skylib/push-tag.sh.tpl`
- Create: `skylib/kustomize/kubectl.sh.tpl`
- Create: `skylib/kustomize/run-all.sh.tpl`

**Reference:** `inspiration/rules_gitops/skylib/*.sh.tpl`

Direct copy — these are bash templates with `%{variable}` placeholders filled by Starlark rules. No code changes needed.

**Step 1: Copy all shell templates**

**Step 2: Commit**

```bash
git add skylib/*.sh.tpl skylib/kustomize/*.sh.tpl
git commit -m "feat: port shell script templates"
```

---

## Phase 6: Kustomize Module Extension

### Task 6.1: Create kustomize module extension

**Files:**
- Create: `skylib/kustomize/extensions.bzl`
- Update: `MODULE.bazel`

**Reference:** `inspiration/rules_gitops/skylib/kustomize/kustomize.bzl` (the `kustomize_setup` function)

**Step 1: Create the module extension**

The extension downloads a platform-specific kustomize v5.x binary. Upgrade the URL patterns and SHA256 hashes. Add macOS arm64 support.

```starlark
_KUSTOMIZE_VERSION = "5.4.3"

_BINARIES = {
    "darwin_amd64": ("kustomize_v{v}_darwin_amd64.tar.gz", "<sha256>"),
    "darwin_arm64": ("kustomize_v{v}_darwin_arm64.tar.gz", "<sha256>"),
    "linux_amd64": ("kustomize_v{v}_linux_amd64.tar.gz", "<sha256>"),
    "linux_arm64": ("kustomize_v{v}_linux_arm64.tar.gz", "<sha256>"),
}
```

**Step 2: Register in MODULE.bazel**

```starlark
gitops = use_extension("//skylib/kustomize:extensions.bzl", "gitops")
use_repo(gitops, "kustomize_bin")
```

**Step 3: Verify kustomize downloads**

Run: `bazel build @kustomize_bin//:kustomize`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add skylib/kustomize/extensions.bzl MODULE.bazel
git commit -m "feat: add kustomize module extension for Bazel 8 bzlmod"
```

---

### Task 6.2: Port kustomize.bzl core rule

**Files:**
- Create: `skylib/kustomize/kustomize.bzl`

**Reference:** `inspiration/rules_gitops/skylib/kustomize/kustomize.bzl`

Port the `kustomize`, `kubectl`, `gitops`, and `kustomize_setup` rules. Replace all `@com_adobe_rules_gitops//` labels with `@rules_gitops//`. Replace `@io_bazel_rules_docker` references.

**Step 1: Create kustomize.bzl** with `KustomizeInfo` provider, `kustomize` rule, `kubectl` rule, and `gitops` rule.

**Step 2: Commit**

```bash
git add skylib/kustomize/kustomize.bzl
git commit -m "feat: port kustomize rule with provider and kubectl/gitops targets"
```

---

## Phase 7: Core k8s_deploy Macro

### Task 7.1: Port k8s.bzl

**Files:**
- Create: `skylib/k8s.bzl`

**Reference:** `inspiration/rules_gitops/skylib/k8s.bzl`

This is the central macro. Port `k8s_deploy`, `k8s_test_setup`, `kubeconfig`, and `k8s_test_namespace`. Update all internal label references.

**Step 1: Create k8s.bzl**

Port the full macro, updating:
- All `@com_adobe_rules_gitops//` → `@rules_gitops//`
- All `@io_bazel_rules_docker//` references → `@rules_oci//` equivalents
- The `workspace_binary` references

**Step 2: Create run_in_workspace.bzl**

Port from `inspiration/rules_gitops/skylib/run_in_workspace.bzl`. Update `root_file` default from `//:WORKSPACE` to `//:MODULE.bazel`.

**Step 3: Create public API entry point**

Create `gitops/defs.bzl`:
```starlark
load("@rules_gitops//skylib:external_image.bzl", _external_image = "external_image")
load("@rules_gitops//skylib:k8s.bzl", _k8s_deploy = "k8s_deploy", _k8s_test_setup = "k8s_test_setup")

k8s_deploy = _k8s_deploy
k8s_test_setup = _k8s_test_setup
external_image = _external_image
```

**Step 4: Commit**

```bash
git add skylib/k8s.bzl skylib/run_in_workspace.bzl gitops/defs.bzl
git commit -m "feat: port k8s_deploy macro and public API entry point"
```

---

### Task 7.2: Add Starlark rule tests

**Files:**
- Create: `skylib/kustomize/tests/BUILD`
- Create: `skylib/kustomize/tests/*.bzl` test files
- Create: `skylib/kustomize/tests/testdata/` golden files

**Reference:** `inspiration/rules_gitops/skylib/kustomize/tests/`

Port the skylib `analysistest` and `unittest` tests. Verify `k8s_deploy` generates the correct targets, kustomize output matches golden files, etc.

**Step 1: Port test BUILD and test bzl files**

**Step 2: Verify tests pass**

Run: `bazel test //skylib/kustomize/tests/...`
Expected: PASS

**Step 3: Commit**

```bash
git add skylib/kustomize/tests/
git commit -m "feat: port Starlark rule tests for kustomize"
```

---

## Phase 8: Testing Infrastructure

### Task 8.1: Port it_manifest_filter binary

**Files:**
- Create: `testing/it_manifest_filter/main.go`
- Create: `testing/it_manifest_filter/pkg/filter.go`
- Create: `testing/it_manifest_filter/pkg/filter_test.go`
- Create: `testing/it_manifest_filter/pkg/testdata/*.yaml`

**Reference:** `inspiration/rules_gitops/testing/it_manifest_filter/`

Replace `github.com/ghodss/yaml` with `github.com/goccy/go-yaml`.

**Step 1-7:** TDD cycle with golden file tests.

```bash
git add testing/it_manifest_filter/
git commit -m "feat: port it_manifest_filter for integration test setup"
```

---

### Task 8.2: Port it_sidecar binary

**Files:**
- Create: `testing/it_sidecar/it_sidecar.go`
- Create: `testing/it_sidecar/stern/` (all files)
- Create: `testing/it_sidecar/client/sidecar_client.go`

**Reference:** `inspiration/rules_gitops/testing/it_sidecar/`

Larger binary — manages Kubernetes namespace lifecycle, pod watching, log tailing, port forwarding.

**Step 1-7:** Port with tests.

```bash
git add testing/it_sidecar/
git commit -m "feat: port it_sidecar for integration test lifecycle"
```

---

## Phase 9: Git Backends and create_gitops_prs CLI

### Task 9.1: Define GitProvider interface

**Files:**
- Create: `gitops/git/provider.go`
- Create: `gitops/git/provider_test.go`

```go
// Pattern: Strategy — swap git platform without changing
// PR creation logic.

// GitProvider creates pull requests on a git hosting
// platform.
type GitProvider interface {
	CreatePR(
		ctx context.Context,
		from string,
		to string,
		title string,
		body string,
	) error
}
```

**Step 1-4:** TDD cycle.

```bash
git add gitops/git/provider.go gitops/git/provider_test.go
git commit -m "feat: define GitProvider strategy interface"
```

---

### Task 9.2: Port git repo operations

**Files:**
- Create: `gitops/git/repo.go`
- Create: `gitops/git/repo_test.go`

**Reference:** `inspiration/rules_gitops/gitops/git/git.go`

Port `Clone`, `Fetch`, `SwitchToBranch`, `RecreateBranch`, `Commit`, `Push`, etc.

**Step 1-6:** TDD cycle.

```bash
git add gitops/git/repo.go gitops/git/repo_test.go
git commit -m "feat: port git repo operations"
```

---

### Task 9.3: Port GitHub backend

**Files:**
- Create: `gitops/git/github/github.go`
- Create: `gitops/git/github/github_test.go`

**Reference:** `inspiration/rules_gitops/gitops/git/github/github.go`

Upgrade `google/go-github` from v32 to latest. Implement `GitProvider` interface.

**Step 1-6:** TDD cycle.

```bash
git add gitops/git/github/
git commit -m "feat: port GitHub GitProvider with updated client"
```

---

### Task 9.4: Port GitLab backend

**Files:**
- Create: `gitops/git/gitlab/gitlab.go`
- Create: `gitops/git/gitlab/gitlab_test.go`

**Reference:** `inspiration/rules_gitops/gitops/git/gitlab/gitlab.go`

Upgrade `xanzy/go-gitlab` to latest. Implement `GitProvider` interface.

**Step 1-6:** TDD cycle.

```bash
git add gitops/git/gitlab/
git commit -m "feat: port GitLab GitProvider with updated client"
```

---

### Task 9.5: Port Bitbucket backend

**Files:**
- Create: `gitops/git/bitbucket/bitbucket.go`
- Create: `gitops/git/bitbucket/bitbucket_test.go`

**Reference:** `inspiration/rules_gitops/gitops/git/bitbucket/bitbucket.go`

Pure HTTP client. Implement `GitProvider` interface.

**Step 1-6:** TDD cycle.

```bash
git add gitops/git/bitbucket/
git commit -m "feat: port Bitbucket GitProvider"
```

---

### Task 9.6: Port create_gitops_prs CLI

**Files:**
- Create: `gitops/prer/create_gitops_prs.go`
- Create: `gitops/prer/create_gitops_prs_test.go`

**Reference:** `inspiration/rules_gitops/gitops/prer/create_gitops_prs.go`

Port the main CLI. Use config struct pattern (>4 args). Inject `GitProvider` via factory based on `--git_server` flag.

```go
// Pattern: Factory — select git platform implementation
// based on --git_server flag value.
func NewGitProvider(server string) (git.GitProvider, error) {
```

**Step 1-6:** TDD cycle.

```bash
git add gitops/prer/
git commit -m "feat: port create_gitops_prs CLI with Strategy/Factory patterns"
```

---

## Phase 10: Examples

### Task 10.1: Create helloworld example

**Files:**
- Create: `examples/MODULE.bazel`
- Create: `examples/.bazelrc`
- Create: `examples/helloworld/BUILD`
- Create: `examples/helloworld/helloworld.go`
- Create: `examples/helloworld/deployment.yaml`
- Create: `examples/helloworld/service.yaml`

**Reference:** `inspiration/rules_gitops/examples/`

Update the example's `MODULE.bazel` to use `local_path_override` to reference the parent module.

**Step 1: Create example files**

**Step 2: Verify build**

Run: `cd examples && bazel build //...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add examples/
git commit -m "feat: add helloworld example with bzlmod local override"
```

---

## Phase 11: E2E Tests

### Task 11.1: Port e2e test scripts

**Files:**
- Create: `e2e_test.sh`
- Create: `create_kind_cluster.sh`
- Create: `examples/e2e-test.sh`

**Reference:** `inspiration/rules_gitops/e2e_test.sh`, `create_kind_cluster.sh`

**Step 1: Port scripts**, updating kind version and registry setup.

**Step 2: Verify e2e (requires kind)**

Run: `./e2e_test.sh`
Expected: PASS

**Step 3: Commit**

```bash
git add e2e_test.sh create_kind_cluster.sh examples/e2e-test.sh
git commit -m "feat: port e2e test scripts with updated kind version"
```

---

## Phase 12: CI Pipeline

### Task 12.1: Create GitHub Actions workflow

**Files:**
- Create: `.github/workflows/ci.yaml`

**Reference:** `inspiration/rules_gitops/.github/workflows/ci.yaml`

Five jobs: `lint-go`, `test-go` (coverage + fuzz), `test-starlark`, `buildifier`, `e2e`.

Single Bazel version (8.x). Add coverage threshold check (80%). Add fuzz step with 30s budget.

**Step 1: Create ci.yaml**

**Step 2: Commit**

```bash
git add .github/workflows/ci.yaml
git commit -m "feat: add GitHub Actions CI with coverage and fuzz testing"
```

---

## Phase 13: CLAUDE.md

### Task 13.1: Create CLAUDE.md

**Files:**
- Create: `CLAUDE.md`

Document the project's build commands, architecture, and conventions based on everything built in prior phases.

**Step 1: Write CLAUDE.md**

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md for Claude Code guidance"
```

---

## Dependency Graph

```
Phase 1 (scaffolding)
  └─ Phase 2 (utility packages) ──┐
  └─ Phase 3 (template + stamper) ├─ Phase 5 (Starlark foundation)
  └─ Phase 4 (resolver) ──────────┘       │
                                    Phase 6 (kustomize extension)
                                           │
                                    Phase 7 (k8s_deploy macro)
                                       │         │
                                Phase 8 (testing) │
                                       │         │
                                Phase 9 (git + CLI)
                                       │
                                Phase 10 (examples)
                                       │
                                Phase 11 (e2e)
                                       │
                                Phase 12 (CI)
                                       │
                                Phase 13 (CLAUDE.md)
```

Phases 2, 3, and 4 can run in parallel after Phase 1.
