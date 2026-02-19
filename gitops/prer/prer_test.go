package prer_test

import (
	"os"
	"path/filepath"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/gitops/prer"
)

func TestBazelQuery_parseJSON(t *testing.T) {
	t.Parallel()

	// Verify that our lightweight structs correctly
	// parse the jsonproto output format.
	raw := `{
		"results": [
			{
				"target": {
					"rule": {
						"name": "//pkg:deploy",
						"attribute": [
							{
								"name": "deployment_branch",
								"stringValue": "prod"
							},
							{
								"name": "release_branch_prefix",
								"stringValue": "release/v1"
							}
						]
					}
				}
			},
			{
				"target": {
					"rule": {
						"name": "//pkg:staging",
						"attribute": [
							{
								"name": "deployment_branch",
								"stringValue": "staging"
							},
							{
								"name": "release_branch_prefix",
								"stringValue": "release/v1"
							}
						]
					}
				}
			}
		]
	}`

	var qr prer.CqueryResult

	err := json.Unmarshal([]byte(raw), &qr)
	require.NoError(t, err)
	assert.Len(t, qr.Results, 2)

	first := qr.Results[0]
	assert.Equal(
		t, "//pkg:deploy", first.Target.Rule.Name,
	)
	assert.Len(t, first.Target.Rule.Attribute, 2)
	assert.Equal(
		t,
		"deployment_branch",
		first.Target.Rule.Attribute[0].Name,
	)
	assert.Equal(
		t,
		"prod",
		first.Target.Rule.Attribute[0].StringValue,
	)
}

func TestBazelQuery_emptyResults(t *testing.T) {
	t.Parallel()

	raw := `{"results": []}`

	var qr prer.CqueryResult

	err := json.Unmarshal([]byte(raw), &qr)
	require.NoError(t, err)
	assert.Empty(t, qr.Results)
}

func TestBuildDepsQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		targets   []string
		ruleNames []string
		want      string
	}{
		{
			name:      "single target single rule",
			targets:   []string{"//pkg:deploy"},
			ruleNames: []string{"push_image"},
			want: `kind("push_image", ` +
				`deps(//pkg:deploy))`,
		},
		{
			name: "two targets one rule",
			targets: []string{
				"//a:deploy", "//b:deploy",
			},
			ruleNames: []string{"push_image"},
			want: `kind("push_image", ` +
				`deps(//a:deploy)) + ` +
				`kind("push_image", ` +
				`deps(//b:deploy))`,
		},
		{
			name:      "one target two rules",
			targets:   []string{"//a:deploy"},
			ruleNames: []string{"push", "upload"},
			want: `kind("push", deps(//a:deploy))` +
				` + kind("upload", deps(//a:deploy))`,
		},
		{
			name:      "no rule names returns empty",
			targets:   []string{"//a:deploy"},
			ruleNames: nil,
			want:      "",
		},
		{
			name:      "no targets returns empty",
			targets:   nil,
			ruleNames: []string{"push_image"},
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := prer.Config{
				GitopsRuleNames: tt.ruleNames,
			}

			got := prer.BuildDepsQueryForTest(
				tt.targets, cfg,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetStampContext(t *testing.T) {
	t.Parallel()

	ctx := prer.GetStampContextForTest(
		"abc123", "feature/foo",
	)

	assert.Equal(
		t, "abc123", ctx["STABLE_GIT_COMMIT"],
	)
	assert.Equal(
		t, "feature/foo", ctx["STABLE_GIT_BRANCH"],
	)
	assert.Equal(
		t, "0", ctx["BUILD_TIMESTAMP"],
	)

	// Verify all expected keys are present.
	expectedKeys := []string{
		"STABLE_GIT_COMMIT",
		"STABLE_GIT_BRANCH",
		"BUILD_TIMESTAMP",
		"BUILD_EMBED_LABEL",
		"RANDOM_SEED",
		"STABLE_BUILD_LABEL",
	}

	for _, key := range expectedKeys {
		_, ok := ctx[key]
		assert.True(
			t, ok, "missing key: %s", key,
		)
	}
}

func TestStampFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fp := filepath.Join(dir, "manifest.yaml")

	content := `apiVersion: v1
image: myapp:{{STABLE_GIT_COMMIT}}
branch: {{STABLE_GIT_BRANCH}}
`

	err := os.WriteFile(
		fp, []byte(content), 0o600,
	)
	require.NoError(t, err)

	ctx := map[string]any{
		"STABLE_GIT_COMMIT": "deadbeef",
		"STABLE_GIT_BRANCH": "main",
	}

	err = prer.StampFileForTest(fp, ctx)
	require.NoError(t, err)

	got, err := os.ReadFile(fp)
	require.NoError(t, err)

	want := `apiVersion: v1
image: myapp:deadbeef
branch: main
`
	assert.Equal(t, want, string(got))
}

func TestStampFile_noPlaceholders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fp := filepath.Join(dir, "plain.txt")

	content := "no placeholders here\n"

	err := os.WriteFile(
		fp, []byte(content), 0o600,
	)
	require.NoError(t, err)

	ctx := map[string]any{
		"STABLE_GIT_COMMIT": "abc",
	}

	err = prer.StampFileForTest(fp, ctx)
	require.NoError(t, err)

	got, err := os.ReadFile(fp)
	require.NoError(t, err)

	assert.Equal(t, content, string(got))
}

func TestStampFile_missingFile(t *testing.T) {
	t.Parallel()

	err := prer.StampFileForTest(
		"/nonexistent/path.txt",
		map[string]any{},
	)
	assert.Error(t, err)
}

func TestGroupByTrain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		results       []prer.ConfiguredTarget
		releaseBranch string
		want          map[string][]string
	}{
		{
			name: "groups by deployment branch",
			results: []prer.ConfiguredTarget{
				makeTarget(
					"//a:deploy",
					"prod",
					"release/v1",
				),
				makeTarget(
					"//b:deploy",
					"prod",
					"release/v1",
				),
				makeTarget(
					"//c:deploy",
					"staging",
					"release/v1",
				),
			},
			releaseBranch: "release/v1",
			want: map[string][]string{
				"prod": {
					"//a:deploy",
					"//b:deploy",
				},
				"staging": {"//c:deploy"},
			},
		},
		{
			name: "filters by release branch",
			results: []prer.ConfiguredTarget{
				makeTarget(
					"//a:deploy",
					"prod",
					"release/v1",
				),
				makeTarget(
					"//b:deploy",
					"prod",
					"release/v2",
				),
			},
			releaseBranch: "release/v1",
			want: map[string][]string{
				"prod": {"//a:deploy"},
			},
		},
		{
			name: "skips targets without dep branch",
			results: []prer.ConfiguredTarget{
				{
					Target: prer.QueryTarget{
						Rule: prer.QueryRule{
							Name: "//a:deploy",
							Attribute: []prer.QueryAttribute{
								{
									Name:        "release_branch_prefix",
									StringValue: "release/v1",
								},
							},
						},
					},
				},
			},
			releaseBranch: "release/v1",
			want:          map[string][]string{},
		},
		{
			name:          "empty results",
			results:       nil,
			releaseBranch: "release/v1",
			want:          map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			qr := &prer.CqueryResult{
				Results: tt.results,
			}

			got := prer.GroupByTrainForTest(
				qr, tt.releaseBranch,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasDeletedTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prev    []string
		current []string
		want    bool
	}{
		{
			name:    "no deletions",
			prev:    []string{"//a:t", "//b:t"},
			current: []string{"//a:t", "//b:t"},
			want:    false,
		},
		{
			name: "has deletions",
			prev: []string{
				"//a:t", "//b:t", "//c:t",
			},
			current: []string{"//a:t", "//b:t"},
			want:    true,
		},
		{
			name:    "empty prev",
			prev:    nil,
			current: []string{"//a:t"},
			want:    false,
		},
		{
			name:    "empty current",
			prev:    []string{"//a:t"},
			current: nil,
			want:    true,
		},
		{
			name:    "both empty",
			prev:    nil,
			current: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := prer.HasDeletedTargetsForTest(
				tt.prev, tt.current,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollectAllTargets(t *testing.T) {
	t.Parallel()

	trains := map[string][]string{
		"prod":    {"//b:deploy", "//a:deploy"},
		"staging": {"//a:deploy", "//c:deploy"},
	}

	got := prer.CollectAllTargetsForTest(trains)

	// Should be sorted and deduplicated.
	want := []string{
		"//a:deploy", "//b:deploy", "//c:deploy",
	}
	assert.Equal(t, want, got)
}

func TestExtractTargetNames(t *testing.T) {
	t.Parallel()

	qr := &prer.CqueryResult{
		Results: []prer.ConfiguredTarget{
			{
				Target: prer.QueryTarget{
					Rule: prer.QueryRule{
						Name: "//a:push",
					},
				},
			},
			{
				Target: prer.QueryTarget{
					Rule: prer.QueryRule{
						Name: "//b:push",
					},
				},
			},
			{
				Target: prer.QueryTarget{
					Rule: prer.QueryRule{
						Name: "",
					},
				},
			},
		},
	}

	got := prer.ExtractTargetNamesForTest(qr)
	assert.Equal(
		t, []string{"//a:push", "//b:push"}, got,
	)
}

// makeTarget is a test helper that builds a
// ConfiguredTarget with deployment_branch and
// release_branch_prefix attributes.
func makeTarget(
	name string,
	depBranch string,
	relBranch string,
) prer.ConfiguredTarget {
	return prer.ConfiguredTarget{
		Target: prer.QueryTarget{
			Rule: prer.QueryRule{
				Name: name,
				Attribute: []prer.QueryAttribute{
					{
						Name:        "deployment_branch",
						StringValue: depBranch,
					},
					{
						Name:        "release_branch_prefix",
						StringValue: relBranch,
					},
				},
			},
		},
	}
}
