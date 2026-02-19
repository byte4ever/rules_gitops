package filter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	filter "github.com/byte4ever/rules_gitops/testing/it_manifest_filter"
)

// decodeAllDocs decodes all YAML documents from raw
// bytes into a slice of maps for structural comparison.
func decodeAllDocs(
	tb testing.TB,
	raw []byte,
) []map[string]interface{} {
	tb.Helper()

	docs, err := filter.DecodeAllDocs(raw)
	require.NoError(tb, err)

	return docs
}

func TestReplacePDWithEmptyDirs(t *testing.T) {
	t.Parallel()

	testcases := []string{
		"happypath",
		"statefulset",
		"statefulset2",
		"statefulset3",
		"certificate",
	}

	for _, tc := range testcases {
		t.Run(tc, func(t *testing.T) {
			t.Parallel()
			runGoldenTest(t, tc)
		})
	}
}

func TestReplacePDWithEmptyDirs_empty_input(
	t *testing.T,
) {
	t.Parallel()

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(""),
		&out,
	)

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestReplacePDWithEmptyDirs_missing_name(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  namespace: default
data:
  key: value
`

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(input),
		&out,
	)

	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "missing metadata.name",
	)
}

func TestReplacePDWithEmptyDirs_missing_kind(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: v1
metadata:
  name: test
data:
  key: value
`

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(input),
		&out,
	)

	require.Error(t, err)
	assert.Contains(
		t, err.Error(), "missing kind",
	)
}

func TestReplacePDWithEmptyDirs_skips_pvc(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
  - ReadWriteOnce
`

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(input),
		&out,
	)

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestReplacePDWithEmptyDirs_skips_ingress(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
spec:
  rules:
  - host: example.com
`

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(input),
		&out,
	)

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestReplacePDWithEmptyDirs_multi_document(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: cm1
data:
  key: value
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cm2
data:
  other: data
`

	var out bytes.Buffer

	err := filter.ReplacePDWithEmptyDirs(
		strings.NewReader(input),
		&out,
	)

	require.NoError(t, err)

	outStr := out.String()
	assert.Contains(t, outStr, "---")

	docs := decodeAllDocs(t, out.Bytes())
	require.Len(t, docs, 2)
}

func FuzzReplacePDWithEmptyDirs(f *testing.F) {
	f.Add(
		[]byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`),
	)
	f.Add(
		[]byte(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
  - ReadWriteOnce
`),
	)
	f.Add(
		[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: my-pvc
`),
	)
	f.Add([]byte(""))
	f.Add([]byte("---\n"))
	f.Add(
		[]byte(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ss
spec:
  template:
    metadata:
      labels:
        app: ss
    spec:
      containers:
      - name: c
        image: img
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      resources:
        requests:
          storage: 1Gi
`),
	)

	f.Fuzz(func(t *testing.T, input []byte) {
		var out bytes.Buffer
		// We only verify it does not panic.
		//nolint:errcheck // fuzz: error irrelevant
		_ = filter.ReplacePDWithEmptyDirs(
			bytes.NewReader(input),
			&out,
		)
	})
}

// runGoldenTest reads input and expected YAML from
// testdata, runs ReplacePDWithEmptyDirs, then
// structurally compares the output.
func runGoldenTest(
	tb testing.TB,
	name string,
) {
	tb.Helper()

	inputPath := filepath.Join(
		"testdata", name+".yaml",
	)
	expectedPath := filepath.Join(
		"testdata", name+".expected.yaml",
	)

	//nolint:gosec // test fixture path
	inputData, err := os.ReadFile(inputPath)
	require.NoError(tb, err)

	//nolint:gosec // test fixture path
	expectedData, err := os.ReadFile(
		expectedPath,
	)
	require.NoError(tb, err)

	var out bytes.Buffer

	err = filter.ReplacePDWithEmptyDirs(
		bytes.NewReader(inputData),
		&out,
	)
	require.NoError(tb, err)

	expectedDocs := decodeAllDocs(tb, expectedData)
	actualDocs := decodeAllDocs(tb, out.Bytes())

	require.Equal(
		tb,
		len(expectedDocs),
		len(actualDocs),
		"document count mismatch",
	)

	for idx := range expectedDocs {
		assert.Equal(
			tb,
			expectedDocs[idx],
			actualDocs[idx],
			"document %d mismatch",
			idx,
		)
	}
}
