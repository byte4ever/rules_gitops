package gitlab_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glprov "github.com/byte4ever/rules_gitops/gitops/git/gitlab"
)

func TestNewProvider_valid(t *testing.T) {
	t.Parallel()

	pv, err := glprov.NewProvider(glprov.Config{
		Repo:        "org/project",
		AccessToken: "tok",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}

func TestNewProvider_default_host(t *testing.T) {
	t.Parallel()

	// When Host is empty, default to gitlab.com.
	pv, err := glprov.NewProvider(glprov.Config{
		Repo:        "org/project",
		AccessToken: "tok",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}

func TestNewProvider_custom_host(t *testing.T) {
	t.Parallel()

	pv, err := glprov.NewProvider(glprov.Config{
		Host:        "https://gl.corp.example.com",
		Repo:        "org/project",
		AccessToken: "tok",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}

func TestNewProvider_missing_token(t *testing.T) {
	t.Parallel()

	pv, err := glprov.NewProvider(glprov.Config{
		Repo: "org/project",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "access token")
}

func TestNewProvider_missing_repo(t *testing.T) {
	t.Parallel()

	pv, err := glprov.NewProvider(glprov.Config{
		AccessToken: "tok",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "repo must be set")
}
