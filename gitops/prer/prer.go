// Package prer orchestrates the creation of gitops
// pull requests. It queries Bazel for gitops targets,
// groups them by deployment train, clones the git
// repository, runs each target, stamps files, commits
// changes, pushes images, and creates PRs.
package prer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
	"github.com/valyala/fasttemplate"

	"github.com/byte4ever/rules_gitops/gitops/bazel"
	"github.com/byte4ever/rules_gitops/gitops/commitmsg"
	"github.com/byte4ever/rules_gitops/gitops/digester"
	"github.com/byte4ever/rules_gitops/gitops/exec"
	"github.com/byte4ever/rules_gitops/gitops/git"
)

// Config holds all settings for a gitops PR creation
// run. Use a Config struct instead of many arguments.
type Config struct {
	// BazelCmd is the bazel binary name or path.
	BazelCmd string

	// Workspace is the bazel workspace root.
	Workspace string

	// Target is the bazel query target pattern.
	Target string

	// GitRepo is the remote repository URL.
	GitRepo string

	// GitMirror is an optional local mirror path.
	GitMirror string

	// GitopsPath restricts the git sparse checkout
	// to a subdirectory (empty means root).
	GitopsPath string

	// TmpDir is the directory for temporary clones.
	TmpDir string

	// ReleaseBranch filters targets by this release
	// branch attribute value.
	ReleaseBranch string

	// PrimaryBranch is the main branch (e.g. "main").
	PrimaryBranch string

	// DeploymentBranchPrefix prepended to deployment
	// branch names.
	DeploymentBranchPrefix string

	// DeploymentBranchSuffix appended to deployment
	// branch names.
	DeploymentBranchSuffix string

	// BranchName is the source branch name used in
	// stamp context.
	BranchName string

	// GitCommit is the source commit SHA used in
	// stamp context.
	GitCommit string

	// PushParallelism is the number of concurrent
	// image push workers.
	PushParallelism int

	// GitopsKinds restricts which Bazel rule kinds
	// to query (e.g. "gitops", "k8s_deploy").
	GitopsKinds []string

	// GitopsRuleNames restricts which rule names
	// to look for when building push queries.
	GitopsRuleNames []string

	// GitopsRuleAttrs restricts which rule
	// attributes to match when building push
	// queries.
	GitopsRuleAttrs []string

	// PRTitle is the title for created pull requests.
	PRTitle string

	// PRBody is the body for created pull requests.
	PRBody string

	// DryRun skips push and PR creation when true.
	DryRun bool

	// Stamp enables file stamping when true.
	Stamp bool

	// Provider creates pull requests on a git
	// hosting platform.
	Provider git.GitProvider
}

// cqueryResult mirrors the JSON output of
// bazel cquery --output=jsonproto.
type cqueryResult struct {
	Results []configuredTarget `json:"results"`
}

// configuredTarget wraps a single target from the
// cquery result set.
type configuredTarget struct {
	Target queryTarget `json:"target"`
}

// queryTarget holds the rule information for one
// configured target.
type queryTarget struct {
	Rule queryRule `json:"rule"`
}

// queryRule holds the name and attributes of a Bazel
// rule as returned by cquery jsonproto output.
type queryRule struct {
	Name      string           `json:"name"`
	Attribute []queryAttribute `json:"attribute"`
}

// queryAttribute holds a single attribute name/value
// pair from a Bazel rule.
type queryAttribute struct {
	Name        string `json:"name"`
	StringValue string `json:"stringValue"`
}

