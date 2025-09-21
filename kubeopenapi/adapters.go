package kubeopenapi

import (
	"encoding/json"

	"github.com/reoring/goskema/dsl"
)

// minimalAdapterFor returns an AnyAdapter for basic primitive/object/array.
func minimalAdapterFor(ps map[string]any) (dsl.AnyAdapter, string) {
	// int-or-string is handled earlier
	t, _ := ps["type"].(string)
	switch t {
	case "string":
		return dsl.StringOf[string](), ""
	case "boolean":
		return dsl.BoolOf[bool](), ""
	case "number", "integer":
		return dsl.SchemaOf[json.Number](dsl.NumberJSON()), ""
	case "object":
		// if explicit properties are present, try to build nested object adapter
		if pm, ok := ps["properties"].(map[string]any); ok && len(pm) > 0 {
			// For nested object properties, keep lenient (do not enforce required) to allow minimal spec {}.
			if ad, ok := buildObjectAdapterFromProperties(ps, nil); ok {
				return ad, ""
			}
		}
		// fallback to MapAny
		return dsl.SchemaOf[map[string]any](dsl.MapAny()), "nested object treated as MapAny (MVP)"
	case "array":
		// Support single items schema
		if it, ok := ps["items"].(map[string]any); ok {
			// special-case int-or-string in items
			if isIntOrString(it) {
				return dsl.ArrayOf[json.Number](dsl.NumberJSON().CoerceFromString()), "array items int-or-string as json.Number (coerce from string)"
			}
			return arrayAdapterFromItems(it)
		}
		// Accept arrays without items loosely as Any
		return dsl.ArrayOf[map[string]any](dsl.MapAny()), "array without items treated as Array<MapAny>"
	default:
		return dsl.SchemaOf[map[string]any](dsl.MapAny()), "unknown type treated as MapAny (MVP)"
	}
}

func isIntOrString(ps map[string]any) bool {
	if b, ok := ps["x-kubernetes-int-or-string"].(bool); ok && b {
		return true
	}
	return false
}

// nullableTrue reports whether OpenAPI 3.0 style nullable is enabled.
func nullableTrue(ps map[string]any) bool {
	if b, ok := ps["nullable"].(bool); ok && b {
		return true
	}
	return false
}

// arrayAdapterFromItems builds an Array adapter from a single-schema items definition.
func arrayAdapterFromItems(items map[string]any) (dsl.AnyAdapter, string) {
	t, _ := items["type"].(string)
	switch t {
	case "string":
		return dsl.ArrayOf[string](dsl.String()), ""
	case "boolean":
		return dsl.ArrayOf[bool](dsl.Bool()), ""
	case "number", "integer":
		return dsl.ArrayOf[json.Number](dsl.NumberJSON()), ""
	case "object":
		// When items is an object with properties, recursively build nested structure
		if pm, ok := items["properties"].(map[string]any); ok && len(pm) > 0 {
			if _, reqPresent := items["required"].([]any); reqPresent {
				if s, ok := buildObjectSchemaFromPropertiesWithRequired(items, nil); ok {
					return dsl.ArrayOf(s), "array items object (with required)"
				}
			}
			if s, ok := buildObjectSchemaFromProperties(items, nil); ok {
				return dsl.ArrayOf(s), "array items object built as nested object"
			}
		}
		// Avoid recursion for MVP and fallback to MapAny
		return dsl.ArrayOf(dsl.MapAny()), "array items object treated as MapAny"
	default:
		return dsl.ArrayOf(dsl.MapAny()), "array items unknown treated as MapAny"
	}
}
