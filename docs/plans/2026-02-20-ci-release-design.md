# CI and Release Process Design

## Context

The project has a working CI workflow (`ci.yaml`) with lint, Go tests (80% coverage + fuzz), buildifier, Bazel build/test, and e2e. No release process exists — no tags, no release workflow, no changelog.

There are version mismatches: `.bazelversion` says 8.1.0 but CI hardcodes 8.0.0; `go.mod` says 1.25.0 but MODULE.bazel downloads 1.24.0.

## Approach: Simple Tag-Triggered Release

### Release Workflow

New `.github/workflows/release.yaml` triggered on `*.*.*` tag push (no `v` prefix).

Steps:
1. Checkout code
2. Validate tag matches MODULE.bazel version (fail if mismatch)
3. Run full CI gate
4. Create GitHub Release with auto-generated release notes

### CI Fixes

1. Bazel version: stop hardcoding `USE_BAZEL_VERSION: "8.0.0"` in CI. Read from `.bazelversion` file instead.
2. Go version: align MODULE.bazel Go SDK download with go.mod version.

### Release Process (human steps)

1. Update MODULE.bazel version to X.Y.Z
2. Commit: `release: prepare X.Y.Z`
3. Tag: `git tag X.Y.Z`
4. Push: `git push && git push --tags`
5. Release workflow creates GitHub Release automatically

### Tag format

`X.Y.Z` (no `v` prefix). Semver.

## Deliverables

1. `.github/workflows/release.yaml` — tag-triggered release workflow
2. `.github/workflows/ci.yaml` — fix Bazel/Go version alignment
3. `MODULE.bazel` — align Go SDK version with go.mod
