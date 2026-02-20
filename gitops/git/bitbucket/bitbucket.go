package bitbucket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	json "github.com/goccy/go-json"
)

// Config holds the settings needed to create a
// Bitbucket pull request provider.
type Config struct {
	// APIEndpoint is the full Bitbucket Server REST
	// API URL for pull requests, including project
	// and repo path (e.g.
	// "https://bb.example.com/rest/api/1.0/
	// projects/PROJ/repos/repo/pull-requests").
	APIEndpoint string
	// User is the Bitbucket API username.
	User string
	// Password is the Bitbucket API password (or
	// personal access token).
	Password string
}

// Provider creates pull requests on Bitbucket Server.
//
// Pattern: Strategy -- implements git.GitProvider.
type Provider struct {
	endpoint string
	user     string
	password string
}

type project struct {
	Key string `json:"key,omitempty"`
}

type repository struct {
	Slug    string  `json:"slug,omitempty"`
	Project project `json:"project"`
}

type pullrequestEndpoint struct {
	ID         string     `json:"id,omitempty"`
	Repository repository `json:"repository,omitempty"`
}

type pullrequest struct {
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	State       string               `json:"state,omitempty"`
	Open        bool                 `json:"open"`
	Closed      bool                 `json:"closed"`
	FromRef     *pullrequestEndpoint `json:"fromRef,omitempty"`
	ToRef       *pullrequestEndpoint `json:"toRef,omitempty"`
	Locked      bool                 `json:"locked"`
	Reviewers   []account            `json:"reviewers,omitempty"`
}

type account struct {
	User user `json:"user"`
}

type user struct {
	Name string `json:"name,omitempty"`
}

// NewProvider validates cfg and returns a Provider
// ready to create pull requests.
func NewProvider(cfg Config) (*Provider, error) {
	const errCtx = "creating bitbucket provider"

	if cfg.APIEndpoint == "" {
		return nil, fmt.Errorf(
			"%s: api endpoint must be set",
			errCtx,
		)
	}

	if cfg.User == "" {
		return nil, fmt.Errorf(
			"%s: user must be set", errCtx,
		)
	}

	if cfg.Password == "" {
		return nil, fmt.Errorf(
			"%s: password must be set", errCtx,
		)
	}

	return &Provider{
		endpoint: cfg.APIEndpoint,
		user:     cfg.User,
		password: cfg.Password,
	}, nil
}

// CreatePR creates a pull request from branch "from"
// into branch "to". Returns nil on 201 (created) or
// 409 (already exists).
func (p *Provider) CreatePR(
	ctx context.Context,
	from string,
	to string,
	title string,
	body string,
) error {
	const errCtx = "creating bitbucket pull request"

	repo := repository{
		Slug:    "repo",
		Project: project{Key: "TM"},
	}

	pr := pullrequest{
		Title:       title,
		Description: body,
		State:       "OPEN",
		Open:        true,
		Closed:      false,
		FromRef: &pullrequestEndpoint{
			ID:         "refs/heads/" + from,
			Repository: repo,
		},
		ToRef: &pullrequestEndpoint{
			ID:         "refs/heads/" + to,
			Repository: repo,
		},
		Locked:    false,
		Reviewers: []account{},
	}

	payload, err := json.Marshal(&pr)
	if err != nil {
		return fmt.Errorf(
			"%s: marshal request: %w", errCtx, err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.endpoint,
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf(
			"%s: build request: %w", errCtx, err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	req.SetBasicAuth(p.user, p.password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf(
			"%s: send request: %w", errCtx, err,
		)
	}

	defer resp.Body.Close() //nolint:errcheck

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn(
			"cannot read response body",
			"error", err,
		)
	} else {
		slog.Info(
			"bitbucket response",
			"status", resp.Status,
			"body", string(rb),
		)
	}

	// 201 Created: PR was created successfully.
	if resp.StatusCode == http.StatusCreated {
		slog.Info("pull request created")

		return nil
	}

	// 409 Conflict: PR already exists.
	if resp.StatusCode == http.StatusConflict {
		slog.Info("reusing existing pull request")

		return nil
	}

	return fmt.Errorf(
		"%s: unexpected status %d",
		errCtx, resp.StatusCode,
	)
}
