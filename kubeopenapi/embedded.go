package kubeopenapi

import (
	"context"
	"strconv"

	goskema "github.com/reoring/goskema"
)

// buildEmbeddedResourcePresenceRefiner returns a refine that enforces minimal presence
// checks for x-kubernetes-embedded-resource.
//
// Rules (minimal):
// - apiVersion: required, string
// - kind:       required, string
// - metadata:   required, object (map)
//
// It accepts either an object value or an array of objects at the target field.
func buildEmbeddedResourcePresenceRefiner(fieldName string) func(ctx context.Context, m map[string]any) error {
	return func(ctx context.Context, m map[string]any) error {
		v, ok := m[fieldName]
		if !ok || v == nil {
			// Outer field presence is governed by required; do nothing here.
			return nil
		}
		var iss goskema.Issues
		switch t := v.(type) {
		case map[string]any:
			iss = checkOneEmbeddedResource("/"+fieldName, t, iss)
		case []map[string]any:
			for i, el := range t {
				iss = checkOneEmbeddedResource("/"+fieldName+"/"+strconv.Itoa(i), el, iss)
			}
		case []any:
			for i, el := range t {
				if mv, ok := el.(map[string]any); ok {
					iss = checkOneEmbeddedResource("/"+fieldName+"/"+strconv.Itoa(i), mv, iss)
				}
			}
		default:
			// Non-object/array values are validated elsewhere by type; ignore here.
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

func checkOneEmbeddedResource(basePath string, mv map[string]any, iss goskema.Issues) goskema.Issues {
	if _, ok := mv["apiVersion"]; !ok {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/apiVersion", Code: goskema.CodeRequired, Message: "required for embedded resource"})
	} else {
		if _, ok := mv["apiVersion"].(string); !ok {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/apiVersion", Code: goskema.CodeInvalidType, Message: "apiVersion must be string"})
		}
	}
	if _, ok := mv["kind"]; !ok {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/kind", Code: goskema.CodeRequired, Message: "required for embedded resource"})
	} else {
		if _, ok := mv["kind"].(string); !ok {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/kind", Code: goskema.CodeInvalidType, Message: "kind must be string"})
		}
	}
	if _, ok := mv["metadata"]; !ok {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/metadata", Code: goskema.CodeRequired, Message: "required for embedded resource"})
	} else {
		if _, ok := mv["metadata"].(map[string]any); !ok {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: basePath + "/metadata", Code: goskema.CodeInvalidType, Message: "metadata must be object"})
		}
	}
	return iss
}

// applyEmbeddedResourceRefine attaches presence checks for x-kubernetes-embedded-resource
// when enabled via Options. It recognizes the extension on the field schema or on items
// when the field is an array.
func applyEmbeddedResourceRefine(pp propertyPlan, name string, ps map[string]any, opts Options) propertyPlan {
	if !opts.EnableEmbeddedChecks {
		return pp
	}
	// field-level flag
	if b, ok := ps["x-kubernetes-embedded-resource"].(bool); !ok || !b {
		// items-level flag (array of embedded resources)
		if it, ok := ps["items"].(map[string]any); ok {
			if bb, ok2 := it["x-kubernetes-embedded-resource"].(bool); !ok2 || !bb {
				return pp
			}
		} else {
			return pp
		}
	}
	ref := buildEmbeddedResourcePresenceRefiner(name)
	if ref == nil {
		return pp
	}
	if pp.refine == nil {
		pp.refine = ref
		return pp
	}
	prev := pp.refine
	pp.refine = func(ctx context.Context, m map[string]any) error {
		if err := prev(ctx, m); err != nil {
			return err
		}
		return ref(ctx, m)
	}
	return pp
}
