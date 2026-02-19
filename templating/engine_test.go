package templating_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/templating"
)

// helper creates a temporary file with content and
// returns its path.
func writeTemp(
	tb testing.TB,
	dir string,
	name string,
	content string,
) string {
	tb.Helper()

	pa := filepath.Join(dir, name)
	require.NoError(tb, os.WriteFile(pa, []byte(content), 0o600))

	return pa
}

func TestExpand_variable_substitution_custom_tags(
	t *testing.T,
) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(
		t, dir, "tpl.txt", "Hello <%name%>!",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StartTag: "<%",
		EndTag:   "%>",
	}

	err := en.Expand(
		tplPath, outPath,
		[]string{"name=World"},
		nil,
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", string(got))
}

func TestExpand_stamp_file_substitution(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	stampPath := writeTemp(
		t, dir, "stamp.txt",
		"BUILD_USER alice\nBUILD_HOST ci-01\n",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"Built by {{BUILD_USER}} on {{BUILD_HOST}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StampInfoFiles: []string{stampPath},
	}

	err := en.Expand(
		tplPath, outPath, nil, nil, false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(
		t,
		"Built by alice on ci-01",
		string(got),
	)
}

func TestExpand_missing_template_file(t *testing.T) {
	t.Parallel()

	en := templating.Engine{}

	err := en.Expand(
		"/nonexistent/template.txt",
		"",
		nil,
		nil,
		false,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding template")
}

func TestExpand_variables_override_stamps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	stampPath := writeTemp(
		t, dir, "stamp.txt",
		"VERSION 1.0.0\n",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"version={{VERSION}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StampInfoFiles: []string{stampPath},
	}

	// Stamp sets VERSION=1.0.0; variable overrides it.
	err := en.Expand(
		tplPath, outPath,
		[]string{"VERSION=2.0.0"},
		nil,
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "version=2.0.0", string(got))
}

func TestExpand_stamp_substitution_in_variable_values(
	t *testing.T,
) {
	t.Parallel()

	dir := t.TempDir()

	stampPath := writeTemp(
		t, dir, "stamp.txt",
		"BUILD_USER alice\n",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"author={{AUTHOR}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StampInfoFiles: []string{stampPath},
	}

	// Variable value contains stamp reference with
	// single-brace tags.
	err := en.Expand(
		tplPath, outPath,
		[]string{"AUTHOR={BUILD_USER}"},
		nil,
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "author=alice", string(got))
}

func TestExpand_variables_prefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"{{variables.APP}}-{{APP}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, outPath,
		[]string{"APP=myapp"},
		nil,
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "myapp-myapp", string(got))
}

func TestExpand_imports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	importFile := writeTemp(
		t, dir, "partial.txt",
		"Hello {{NAME}}!",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"result={{imports.greeting}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, outPath,
		[]string{"NAME=World"},
		[]string{"greeting=" + importFile},
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "result=Hello World!", string(got))
}

func TestExpand_imports_with_stamps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	stampPath := writeTemp(
		t, dir, "stamp.txt",
		"BUILD_NUM 42\n",
	)

	importFile := writeTemp(
		t, dir, "partial.txt",
		"build {BUILD_NUM}",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"info={{imports.info}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StampInfoFiles: []string{stampPath},
	}

	err := en.Expand(
		tplPath, outPath,
		nil,
		[]string{"info=" + importFile},
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "info=build 42", string(got))
}

func TestExpand_executable_output(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(
		t, dir, "script.sh", "#!/bin/sh\necho hi\n",
	)

	outPath := filepath.Join(dir, "out.sh")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, outPath, nil, nil, true,
	)
	require.NoError(t, err)

	info, err := os.Stat(outPath)
	require.NoError(t, err)

	// Owner executable bit must be set.
	assert.NotZero(t, info.Mode()&0o100)
}

func TestExpand_unknown_tags_preserved(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(
		t, dir, "tpl.txt",
		"{{known}} and {{unknown}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, outPath,
		[]string{"known=yes"},
		nil,
		false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(
		t,
		"yes and {{unknown}}",
		string(got),
	)
}

func TestExpand_bad_variable_format(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(t, dir, "tpl.txt", "hi")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, "",
		[]string{"NOEQUALS"},
		nil,
		false,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VAR=value")
}

func TestExpand_bad_import_format(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(t, dir, "tpl.txt", "hi")

	en := templating.Engine{}

	err := en.Expand(
		tplPath, "",
		nil,
		[]string{"NOEQUALS"},
		false,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NAME=filename")
}

func TestExpand_multiple_stamp_files(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf1 := writeTemp(
		t, dir, "s1.txt", "K1 v1\n",
	)
	sf2 := writeTemp(
		t, dir, "s2.txt", "K2 v2\n",
	)

	tplPath := writeTemp(
		t, dir, "tpl.txt", "{{K1}}-{{K2}}",
	)

	outPath := filepath.Join(dir, "out.txt")

	en := templating.Engine{
		StampInfoFiles: []string{sf1, sf2},
	}

	err := en.Expand(
		tplPath, outPath, nil, nil, false,
	)
	require.NoError(t, err)

	got, err := os.ReadFile(outPath) //nolint:gosec // test file
	require.NoError(t, err)
	assert.Equal(t, "v1-v2", string(got))
}

func TestExpand_missing_stamp_file(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	tplPath := writeTemp(t, dir, "tpl.txt", "hi")

	en := templating.Engine{
		StampInfoFiles: []string{"/nonexistent/stamp.txt"},
	}

	err := en.Expand(
		tplPath, "", nil, nil, false,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expanding template")
}

func FuzzExpand(f *testing.F) {
	f.Add("Hello {{name}}!", "name", "World")
	f.Add("{{a}}{{b}}", "a", "x")
	f.Add("no tags here", "key", "val")
	f.Add("{{", "k", "v")
	f.Add("}}", "k", "v")
	f.Add("{{key}}", "key", "")
	f.Add("", "key", "val")

	f.Fuzz(func(
		t *testing.T,
		tpl string,
		key string,
		val string,
	) {
		if key == "" {
			return
		}

		dir := t.TempDir()
		tplPath := filepath.Join(dir, "tpl.txt")
		outPath := filepath.Join(dir, "out.txt")

		err := os.WriteFile(
			tplPath, []byte(tpl), 0o600,
		)
		if err != nil {
			return
		}

		en := templating.Engine{}

		// We only verify it does not panic.
		_ = en.Expand( //nolint:errcheck // fuzz: error irrelevant
			tplPath,
			outPath,
			[]string{key + "=" + val},
			nil,
			false,
		)
	})
}
