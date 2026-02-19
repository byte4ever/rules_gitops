package git_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/gitops/git"
)

func TestGitProviderFunc_CreatePR_passes_args(
	t *testing.T,
) {
	t.Parallel()

	var (
		gotFrom  string
		gotTo    string
		gotTitle string
		gotBody  string
	)

	fn := git.GitProviderFunc(
		func(
			_ context.Context,
			from string,
			to string,
			title string,
			body string,
		) error {
			gotFrom = from
			gotTo = to
			gotTitle = title
			gotBody = body

			return nil
		},
	)

	err := fn.CreatePR(
		context.Background(),
		"feature/x",
		"main",
		"my title",
		"my body",
	)

	require.NoError(t, err)
	assert.Equal(t, "feature/x", gotFrom)
	assert.Equal(t, "main", gotTo)
	assert.Equal(t, "my title", gotTitle)
	assert.Equal(t, "my body", gotBody)
}

func TestGitProviderFunc_CreatePR_empty_body_uses_title(
	t *testing.T,
) {
	t.Parallel()

	var gotBody string

	fn := git.GitProviderFunc(
		func(
			_ context.Context,
			_ string,
			_ string,
			_ string,
			body string,
		) error {
			gotBody = body

			return nil
		},
	)

	err := fn.CreatePR(
		context.Background(),
		"a",
		"b",
		"the title",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, "the title", gotBody)
}

func TestGitProviderFunc_CreatePR_returns_error(
	t *testing.T,
) {
	t.Parallel()

	errTest := errors.New("test error")

	fn := git.GitProviderFunc(
		func(
			_ context.Context,
			_ string,
			_ string,
			_ string,
			_ string,
		) error {
			return errTest
		},
	)

	err := fn.CreatePR(
		context.Background(),
		"a",
		"b",
		"t",
		"d",
	)

	assert.ErrorIs(t, err, errTest)
}
