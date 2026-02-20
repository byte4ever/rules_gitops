package gitlab

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// Config holds the settings needed to create a GitLab
// merge request provider.
type Config struct {
	// Host is the base URL of the GitLab instance
	// (e.g. "https://gitlab.com").
	Host string
	// Repo is the full project path
	// (e.g. "org/project").
	Repo string
	// AccessToken is a personal or project access
	// token used for authentication.
	AccessToken string
}

// Provider creates merge requests on GitLab.
//
// Pattern: Strategy -- implements git.GitProvider.
type Provider struct {
	client *gl.Client
	repo   string
}

// NewProvider validates cfg and returns a Provider
// ready to create merge requests.
func NewProvider(cfg Config) (*Provider, error) {
	const errCtx = "creating gitlab provider"

	if cfg.AccessToken == "" {
		return nil, fmt.Errorf(
			"%s: access token must be set", errCtx,
		)
	}

	if cfg.Repo == "" {
		return nil, fmt.Errorf(
			"%s: repo must be set", errCtx,
		)
	}

	host := cfg.Host
	if host == "" {
		host = "https://gitlab.com"
	}

	client, err := gl.NewClient(
		cfg.AccessToken,
		gl.WithBaseURL(host),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%s: new client: %w", errCtx, err,
		)
	}

	return &Provider{
		client: client,
		repo:   cfg.Repo,
	}, nil
}

// CreatePR creates a merge request from branch "from"
// into branch "to". If a MR already exists (HTTP 409)
// the error is suppressed.
func (p *Provider) CreatePR(
	_ context.Context,
	from string,
	to string,
	title string,
	_ string,
) error {
	const errCtx = "creating gitlab merge request"

	opts := gl.CreateMergeRequestOptions{
		Title:        &title,
		SourceBranch: &from,
		TargetBranch: &to,
	}

	created, resp, err := p.client.MergeRequests.CreateMergeRequest(
		p.repo, &opts,
	)
	if err == nil {
		slog.Info(
			"created merge request",
			"url", created.WebURL,
		)

		return nil
	}

	// HTTP 409: MR already exists for this source
	// branch.
	if resp != nil &&
		resp.StatusCode == http.StatusConflict {
		slog.Info(
			"reusing existing merge request",
		)

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
				"gitlab response",
				"body", string(rb),
			)
		}
	}

	return fmt.Errorf("%s: %w", errCtx, err)
}