// Run executes the full gitops PR creation workflow.
// It queries Bazel, groups targets by deployment
// train, clones the repo, runs targets, stamps files,
// commits changes, pushes images, and creates PRs.
func Run(ctx context.Context, cfg Config) error {
	const errCtx = "running gitops pr creation"

	// Step 1: Query bazel for gitops targets.
	query := buildKindQuery(cfg)

	qr, err := bazelQuery(cfg.BazelCmd, query)
	if err != nil {
		return fmt.Errorf(
			"%s: query targets: %w", errCtx, err,
		)
	}

	// Step 2: Group by deployment train.
	trains := groupByTrain(qr, cfg.ReleaseBranch)
	if len(trains) == 0 {
		slog.Info(
			"no targets matching release branch",
			"branch", cfg.ReleaseBranch,
		)

		return nil
	}

	// Step 3: Clone git repository.
	cloneDir := filepath.Join(cfg.TmpDir, "gitops")

	repo, err := git.Clone(
		cfg.GitRepo,
		cloneDir,
		cfg.GitMirror,
		cfg.PrimaryBranch,
		cfg.GitopsPath,
	)
	if err != nil {
		return fmt.Errorf(
			"%s: clone repo: %w", errCtx, err,
		)
	}

	defer func() {
		if cleanErr := repo.Clean(); cleanErr != nil {
			slog.Error(
				"failed to clean repo",
				"error", cleanErr,
			)
		}
	}()

	// Fetch deployment branch patterns.
	fetchPattern := cfg.DeploymentBranchPrefix + "*"
	repo.Fetch(fetchPattern)

	// Step 4: Process each deployment train.
	var updatedBranches []string

	stampCtx := getStampContext(
		cfg.GitCommit, cfg.BranchName,
	)

	for branch, targets := range trains {
		depBranch := cfg.DeploymentBranchPrefix +
			branch +
			cfg.DeploymentBranchSuffix

		updated, branchErr := processTrain(
			repo, cfg, depBranch, targets, stampCtx,
		)
		if branchErr != nil {
			return fmt.Errorf(
				"%s: train %s: %w",
				errCtx, branch, branchErr,
			)
		}

		if updated {
			updatedBranches = append(
				updatedBranches, depBranch,
			)
		}
	}

	if len(updatedBranches) == 0 {
		slog.Info("no branches updated, skipping push")

		return nil
	}

	// Step 5: Build deps query and push images.
	allTargets := collectAllTargets(trains)

	if err := pushImages(
		ctx, cfg, allTargets,
	); err != nil {
		return fmt.Errorf(
			"%s: push images: %w", errCtx, err,
		)
	}

	// Step 6: Push branches and create PRs.
	if cfg.DryRun {
		slog.Info(
			"dry run: skipping push and PR creation",
			"branches", updatedBranches,
		)

		return nil
	}

	repo.Push(updatedBranches)

	for _, branch := range updatedBranches {
		if err := cfg.Provider.CreatePR(
			ctx,
			branch,
			cfg.PrimaryBranch,
			cfg.PRTitle,
			cfg.PRBody,
		); err != nil {
			return fmt.Errorf(
				"%s: create PR for %s: %w",
				errCtx, branch, err,
			)
		}
	}

	return nil
}

// processTrain handles a single deployment train:
// switches branch, runs targets, stamps files, and
// commits. Returns true if changes were committed.
func processTrain(
	repo *git.Repo,
	cfg Config,
	depBranch string,
	targets []string,
	stampCtx map[string]any,
) (bool, error) {
	const errCtx = "processing deployment train"

	isNew := repo.SwitchToBranch(
		depBranch, cfg.PrimaryBranch,
	)

	// Check if previously deployed targets were
	// removed — if so, recreate the branch.
	if !isNew {
		lastMsg := repo.GetLastCommitMessage()
		prev := commitmsg.ExtractTargets(lastMsg)

		if hasDeletedTargets(prev, targets) {
			slog.Info(
				"recreating branch due to deleted "+
					"targets",
				"branch", depBranch,
			)

			repo.RecreateBranch(
				depBranch, cfg.PrimaryBranch,
			)
		}
	}

	// Run each gitops target executable.
	for _, target := range targets {
		exe := bazel.TargetToExecutable(target)
		exec.MustEx(cfg.Workspace, exe)
	}

	// Stamp changed files if enabled.
	if cfg.Stamp {
		if err := stampChangedFiles(
			repo, stampCtx,
		); err != nil {
			return false, fmt.Errorf(
				"%s: stamp files: %w", errCtx, err,
			)
		}
	}

	// Commit changes.
	msg := commitmsg.Generate(targets)

	committed := repo.Commit(msg, cfg.GitopsPath)

	return committed, nil
}

