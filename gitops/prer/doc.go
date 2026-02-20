// Package prer orchestrates the creation of gitops pull requests. It queries
// Bazel for gitops targets, groups them by deployment train, clones the git
// repository, runs each target, stamps files, commits changes, pushes images
// using a configurable worker pool, and creates PRs via a git.GitProvider.
//
// The main entry point is Run, which accepts a Config struct with all
// parameters for the workflow.
package prer
