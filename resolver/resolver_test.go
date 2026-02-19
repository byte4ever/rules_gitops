package resolver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/byte4ever/rules_gitops/resolver"
)

// decodeAllDocs decodes all YAML documents from raw bytes
// into a slice of maps for structural comparison.
func decodeAllDocs(
	tb testing.TB,
	raw []byte,
) []map[string]interface{} {
	tb.Helper()

	docs, err := resolver.DecodeAllDocs(raw)
	require.NoError(tb, err)

	return docs
}

func TestResolveImages_happypath(t *testing.T) {
	t.Parallel()

	imgMap := map[string]string{
		"salist": "docker.io/rtb/sacli/cmd/salist/" +
			"image@sha256:5711bcf54511ab2fef6e08d9c" +
			"9f9ae3f3a269e66834048465cc7502adb0d489b",
		"filewatcher": "docker.io/kube/" +
			"filewatcher/image:tag",
	}

	runGoldenTest(t, "happypath", imgMap)
}

func TestResolveImages_cronworkflow(t *testing.T) {
	t.Parallel()

	imgMap := map[string]string{
		"helloworld-image": "docker.io/kube/" +
			"hello/image:tag",
	}

	runGoldenTest(t, "cwf", imgMap)
}

func TestResolveImages_flinkapp(t *testing.T) {
	t.Parallel()

	imgMap := map[string]string{
		"flinkapp-image": "docker.io/kube/" +
			"flink/image:tag",
	}

	runGoldenTest(t, "flinkapp", imgMap)
}

func TestResolveImages_zookeeper_image_map(t *testing.T) {
	t.Parallel()

	imgMap := map[string]string{
		"repository": "should-not-be-used",
	}

	runGoldenTest(t, "zk", imgMap)
}

func TestResolveImages_empty_init_containers(
	t *testing.T,
) {
	t.Parallel()

	imgMap := map[string]string{
		"helloworld-image": "docker.io/kube/" +
			"hello/image:tag",
	}

	runGoldenTest(t, "emptyinit", imgMap)
}

func TestResolveImages_missing_name(t *testing.T) {
	t.Parallel()

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  namespace: default
data:
  key: value
`

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing metadata.name")
}

func TestResolveImages_missing_kind(t *testing.T) {
	t.Parallel()

	input := `apiVersion: v1
metadata:
  name: test
data:
  key: value
`

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing kind")
}

func TestResolveImages_unresolved_image(t *testing.T) {
	t.Parallel()

	input := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  containers:
  - name: app
    image: "//bazel/target:image"
`

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved image")
}

func TestResolveImages_empty_input(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(""),
		&out,
		nil,
	)

	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestResolveImages_single_document(t *testing.T) {
	t.Parallel()

	input := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.NoError(t, err)

	docs := decodeAllDocs(t, out.Bytes())
	require.Len(t, docs, 1)
	assert.Equal(t, "ConfigMap", docs[0]["kind"])
}

func TestResolveImages_multi_document(t *testing.T) {
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

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.NoError(t, err)

	outStr := out.String()
	assert.Contains(t, outStr, "---")

	docs := decodeAllDocs(t, out.Bytes())
	require.Len(t, docs, 2)
}

func TestResolveImages_unresolved_single_container(
	t *testing.T,
) {
	t.Parallel()

	input := `apiVersion: apps/v1
kind: CronWorkFlow
metadata:
  name: test
spec:
  workflowSpec:
    templates:
      container:
        image: "//bazel/target"
`

	var out bytes.Buffer

	err := resolver.ResolveImages(
		strings.NewReader(input),
		&out,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved image")
}

func FuzzResolveImages(f *testing.F) {
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
		[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  containers:
  - name: c
    image: myimage
`),
	)
	f.Add([]byte(""))
	f.Add([]byte("---\n"))
	f.Add(
		[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  containers:
  - name: c
    image: "//bazel/target:image"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cfg
`),
	)
	f.Add(
		[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  initContainers:
  - name: init
    image: myimage
  containers:
  - name: c
    image: myimage
`),
	)

	f.Fuzz(func(t *testing.T, input []byte) {
		var out bytes.Buffer
		// We only verify it does not panic.
		_ = resolver.ResolveImages( //nolint:errcheck // fuzz: error irrelevant
			bytes.NewReader(input),
			&out,
			map[string]string{
				"myimage": "replaced:latest",
			},
		)
	})
}

// runGoldenTest reads input and expected YAML from
// testdata, runs ResolveImages, then structurally
// compares the output.
func runGoldenTest(
	tb testing.TB,
	name string,
	imgMap map[string]string,
) {
	tb.Helper()

	inputPath := filepath.Join(
		"testdata", name+".yaml",
	)
	expectedPath := filepath.Join(
		"testdata", name+".expected.yaml",
	)

	inputData, err := os.ReadFile(inputPath) //nolint:gosec // test fixture path
	require.NoError(tb, err)

	//nolint:gosec // test fixture path
	expectedData, err := os.ReadFile(
		expectedPath,
	)
	require.NoError(tb, err)

	var out bytes.Buffer

	err = resolver.ResolveImages(
		bytes.NewReader(inputData),
		&out,
		imgMap,
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
