package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	gh "github.com/google/go-github/v68/github"
)

// Config holds the settings needed to create a GitHub
// pull request provider.
type Config struct {
	// RepoOwner is the GitHub user or organisation
	// that owns the repository.
	RepoOwner string
	// Repo is the repository name (without owner).
	Repo string
	// AccessToken is a personal access token or
	// GitHub App token used for authentication.
	AccessToken string
	// EnterpriseHost is an optional GitHub Enterprise
	// hostname (e.g. "git.corp.example.com"). Leave
	// empty for github.com.
	EnterpriseHost string
}

// Provider creates pull requests on GitHub.
//
// Pattern: Strategy -- implements git.GitProvider.
type Provider struct {
	client    *gh.Client
	repoOwner string
	repo      string
}

// NewProvider validates cfg and returns a Provider
// ready to create pull requests.
func NewProvider(cfg Config) (*Provider, error) {
	const errCtx = "creating github provider"

	if cfg.RepoOwner == "" {
		return nil, fmt.Errorf(
			"%s: repo owner must be set", errCtx,
		)
	}

	if cfg.Repo == "" {
		return nil, fmt.Errorf(
			"%s: repo must be set", errCtx,
		)
	}

	if cfg.AccessToken == "" {
		return nil, fmt.Errorf(
			"%s: access token must be set", errCtx,
		)
	}

	client := gh.NewClient(nil).
		WithAuthToken(cfg.AccessToken)

	if cfg.EnterpriseHost != "" {
		baseURL := "https://" +
			cfg.EnterpriseHost + "/api/v3/"
		uploadURL := "https://" +
			cfg.EnterpriseHost + "/api/uploads/"

		var err error

		client, err = client.WithEnterpriseURLs(
			baseURL, uploadURL,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%s: enterprise urls: %w",
				errCtx, err,
			)
		}
	}

	return &Provider{
		client:    client,
		repoOwner: cfg.RepoOwner,
		repo:      cfg.Repo,
	}, nil
}

// CreatePR creates a pull request from branch "from"
// into branch "to". If a PR already exists (HTTP 422)
// the error is suppressed.
func (p *Provider) CreatePR(
	ctx context.Context,
	from string,
	to string,
	title string,
	body string,
) error {
	const errCtx = "creating github pull request"

	pr := &gh.NewPullRequest{
		Title: &title,
		Head:  &from,
		Base:  &to,
		Body:  &body,
	}

	created, resp, err := p.client.PullRequests.Create(
		ctx, p.repoOwner, p.repo, pr,
	)
	if err == nil {
		slog.Info(
			"created pull request",
			"url", created.GetURL(),
		)

		return nil
	}

	// HTTP 422: PR already exists for this
	// head/base pair.
	if resp != nil &&
		resp.StatusCode ==
			http.StatusUnprocessableEntity {
		slog.Info("reusing existing pull request")

		return nil
	}

	// Log the response body for debugging.
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close() //nolint:errcheck

		rb, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			slog.Warn(
				"cannot read response body",
				"error", readErr,
			)
		} else {
			slog.Warn(
				"github response",
				"body", string(rb),
			)
		}
	}

	return fmt.Errorf("%s: %w", errCtx, err)
}
