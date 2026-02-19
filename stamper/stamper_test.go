package stamper_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/stamper"
)

// writeTemp creates a temporary file with content and
// returns its path.
func writeTemp(
	tb testing.TB,
	dir string,
	name string,
	content string,
) string {
	tb.Helper()

	pa := filepath.Join(dir, name)
	require.NoError(
		tb,
		os.WriteFile(pa, []byte(content), 0o600),
	)

	return pa
}

func TestStamp_substitutes_variables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := writeTemp(
		t, dir, "status.txt",
		"BUILD_USER alice\nGIT_SHA deadbeef\n",
	)

	got, err := stamper.Stamp(
		[]string{sf},
		"deployed by {BUILD_USER} at {GIT_SHA}",
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		"deployed by alice at deadbeef",
		got,
	)
}

func TestStamp_missing_variable_preserved(t *testing.T) {
	t.Parallel()

	got, err := stamper.Stamp(
		nil,
		"no {SUCH_VAR} here",
	)

	require.NoError(t, err)
	assert.Equal(t, "no {SUCH_VAR} here", got)
}

func TestStamp_empty_format(t *testing.T) {
	t.Parallel()

	got, err := stamper.Stamp(nil, "")

	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestStamp_multiple_stamp_files(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf1 := writeTemp(
		t, dir, "s1.txt", "K1 v1\n",
	)
	sf2 := writeTemp(
		t, dir, "s2.txt", "K2 v2\n",
	)

	got, err := stamper.Stamp(
		[]string{sf1, sf2},
		"{K1}-{K2}",
	)

	require.NoError(t, err)
	assert.Equal(t, "v1-v2", got)
}

func TestStamp_later_file_overrides_earlier(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf1 := writeTemp(
		t, dir, "s1.txt", "VER 1.0\n",
	)
	sf2 := writeTemp(
		t, dir, "s2.txt", "VER 2.0\n",
	)

	got, err := stamper.Stamp(
		[]string{sf1, sf2},
		"version={VER}",
	)

	require.NoError(t, err)
	assert.Equal(t, "version=2.0", got)
}

func TestStamp_missing_stamp_file(t *testing.T) {
	t.Parallel()

	_, err := stamper.Stamp(
		[]string{"/nonexistent/stamp.txt"},
		"hello",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading stamps")
}

func TestStamp_known_and_unknown_variable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := writeTemp(
		t, dir, "status.txt", "KNOWN val\n",
	)

	got, err := stamper.Stamp(
		[]string{sf},
		"{KNOWN} and {UNKNOWN}",
	)

	require.NoError(t, err)
	assert.Equal(t, "val and {UNKNOWN}", got)
}

func TestStamp_value_with_spaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := writeTemp(
		t, dir, "status.txt",
		"MSG hello world from CI\n",
	)

	got, err := stamper.Stamp(
		[]string{sf},
		"message={MSG}",
	)

	require.NoError(t, err)
	assert.Equal(
		t,
		"message=hello world from CI",
		got,
	)
}

func TestLoadStamps_returns_map(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := writeTemp(
		t, dir, "status.txt",
		"BUILD_USER alice\nGIT_SHA deadbeef\n",
	)

	stamps, err := stamper.LoadStamps([]string{sf})

	require.NoError(t, err)
	assert.Equal(t, "alice", stamps["BUILD_USER"])
	assert.Equal(t, "deadbeef", stamps["GIT_SHA"])
}

func TestLoadStamps_nil_files(t *testing.T) {
	t.Parallel()

	stamps, err := stamper.LoadStamps(nil)

	require.NoError(t, err)
	assert.Empty(t, stamps)
}

func TestLoadStamps_skips_malformed_lines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	sf := writeTemp(
		t, dir, "status.txt",
		"GOOD value\nBADLINE\n\nALSO_GOOD val2\n",
	)

	stamps, err := stamper.LoadStamps([]string{sf})

	require.NoError(t, err)
	assert.Len(t, stamps, 2)
	assert.Equal(t, "value", stamps["GOOD"])
	assert.Equal(t, "val2", stamps["ALSO_GOOD"])
}

func TestLoadStamps_missing_file(t *testing.T) {
	t.Parallel()

	_, err := stamper.LoadStamps(
		[]string{"/nonexistent/file.txt"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading stamps")
}

func FuzzStamp(f *testing.F) {
	f.Add("Hello {name}!", "name", "World")
	f.Add("{a}{b}", "a", "x")
	f.Add("no tags here", "key", "val")
	f.Add("{", "k", "v")
	f.Add("}", "k", "v")
	f.Add("{key}", "key", "")
	f.Add("", "key", "val")
	f.Add("{a} and {b}", "a", "{nested}")

	f.Fuzz(func(
		t *testing.T,
		format string,
		key string,
		val string,
	) {
		if key == "" {
			return
		}

		dir := t.TempDir()
		sf := filepath.Join(dir, "stamp.txt")

		err := os.WriteFile(
			sf,
			[]byte(key+" "+val+"\n"),
			0o600,
		)
		if err != nil {
			return
		}

		// We only verify it does not panic.
		_, _ = stamper.Stamp( //nolint:errcheck // fuzz: error irrelevant
			[]string{sf},
			format,
		)
	})
}
