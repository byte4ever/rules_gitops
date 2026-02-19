package bitbucket_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bb "github.com/byte4ever/rules_gitops/gitops/git/bitbucket"
)

func TestNewProvider_valid(t *testing.T) {
	t.Parallel()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: "https://bb.example.com/rest",
		User:        "admin",
		Password:    "secret",
	})

	require.NoError(t, err)
	assert.NotNil(t, pv)
}

func TestNewProvider_missing_endpoint(t *testing.T) {
	t.Parallel()

	pv, err := bb.NewProvider(bb.Config{
		User:     "admin",
		Password: "secret",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "api endpoint")
}

func TestNewProvider_missing_user(t *testing.T) {
	t.Parallel()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: "https://bb.example.com/rest",
		Password:    "secret",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "user must be set")
}

func TestNewProvider_missing_password(t *testing.T) {
	t.Parallel()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: "https://bb.example.com/rest",
		User:        "admin",
	})

	assert.Nil(t, pv)
	assert.ErrorContains(t, err, "password")
}

func TestProvider_CreatePR_created(t *testing.T) {
	t.Parallel()

	var gotBody []byte

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				var err error

				gotBody, err = io.ReadAll(r.Body)
				if err != nil {
					http.Error(
						w,
						"read error",
						http.StatusInternalServerError,
					)

					return
				}

				w.WriteHeader(http.StatusCreated)
			},
		),
	)
	defer ts.Close()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: ts.URL,
		User:        "admin",
		Password:    "secret",
	})
	require.NoError(t, err)

	err = pv.CreatePR(
		context.Background(),
		"deploy/test1",
		"feature/AP-0000",
		"test",
		"hello world",
	)

	require.NoError(t, err)
	assert.Contains(
		t, string(gotBody), `"title":"test"`,
	)
	assert.Contains(
		t, string(gotBody),
		`"description":"hello world"`,
	)
	assert.Contains(
		t, string(gotBody),
		`refs/heads/deploy/test1`,
	)
}

func TestProvider_CreatePR_conflict(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(http.StatusConflict)
			},
		),
	)
	defer ts.Close()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: ts.URL,
		User:        "admin",
		Password:    "secret",
	})
	require.NoError(t, err)

	err = pv.CreatePR(
		context.Background(),
		"a", "b", "t", "d",
	)

	assert.NoError(t, err)
}

func TestProvider_CreatePR_unexpected_status(
	t *testing.T,
) {
	t.Parallel()

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				_ *http.Request,
			) {
				w.WriteHeader(
					http.StatusInternalServerError,
				)
			},
		),
	)
	defer ts.Close()

	pv, err := bb.NewProvider(bb.Config{
		APIEndpoint: ts.URL,
		User:        "admin",
		Password:    "secret",
	})
	require.NoError(t, err)

	err = pv.CreatePR(
		context.Background(),
		"a", "b", "t", "d",
	)

	assert.ErrorContains(t, err, "unexpected status")
}
