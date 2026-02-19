// Package filter transforms Kubernetes manifests for
// integration testing by replacing persistent storage
// references with ephemeral volumes and adjusting
// certificate issuers.
package filter

import (
	"bytes"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

// ReplacePDWithEmptyDirs reads multi-document YAML,
// filters out PVC and Ingress objects, replaces PVC
// volume sources with emptyDir, processes StatefulSet
// volumeClaimTemplates, and replaces letsencrypt-prod
// certificate issuers with letsencrypt-staging.
func ReplacePDWithEmptyDirs(
	in io.Reader,
	out io.Writer,
) error {
	const errCtx = "replacing PD with empty dirs"

	decoder := yaml.NewDecoder(in)
	firstObj := true

	// Process each YAML document in the stream.
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

		// Skip PVC and Ingress objects entirely.
		if kind == "PersistentVolumeClaim" {
			continue
		}

		if kind == "Ingress" {
			continue
		}

		apiVersion := extractAPIVersion(obj)

		// Handle special object types that need
		// additional transformations.
		if kind == "StatefulSet" &&
			apiVersion == "apps/v1" {
			processStatefulSet(obj)
		}

		if kind == "Certificate" {
			findAndReplaceIssuerName(obj)
		}

		// Replace any remaining PVC volume refs
		// with emptyDir across the entire object.
		findAndReplacePVC(obj)

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

// decodeAllDocs decodes all YAML documents from raw
// bytes into a slice of maps.
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

// extractName retrieves metadata.name from a YAML
// object represented as a nested map.
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

// extractKind retrieves the kind field from a YAML
// object.
func extractKind(
	obj map[string]interface{},
) string {
	kind, ok := obj["kind"].(string)
	if !ok {
		return ""
	}

	return kind
}

// extractAPIVersion retrieves the apiVersion field from
// a YAML object.
func extractAPIVersion(
	obj map[string]interface{},
) string {
	v, ok := obj["apiVersion"].(string)
	if !ok {
		return ""
	}

	return v
}

// findAndReplacePVC replaces persistentVolumeClaim
// volume sources with emptyDir in the given map. When a
// persistentVolumeClaim key is found, it is deleted and
// replaced with an empty emptyDir entry.
func findAndReplacePVC(obj map[string]interface{}) {
	_, found := obj["persistentVolumeClaim"]
	if found {
		delete(obj, "persistentVolumeClaim")

		obj["emptyDir"] = map[string]interface{}{}

		return
	}

	findPVC(obj)
}

// findPVC recursively walks maps and slices looking for
// persistentVolumeClaim keys to replace.
func findPVC(obj map[string]interface{}) {
	for key := range obj {
		switch typedVal := obj[key].(type) {
		case map[string]interface{}:
			findAndReplacePVC(typedVal)
		case []interface{}:
			for idx := range typedVal {
				item, ok := typedVal[idx].(map[string]interface{})
				if ok {
					findAndReplacePVC(item)
				}
			}
		}
	}
}

// processStatefulSet converts volumeClaimTemplates into
// emptyDir volumes, removes the volumeClaimTemplates
// field, and deletes the status field.
func processStatefulSet(
	obj map[string]interface{},
) {
	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		return
	}

	vctRaw, ok := spec["volumeClaimTemplates"]
	if !ok {
		return
	}

	vctSlice, ok := vctRaw.([]interface{})
	if !ok {
		return
	}

	if len(vctSlice) == 0 {
		return
	}

	// Navigate to the pod template spec where the
	// volumes list lives.
	templateSpec := navigateToTemplateSpec(spec)
	if templateSpec == nil {
		return
	}

	volumes := extractVolumes(templateSpec)

	// Build index of existing volumes by name.
	existingVolumes := make(map[string]int)
	for idx, volRaw := range volumes {
		volMap, ok := volRaw.(map[string]interface{})
		if !ok {
			continue
		}

		volName, ok := volMap["name"].(string)
		if !ok {
			continue
		}

		existingVolumes[volName] = idx
	}

	// Convert each volumeClaimTemplate to an
	// emptyDir volume.
	for _, vctItem := range vctSlice {
		vctMap, ok := vctItem.(map[string]interface{})
		if !ok {
			continue
		}

		tplName := extractName(vctMap)
		if tplName == "" {
			continue
		}

		storage := extractStorageRequest(vctMap)

		// Set sizeLimit when a storage request was
		// specified in the claim template.
		emptyDir := map[string]interface{}{}
		if storage != "" {
			emptyDir["sizeLimit"] = storage
		}

		vol := map[string]interface{}{
			"name":     tplName,
			"emptyDir": emptyDir,
		}

		if idx, ok := existingVolumes[tplName]; ok {
			volumes[idx] = vol
		} else {
			volumes = append(volumes, vol)
		}
	}

	// Update volumes and clean up the StatefulSet
	// by removing volumeClaimTemplates and status.
	templateSpec["volumes"] = volumes

	delete(spec, "volumeClaimTemplates")
	delete(obj, "status")
}

// navigateToTemplateSpec safely navigates to
// spec.template.spec in a StatefulSet map.
func navigateToTemplateSpec(
	spec map[string]interface{},
) map[string]interface{} {
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil
	}

	tplSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	return tplSpec
}

// extractVolumes retrieves the volumes slice from a
// pod spec map, returning an empty slice if none exist.
func extractVolumes(
	tplSpec map[string]interface{},
) []interface{} {
	volsRaw, ok := tplSpec["volumes"]
	if !ok {
		return []interface{}{}
	}

	vols, ok := volsRaw.([]interface{})
	if !ok {
		return []interface{}{}
	}

	return vols
}

// extractStorageRequest retrieves the storage value from
// spec.resources.requests.storage in a
// volumeClaimTemplate.
func extractStorageRequest(
	vct map[string]interface{},
) string {
	spec, ok := vct["spec"].(map[string]interface{})
	if !ok {
		return ""
	}

	resources, ok := spec["resources"].(map[string]interface{})
	if !ok {
		return ""
	}

	requests, ok := resources["requests"].(map[string]interface{})
	if !ok {
		return ""
	}

	storage, ok := requests["storage"].(string)
	if !ok {
		return ""
	}

	return storage
}

// findAndReplaceIssuerName navigates to
// spec.issuerRef.name and replaces letsencrypt-prod
// with letsencrypt-staging.
func findAndReplaceIssuerName(
	obj map[string]interface{},
) {
	spec, ok := obj["spec"].(map[string]interface{})
	if !ok {
		return
	}

	issuerRef, ok := spec["issuerRef"].(map[string]interface{})
	if !ok {
		return
	}

	if issuerRef["name"] == "letsencrypt-prod" {
		issuerRef["name"] = "letsencrypt-staging"
	}
}
