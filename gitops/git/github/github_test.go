package github_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ghprov "github.com/byte4ever/rules_gitops/gitops/git/github"
)

func TestNewProvider_valid(t *testing.T) {
	t.Parallel()

	pv, err := ghprov.NewProvider(ghprov.Config{
		RepoOwner:   "org",
		Repo:        "repo",
		AccessToken: "tok",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}

func TestNewProvider_missing_owner(t *testing.T) {
	t.Parallel()

	pv, err := ghprov.NewProvider(ghprov.Config{
		Repo:        "repo",
		AccessToken: "tok",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "repo owner")
}

func TestNewProvider_missing_repo(t *testing.T) {
	t.Parallel()

	pv, err := ghprov.NewProvider(ghprov.Config{
		RepoOwner:   "org",
		AccessToken: "tok",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "repo must be set")
}

func TestNewProvider_missing_token(t *testing.T) {
	t.Parallel()

	pv, err := ghprov.NewProvider(ghprov.Config{
		RepoOwner: "org",
		Repo:      "repo",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "access token")
}

func TestNewProvider_enterprise(t *testing.T) {
	t.Parallel()

	pv, err := ghprov.NewProvider(ghprov.Config{
		RepoOwner:      "org",
		Repo:           "repo",
		AccessToken:    "tok",
		EnterpriseHost: "git.corp.example.com",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}