// stampChangedFiles iterates changed files, verifies
// digests, and applies stamp template substitution.
func stampChangedFiles(
	repo *git.Repo,
	stampCtx map[string]any,
) error {
	const errCtx = "stamping changed files"

	changed := repo.GetChangedFiles()

	for _, f := range changed {
		absPath := filepath.Join(repo.Dir, f)

		ok, err := digester.VerifyDigest(absPath)
		if err != nil {
			return fmt.Errorf(
				"%s: verify %s: %w",
				errCtx, f, err,
			)
		}

		if ok {
			// Digest matches — restore file to
			// avoid unnecessary changes.
			repo.RestoreFile(f)

			continue
		}

		if err := stampFile(
			absPath, stampCtx,
		); err != nil {
			return fmt.Errorf(
				"%s: stamp %s: %w",
				errCtx, f, err,
			)
		}

		if err := digester.SaveDigest(
			absPath,
		); err != nil {
			return fmt.Errorf(
				"%s: save digest %s: %w",
				errCtx, f, err,
			)
		}
	}

	return nil
}

// bazelQuery runs a bazel cquery with jsonproto output
// and parses the result into a cqueryResult.
func bazelQuery(
	bazelCmd string,
	query string,
) (*cqueryResult, error) {
	const errCtx = "running bazel cquery"

	out, err := exec.Ex(
		"",
		bazelCmd,
		"cquery",
		"--output=jsonproto",
		query,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: %w", errCtx, err,
		)
	}

	var qr cqueryResult
	if err := json.Unmarshal(
		[]byte(out), &qr,
	); err != nil {
		return nil, fmt.Errorf(
			"%s: parse json: %w", errCtx, err,
		)
	}

	return &qr, nil
}

// getStampContext creates a map of template variables
// used for file stamping. Keys are variable names and
// values are their replacements.
func getStampContext(
	gitCommit string,
	branchName string,
) map[string]any {
	return map[string]any{
		"STABLE_GIT_COMMIT":  gitCommit,
		"STABLE_GIT_BRANCH":  branchName,
		"BUILD_TIMESTAMP":    "0",
		"BUILD_EMBED_LABEL":  "",
		"RANDOM_SEED":        "",
		"STABLE_BUILD_LABEL": "",
	}
}

// stampFile replaces {{VAR}} placeholders in the file
// at path using the provided stamp context. Uses
// valyala/fasttemplate for substitution.
func stampFile(
	path string,
	ctx map[string]any,
) error {
	const errCtx = "stamping file"

	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return fmt.Errorf(
			"%s: read %s: %w", errCtx, path, err,
		)
	}

	tpl := fasttemplate.New(
		string(data), "{{", "}}",
	)

	result := tpl.ExecuteString(ctx)

	//nolint:gosec // permissions match source
	if err := os.WriteFile(
		path, []byte(result), 0o644,
	); err != nil {
		return fmt.Errorf(
			"%s: write %s: %w", errCtx, path, err,
		)
	}

	return nil
}

