package bazel_test

import (
	"testing"

	"github.com/byte4ever/rules_gitops/gitops/bazel"

	"github.com/stretchr/testify/assert"
)

func TestTargetToExecutable_full_target(t *testing.T) {
	t.Parallel()

	got := bazel.TargetToExecutable(
		"//app/deploy:deploy-prod.gitops",
	)

	assert.Equal(
		t,
		"bazel-bin/app/deploy/deploy-prod.gitops",
		got,
	)
}

func TestTargetToExecutable_no_prefix(t *testing.T) {
	t.Parallel()

	got := bazel.TargetToExecutable("some/path")

	assert.Equal(t, "some/path", got)
}
