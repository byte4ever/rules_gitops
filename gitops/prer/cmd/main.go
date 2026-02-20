// Command create_gitops_prs orchestrates the creation
// of gitops deployment pull requests. It queries Bazel
// for gitops targets, runs them, pushes images, and
// creates PRs on the configured git hosting platform.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/byte4ever/rules_gitops/gitops/git"
	"github.com/byte4ever/rules_gitops/gitops/git/bitbucket"
	"github.com/byte4ever/rules_gitops/gitops/git/github"
	"github.com/byte4ever/rules_gitops/gitops/git/gitlab"
	"github.com/byte4ever/rules_gitops/gitops/prer"
)

// sliceFlag implements flag.Value for multi-value
// string flags (repeated --flag=val usage).
type sliceFlag []string

// String returns the flag value as a comma-separated
// string representation.
func (s *sliceFlag) String() string {
	if s == nil {
		return ""
	}

	return strings.Join(*s, ",")
}

// Set appends a value to the slice.
func (s *sliceFlag) Set(val string) error {
	*s = append(*s, val)

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

//nolint:funlen // CLI flag setup is inherently long
func run() error {
	const errCtx = "running create_gitops_prs"

	// Bazel flags.
	bazelCmd := flag.String(
		"bazel_cmd", "bazel",
		"Bazel command name or path",
	)
	workspace := flag.String(
		"workspace", "",
		"Bazel workspace root directory",
	)
	target := flag.String(
		"target", "",
		"Bazel query target pattern",
	)

	// Git repository flags.
	gitRepo := flag.String(
		"git_repo", "",
		"Remote git repository URL",
	)
	gitMirror := flag.String(
		"git_mirror", "",
		"Local git mirror for reference clones",
	)
	gitopsPath := flag.String(
		"gitops_path", "",
		"Subdirectory for sparse checkout",
	)
	tmpDir := flag.String(
		"tmp_dir", os.TempDir(),
		"Temporary directory for clones",
	)

	// Branch flags.
	releaseBranch := flag.String(
		"release_branch", "",
		"Release branch to filter targets by",
	)
	primaryBranch := flag.String(
		"primary_branch", "main",
		"Primary branch name",
	)
	depBranchPrefix := flag.String(
		"deployment_branch_prefix", "deploy/",
		"Prefix for deployment branch names",
	)
	depBranchSuffix := flag.String(
		"deployment_branch_suffix", "",
		"Suffix for deployment branch names",
	)
	branchName := flag.String(
		"branch_name", "",
		"Source branch name for stamp context",
	)
	gitCommit := flag.String(
		"git_commit", "",
		"Source commit SHA for stamp context",
	)

	// Push flags.
	pushParallelism := flag.Int(
		"push_parallelism", 4,
		"Number of concurrent image push workers",
	)

	// Slice flags for rule matching.
	var gitopsKinds sliceFlag

	flag.Var(
		&gitopsKinds,
		"gitops_kind",
		"Rule kind to query (repeatable)",
	)

	var gitopsRuleNames sliceFlag

	flag.Var(
		&gitopsRuleNames,
		"gitops_rule_name",
		"Rule name for push deps query (repeatable)",
	)

	var gitopsRuleAttrs sliceFlag

	flag.Var(
		&gitopsRuleAttrs,
		"gitops_rule_attr",
		"Rule attribute for push deps (repeatable)",
	)

	// PR flags.
	prTitle := flag.String(
		"pr_title", "GitOps deployment",
		"Title for created pull requests",
	)
	prBody := flag.String(
		"pr_body", "",
		"Body for created pull requests",
	)
	dryRun := flag.Bool(
		"dry_run", false,
		"Skip push and PR creation",
	)
	stamp := flag.Bool(
		"stamp", false,
		"Enable file stamping",
	)

	// Git provider selection.
	gitServer := flag.String(
		"git_server", "github",
		"Git hosting platform: github, gitlab, "+
			"or bitbucket",
	)

	// GitHub-specific flags.
	ghRepoOwner := flag.String(
		"github_repo_owner", "",
		"GitHub repository owner",
	)
	ghRepo := flag.String(
		"github_repo", "",
		"GitHub repository name",
	)
	ghToken := flag.String(
		"github_access_token", "",
		"GitHub personal access token",
	)
	ghEnterprise := flag.String(
		"github_enterprise_host", "",
		"GitHub Enterprise hostname",
	)

	// GitLab-specific flags.
	glHost := flag.String(
		"gitlab_host", "",
		"GitLab instance URL",
	)
	glRepo := flag.String(
		"gitlab_repo", "",
		"GitLab project path (org/project)",
	)
	glToken := flag.String(
		"gitlab_access_token", "",
		"GitLab personal access token",
	)

	// Bitbucket-specific flags.
	bbEndpoint := flag.String(
		"bitbucket_api_endpoint", "",
		"Bitbucket Server REST API URL",
	)
	bbUser := flag.String(
		"bitbucket_user", "",
		"Bitbucket API username",
	)
	bbPassword := flag.String(
		"bitbucket_password", "",
		"Bitbucket API password or token",
	)

	flag.Parse()

	// Build git provider from flags.
	provider, err := newGitProvider(
		*gitServer,
		providerFlags{
			ghRepoOwner:  *ghRepoOwner,
			ghRepo:       *ghRepo,
			ghToken:      *ghToken,
			ghEnterprise: *ghEnterprise,
			glHost:       *glHost,
			glRepo:       *glRepo,
			glToken:      *glToken,
			bbEndpoint:   *bbEndpoint,
			bbUser:       *bbUser,
			bbPassword:   *bbPassword,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"%s: create provider: %w", errCtx, err,
		)
	}

	cfg := prer.Config{
		BazelCmd:               *bazelCmd,
		Workspace:              *workspace,
		Target:                 *target,
		GitRepo:                *gitRepo,
		GitMirror:              *gitMirror,
		GitopsPath:             *gitopsPath,
		TmpDir:                 *tmpDir,
		ReleaseBranch:          *releaseBranch,
		PrimaryBranch:          *primaryBranch,
		DeploymentBranchPrefix: *depBranchPrefix,
		DeploymentBranchSuffix: *depBranchSuffix,
		BranchName:             *branchName,
		GitCommit:              *gitCommit,
		PushParallelism:        *pushParallelism,
		GitopsKinds:            gitopsKinds,
		GitopsRuleNames:        gitopsRuleNames,
		GitopsRuleAttrs:        gitopsRuleAttrs,
		PRTitle:                *prTitle,
		PRBody:                 *prBody,
		DryRun:                 *dryRun,
		Stamp:                  *stamp,
		Provider:               provider,
	}

	if err := prer.Run(
		context.Background(), cfg,
	); err != nil {
		return fmt.Errorf("%s: %w", errCtx, err)
	}

	return nil
}

// providerFlags bundles provider-specific flag values
// to keep newGitProvider under the 4-argument limit.
type providerFlags struct {
	ghRepoOwner  string
	ghRepo       string
	ghToken      string
	ghEnterprise string
	glHost       string
	glRepo       string
	glToken      string
	bbEndpoint   string
	bbUser       string
	bbPassword   string
}

// newGitProvider creates a git.GitProvider based on the
// server name. Pattern: Factory -- selects platform
// implementation at runtime.
func newGitProvider(
	server string,
	pf providerFlags,
) (git.GitProvider, error) {
	const errCtx = "creating git provider"

	switch server {
	case "github":
		p, err := github.NewProvider(github.Config{
			RepoOwner:      pf.ghRepoOwner,
			Repo:            pf.ghRepo,
			AccessToken:     pf.ghToken,
			EnterpriseHost:  pf.ghEnterprise,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		return p, nil

	case "gitlab":
		p, err := gitlab.NewProvider(gitlab.Config{
			Host:        pf.glHost,
			Repo:        pf.glRepo,
			AccessToken: pf.glToken,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		return p, nil

	case "bitbucket":
		p, err := bitbucket.NewProvider(
			bitbucket.Config{
				APIEndpoint: pf.bbEndpoint,
				User:        pf.bbUser,
				Password:    pf.bbPassword,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		return p, nil

	default:
		return nil, fmt.Errorf(
			"%s: unknown server %q", errCtx, server,
		)
	}
}
