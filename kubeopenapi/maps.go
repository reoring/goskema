package kubeopenapi

import (
	"encoding/json"

	"github.com/reoring/goskema/dsl"
)

// buildMapAdapterFromAdditional builds a Map[...] AnyAdapter from additionalProperties item schema.
// Returns (adapter, handled). Handles primitive cases (string, boolean, number/integer) and falls back to MapAny.
func buildMapAdapterFromAdditional(ap map[string]any, d *simpleDiag) (dsl.AnyAdapter, bool) {
	if t, _ := ap["type"].(string); t != "" {
		switch t {
		case "string":
			return dsl.MapOf[string](dsl.String()), true
		case "boolean":
			return dsl.MapOf[bool](dsl.Bool()), true
		case "number", "integer":
			return dsl.MapOf[json.Number](dsl.NumberJSON()), true
		case "object":
			// MVP: nested object maps as MapAny (no recursion yet)
			return dsl.MapOf[map[string]any](dsl.MapAny()), true
		default:
			// unknown type → MapAny with warning
			d.warnf("additionalProperties: unknown item type %q treated as any", t)
			return dsl.MapOf[map[string]any](dsl.MapAny()), true
		}
	}
	// no explicit type → treat as any with warning
	d.warnf("additionalProperties: schema without type treated as any")
	return dsl.MapOf[map[string]any](dsl.MapAny()), true
}
