// Package resolver walks multi-document YAML looking for
// container image references and substitutes them with
// registry URLs from an image map.
package resolver

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
)

type imageTagTransformer struct {
	images map[string]string
}

// ResolveImages reads multi-document YAML from in,
// substitutes container image references using imgMap,
// validates each document, and writes the result to out.
func ResolveImages(
	in io.Reader,
	out io.Writer,
	imgMap map[string]string,
) error {
	const errCtx = "resolving images"

	pt := imageTagTransformer{images: imgMap}
	decoder := yaml.NewDecoder(in)

	firstObj := true

	for {
		var obj map[string]interface{}

		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf(
				"%s: decoding yaml: %w",
				errCtx, err,
			)
		}

		if obj == nil {
			continue
		}

		name := extractName(obj)
		if name == "" {
			return fmt.Errorf(
				"%s: missing metadata.name in object %v",
				errCtx, obj,
			)
		}

		kind := extractKind(obj)
		if kind == "" {
			return fmt.Errorf(
				"%s: missing kind in object %v",
				errCtx, obj,
			)
		}

		if err := pt.findAndReplaceTag(obj); err != nil {
			return fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		buf, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf(
				"%s: marshaling object: %w",
				errCtx, err,
			)
		}

		if firstObj {
			firstObj = false
		} else {
			if _, err := out.Write(
				[]byte("---\n"),
			); err != nil {
				return fmt.Errorf(
					"%s: writing separator: %w",
					errCtx, err,
				)
			}
		}

		if _, err := out.Write(buf); err != nil {
			return fmt.Errorf(
				"%s: writing output: %w",
				errCtx, err,
			)
		}
	}

	return nil
}

// decodeAllDocs decodes all YAML documents from raw bytes
// into a slice of maps.
func decodeAllDocs(
	raw []byte,
) ([]map[string]interface{}, error) {
	const errCtx = "decoding all docs"

	decoder := yaml.NewDecoder(bytes.NewReader(raw))

	var docs []map[string]interface{}

	for {
		var doc map[string]interface{}

		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"%s: %w", errCtx, err,
			)
		}

		if doc == nil {
			continue
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// extractName retrieves metadata.name from a YAML object
// represented as a nested map.
func extractName(
	obj map[string]interface{},
) string {
	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return ""
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return ""
	}

	return name
}

// extractKind retrieves the kind field from a YAML object.
func extractKind(
	obj map[string]interface{},
) string {
	kind, ok := obj["kind"].(string)
	if !ok {
		return ""
	}

	return kind
}

func (pt *imageTagTransformer) findAndReplaceTag(
	obj map[string]interface{},
) error {
	// found tracks whether any container-related key was
	// seen. It is overwritten (not OR'd) on each iteration
	// to match the original algorithm: the recursion guard
	// depends only on the last checked path.
	found := false

	singlePaths := []string{"container", "spec"}
	for _, pa := range singlePaths {
		_, found = obj[pa]
		if found {
			if err := pt.updateContainer(
				obj, pa,
			); err != nil {
				return err
			}
		}
	}

	listPaths := []string{"containers", "initContainers"}
	for _, pa := range listPaths {
		_, found = obj[pa]
		if found {
			if err := pt.updateContainers(
				obj, pa,
			); err != nil {
				return err
			}
		}
	}

	if !found {
		return pt.findContainers(obj)
	}

	return nil
}

// updateContainers handles list-style containers. Note:
// the ordering (substitute first, then check //) differs
// from updateContainer (check // first). This matches the
// original inspiration code's behavior intentionally.
func (pt *imageTagTransformer) updateContainers(
	obj map[string]interface{},
	path string,
) error {
	if obj[path] == nil {
		return nil
	}

	containers, ok := obj[path].([]interface{})
	if !ok {
		return nil
	}

	for idx := range containers {
		container, ok := containers[idx].(map[string]interface{})
		if !ok {
			continue
		}

		image, found := container["image"]
		if !found {
			continue
		}

		imageName, ok := image.(string)
		if !ok {
			continue
		}

		if newName, ok := pt.images[imageName]; ok {
			container["image"] = newName
			continue
		}

		if strings.HasPrefix(imageName, "//") {
			return fmt.Errorf(
				"unresolved image found: %s",
				imageName,
			)
		}
	}

	return nil
}

func (pt *imageTagTransformer) updateContainer(
	obj map[string]interface{},
	path string,
) error {
	if obj[path] == nil {
		return nil
	}

	container, ok := obj[path].(map[string]interface{})
	if !ok {
		return nil
	}

	image, found := container["image"]
	if !found {
		return nil
	}

	imageName, ok := image.(string)
	if !ok {
		return nil
	}

	if strings.HasPrefix(imageName, "//") {
		return fmt.Errorf(
			"unresolved image found: %s",
			imageName,
		)
	}

	if newName, ok := pt.images[imageName]; ok {
		container["image"] = newName
	}

	return nil
}

func (pt *imageTagTransformer) findContainers(
	obj map[string]interface{},
) error {
	for key := range obj {
		switch typedVal := obj[key].(type) {
		case map[string]interface{}:
			if err := pt.findAndReplaceTag(
				typedVal,
			); err != nil {
				return err
			}
		case []interface{}:
			for idx := range typedVal {
				item, ok := typedVal[idx].(map[string]interface{})
				if ok {
					if err := pt.findAndReplaceTag(
						item,
					); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