// pushImages runs image push targets in parallel using
// a worker pool bounded by cfg.PushParallelism.
func pushImages(
	ctx context.Context,
	cfg Config,
	targets []string,
) error {
	const errCtx = "pushing images"

	// Build the deps query to find push targets.
	depsQuery := buildDepsQuery(targets, cfg)
	if depsQuery == "" {
		slog.Info("no push targets to query")

		return nil
	}

	qr, err := bazelQuery(cfg.BazelCmd, depsQuery)
	if err != nil {
		return fmt.Errorf(
			"%s: query deps: %w", errCtx, err,
		)
	}

	pushTargets := extractTargetNames(qr)
	if len(pushTargets) == 0 {
		slog.Info("no push targets found")

		return nil
	}

	slog.Info(
		"pushing images",
		"count", len(pushTargets),
		"parallelism", cfg.PushParallelism,
	)

	parallelism := cfg.PushParallelism
	if parallelism <= 0 {
		parallelism = 1
	}

	// Worker pool with bounded concurrency.
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	sem := make(chan struct{}, parallelism)

	for _, target := range pushTargets {
		// Check for context cancellation.
		if ctx.Err() != nil {
			mu.Lock()
			errs = append(errs, ctx.Err())
			mu.Unlock()

			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(tgt string) {
			defer wg.Done()
			defer func() { <-sem }()

			exe := bazel.TargetToExecutable(tgt)

			if _, pushErr := exec.Ex(
				cfg.Workspace, exe,
			); pushErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf(
					"push %s: %w", tgt, pushErr,
				))
				mu.Unlock()
			}
		}(target)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf(
			"%s: %d errors, first: %w",
			errCtx, len(errs), errs[0],
		)
	}

	return nil
}

// buildKindQuery constructs a bazel query expression
// that selects gitops targets matching the configured
// rule kinds.
func buildKindQuery(cfg Config) string {
	if len(cfg.GitopsKinds) == 0 {
		return cfg.Target
	}

	var parts []string

	for _, kind := range cfg.GitopsKinds {
		parts = append(parts, fmt.Sprintf(
			"kind(%q, %s)", kind, cfg.Target,
		))
	}

	return strings.Join(parts, " + ")
}

// buildDepsQuery builds a union query for finding
// push-related dependencies of the given targets.
// Returns empty string if no rule names are configured.
func buildDepsQuery(
	targets []string,
	cfg Config,
) string {
	if len(cfg.GitopsRuleNames) == 0 {
		return ""
	}

	var parts []string

	for _, target := range targets {
		for _, ruleName := range cfg.GitopsRuleNames {
			parts = append(parts, fmt.Sprintf(
				"kind(%q, deps(%s))",
				ruleName, target,
			))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " + ")
}

// groupByTrain groups cquery results by the
// "deployment_branch" attribute value. Only targets
// matching the release branch are included.
func groupByTrain(
	qr *cqueryResult,
	releaseBranch string,
) map[string][]string {
	trains := make(map[string][]string)

	for _, r := range qr.Results {
		rule := r.Target.Rule

		depBranch := ""
		matchesBranch := false

		for _, attr := range rule.Attribute {
			switch attr.Name {
			case "deployment_branch":
				depBranch = attr.StringValue
			case "release_branch_prefix":
				if attr.StringValue == releaseBranch {
					matchesBranch = true
				}
			default:
				continue
			}
		}

		if !matchesBranch || depBranch == "" {
			continue
		}

		trains[depBranch] = append(
			trains[depBranch], rule.Name,
		)
	}

	// Sort targets within each train for
	// deterministic ordering.
	for k := range trains {
		sort.Strings(trains[k])
	}

	return trains
}

// hasDeletedTargets returns true if any previously
// deployed target is missing from the current set.
func hasDeletedTargets(
	prev []string,
	current []string,
) bool {
	cur := make(map[string]struct{}, len(current))
	for _, t := range current {
		cur[t] = struct{}{}
	}

	for _, t := range prev {
		if _, ok := cur[t]; !ok {
			return true
		}
	}

	return false
}

// extractTargetNames returns the rule names from all
// results in a cquery output.
func extractTargetNames(
	qr *cqueryResult,
) []string {
	names := make([]string, 0, len(qr.Results))

	for _, r := range qr.Results {
		if r.Target.Rule.Name != "" {
			names = append(
				names, r.Target.Rule.Name,
			)
		}
	}

	return names
}

// collectAllTargets gathers all target names from
// every deployment train into a single sorted slice.
func collectAllTargets(
	trains map[string][]string,
) []string {
	seen := make(map[string]struct{})
	var all []string

	for _, targets := range trains {
		for _, t := range targets {
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				all = append(all, t)
			}
		}
	}

	sort.Strings(all)

	return all
}
