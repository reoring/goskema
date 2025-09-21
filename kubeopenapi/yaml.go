package kubeopenapi

import (
	"bytes"
	"errors"
	"io"

	goskema "github.com/reoring/goskema"
	"gopkg.in/yaml.v3"
)

// ImportYAMLForCRDKind scans a multi-document YAML (e.g., CRD bundle) and imports
// the first CustomResourceDefinition matching the given spec.names.kind.
// If no matching CRD is found, returns an error.
func ImportYAMLForCRDKind(data []byte, kind string, opts Options) (goskema.Schema[map[string]any], Diag, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node any
		if err := dec.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			// return diag with error
			return nil, &simpleDiag{}, err
		}
		m := yamlAnyToStringMap(node)
		if m == nil {
			continue
		}
		if k, _ := m["kind"].(string); k != "CustomResourceDefinition" {
			continue
		}
		if spec, ok := m["spec"].(map[string]any); ok {
			if names, ok := spec["names"].(map[string]any); ok {
				if k2, _ := names["kind"].(string); k2 == kind {
					return Import(m, opts)
				}
			}
		}
	}
	return nil, &simpleDiag{}, errors.New("kubeopenapi: CRD kind not found in YAML bundle")
}

// ImportYAMLForCRDName scans a multi-document YAML and imports the CRD
// with given metadata.name.
func ImportYAMLForCRDName(data []byte, name string, opts Options) (goskema.Schema[map[string]any], Diag, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node any
		if err := dec.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, &simpleDiag{}, err
		}
		m := yamlAnyToStringMap(node)
		if m == nil {
			continue
		}
		if k, _ := m["kind"].(string); k != "CustomResourceDefinition" {
			continue
		}
		if meta, ok := m["metadata"].(map[string]any); ok {
			if n, _ := meta["name"].(string); n == name {
				return Import(m, opts)
			}
		}
	}
	return nil, &simpleDiag{}, errors.New("kubeopenapi: CRD name not found in YAML bundle")
}

// yamlAnyToStringMap converts YAML-decoded values (which may contain map[any]any)
// into JSON-like map[string]any recursively. Non-map roots return nil.
func yamlAnyToStringMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[k] = yamlNormalizeValue(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			out[ks] = yamlNormalizeValue(vv)
		}
		return out
	default:
		return nil
	}
}

func yamlNormalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any, map[any]any:
		return yamlAnyToStringMap(t)
	case []any:
		arr := make([]any, len(t))
		for i := range t {
			arr[i] = yamlNormalizeValue(t[i])
		}
		return arr
	default:
		return v
	}
}
