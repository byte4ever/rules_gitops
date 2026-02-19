package commitmsg_test

import (
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/commitmsg"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_produces_markers(t *testing.T) {
	t.Parallel()

	targets := []string{"//app:deploy.gitops", "//svc:deploy.gitops"}
	msg := commitmsg.Generate(targets)

	assert.Contains(t, msg, "--- gitops targets begin ---")
	assert.Contains(t, msg, "--- gitops targets end ---")
	assert.Contains(t, msg, "//app:deploy.gitops")
	assert.Contains(t, msg, "//svc:deploy.gitops")
}

func TestExtractTargets_roundtrip(t *testing.T) {
	t.Parallel()

	targets := []string{"target1", "target2"}
	msg := commitmsg.Generate(targets)
	got := commitmsg.ExtractTargets(msg)

	require.Equal(t, targets, got)
}

func TestExtractTargets_no_markers(t *testing.T) {
	t.Parallel()

	got := commitmsg.ExtractTargets("just a regular commit message")

	assert.Empty(t, got)
}

func TestExtractTargets_missing_end_marker(t *testing.T) {
	t.Parallel()

	msg := "--- gitops targets begin ---\ntarget1\n"
	got := commitmsg.ExtractTargets(msg)

	assert.Empty(t, got)
}
