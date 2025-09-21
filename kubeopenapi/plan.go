package kubeopenapi

import (
	"context"
	"encoding/json"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/dsl"
)

// propertyPlan represents a reflection plan for a single field.
type propertyPlan struct {
	name    string
	adapter dsl.AnyAdapter
	refine  func(ctx context.Context, m map[string]any) error
}

// planIntOrStringProperty builds a propertyPlan for x-kubernetes-int-or-string if present.
func planIntOrStringProperty(name string, ps map[string]any) (propertyPlan, bool) {
	if !isIntOrString(ps) {
		return propertyPlan{}, false
	}
	ad := dsl.SchemaOf[json.Number](dsl.NumberJSON().CoerceFromString())
	if nullableTrue(ps) {
		ad = dsl.Nullable(ad)
	}
	return propertyPlan{name: name, adapter: ad}, true
}

// planObjectProperty handles object-type properties including patternProperties and additionalProperties schema.
func planObjectProperty(name string, ps map[string]any, d *simpleDiag) (propertyPlan, bool) {
	t, _ := ps["type"].(string)
	if t != "object" {
		return propertyPlan{}, false
	}
	// Detect propertyNames (approximation: pattern only)
	var propNamesRef func(ctx context.Context, m map[string]any) error
	if pn, ok := ps["propertyNames"].(map[string]any); ok {
		if patt, _ := pn["pattern"].(string); patt != "" {
			propNamesRef = buildKeyPatternRefiner(name, patt)
		}
	}
	// Nested object with explicit properties: build a nested object adapter recursively.
	// Keep lenient at this level (do not enforce required) so that minimal spec objects
	// like spec:{} pass for CRDs that mark nested fields required.
	if pm, ok := ps["properties"].(map[string]any); ok && len(pm) > 0 {
		if ad, ok := buildObjectAdapterFromProperties(ps, d); ok {
			if nullableTrue(ps) {
				ad = dsl.Nullable(ad)
			}
			return propertyPlan{name: name, adapter: ad, refine: propNamesRef}, true
		}
	}
	if ppm, ok := ps["patternProperties"].(map[string]any); ok && len(ppm) >= 1 {
		// Build adapter and refine under patternProperties with multi-pattern support.
		// 1) Decide adapter for map values:
		//    - If all pattern value types are the same and compatible with additionalProperties schema type (if present),
		//      build Map[...] with that element; otherwise MapAny.
		var (
			common      map[string]any
			commonType  string
			allSameType = true
			patterns    []string
			patTypeMap  = make(map[string]string)
		)
		for patt, raw := range ppm {
			psch, _ := raw.(map[string]any)
			patterns = append(patterns, patt)
			if t2, _ := psch["type"].(string); t2 != "" {
				patTypeMap[patt] = t2
				if commonType == "" {
					commonType = t2
					common = psch
				} else if commonType != t2 {
					allSameType = false
				}
			} else {
				allSameType = false
			}
		}
		var mad dsl.AnyAdapter
		var apMap map[string]any
		if apm, ok := ps["additionalProperties"].(map[string]any); ok {
			apMap = apm
		}
		if allSameType && common != nil {
			if apMap != nil {
				apTypeCandidate, _ := apMap["type"].(string)
				if apTypeCandidate != "" && apTypeCandidate != commonType {
					mad = dsl.SchemaOf[map[string]any](dsl.MapAny())
				} else {
					mad, _ = buildMapAdapterFromAdditional(common, d)
				}
			} else {
				mad, _ = buildMapAdapterFromAdditional(common, d)
			}
		} else {
			mad = dsl.SchemaOf[map[string]any](dsl.MapAny())
			d.warnf("patternProperties with heterogeneous value schemas treated as MapAny for values")
		}
		// 2) Key enforcement unless additionalProperties permits unmatched
		var ref func(ctx context.Context, m map[string]any) error
		enforce := true
		if apb, ok := ps["additionalProperties"].(bool); ok && apb {
			enforce = false
		}
		if _, ok := ps["additionalProperties"].(map[string]any); ok {
			enforce = false
		}
		if enforce {
			if len(patterns) == 1 {
				ref = buildKeyPatternRefiner(name, patterns[0])
			} else {
				ref = buildKeyPatternsRefiner(name, patterns)
			}
		}
		// Chain propertyNames refiner first if present
		if propNamesRef != nil {
			if ref == nil {
				ref = propNamesRef
			} else {
				keyRef := ref
				ref = func(ctx context.Context, m map[string]any) error {
					if err := propNamesRef(ctx, m); err != nil {
						return err
					}
					return keyRef(ctx, m)
				}
			}
		}
		// 3) Value type refine with multi-pattern semantics
		var apType string
		if apm, ok := ps["additionalProperties"].(map[string]any); ok {
			apType, _ = apm["type"].(string)
		}
		vtRef := buildPatternPropertiesValueTypesRefiner(name, patTypeMap, apType)
		// Chain: key ref (optional) -> value-type ref
		var chained func(ctx context.Context, m map[string]any) error
		if ref != nil {
			keyRef := ref
			chained = func(ctx context.Context, m map[string]any) error {
				if err := keyRef(ctx, m); err != nil {
					return err
				}
				return vtRef(ctx, m)
			}
		} else {
			chained = vtRef
		}
		pp := propertyPlan{name: name, adapter: mad, refine: chained}
		if nullableTrue(ps) {
			pp.adapter = dsl.Nullable(pp.adapter)
		}
		return pp, true
	}
	if ap, ok := ps["additionalProperties"].(map[string]any); ok {
		ad, handled := buildMapAdapterFromAdditional(ap, d)
		if nullableTrue(ps) {
			ad = dsl.Nullable(ad)
		}
		if handled {
			return propertyPlan{name: name, adapter: ad, refine: propNamesRef}, true
		}
	}
	// Fallback: propertyNames only case on an object without properties/patternProperties/additionalProperties.
	if propNamesRef != nil {
		ad := dsl.SchemaOf[map[string]any](dsl.MapAny())
		if nullableTrue(ps) {
			ad = dsl.Nullable(ad)
		}
		return propertyPlan{name: name, adapter: ad, refine: propNamesRef}, true
	}
	return propertyPlan{}, false
}

// planMinimalProperty handles the fallback minimal adapter and nullable.
func planMinimalProperty(name string, ps map[string]any, d *simpleDiag, opts Options) propertyPlan {
	// Prefer dedicated adapter for anyOf when present (pick first typed branch)
	if anys, ok := ps["anyOf"].([]any); ok && len(anys) > 0 {
		if ad2, ok2 := adapterForAnyOf(anys); ok2 {
			ad := ad2
			if nullableTrue(ps) {
				ad = dsl.Nullable(ad)
			}
			pp := propertyPlan{name: name, adapter: ad}
			var branches []map[string]any
			for _, it := range anys {
				if m, _ := it.(map[string]any); m != nil {
					branches = append(branches, m)
				}
			}
			if len(branches) > 0 {
				strat := opts.Ambiguity
				if strat == 0 {
					strat = AmbiguityError
				}
				pp.refine = buildAnyOfRefiner(name, branches, strat)
			}
			return pp
		}
	}
	ad, jsWarn := minimalAdapterFor(ps)
	if jsWarn != "" {
		d.warnf("%s", jsWarn)
	}
	if nullableTrue(ps) {
		ad = dsl.Nullable(ad)
	}
	pp := propertyPlan{name: name, adapter: ad}
	// Attach anyOf/oneOf/if-then-else refines (MVP: anyOf only)
	if anys, ok := ps["anyOf"].([]any); ok && len(anys) > 0 {
		var branches []map[string]any
		for _, it := range anys {
			if m, _ := it.(map[string]any); m != nil {
				branches = append(branches, m)
			}
		}
		if len(branches) > 0 {
			// strategy from opts
			strat := opts.Ambiguity
			if strat == 0 {
				strat = AmbiguityError
			}
			// We cannot reach Options here easily; stick to default for MVP.
			ref := buildAnyOfRefiner(name, branches, strat)
			if ref != nil {
				pp.refine = ref
			}
		}
	}
	return pp
}

// adapterForAnyOf picks an AnyAdapter based on the first typed branch.
func adapterForAnyOf(branches []any) (dsl.AnyAdapter, bool) {
	for _, it := range branches {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		if t, _ := m["type"].(string); t != "" {
			switch t {
			case "string":
				return dsl.StringOf[string](), true
			case "boolean":
				return dsl.BoolOf[bool](), true
			case "number", "integer":
				return dsl.SchemaOf[json.Number](dsl.NumberJSON()), true
			case "object":
				return dsl.SchemaOf[map[string]any](dsl.MapAny()), true
			case "array":
				return dsl.ArrayOf[map[string]any](dsl.MapAny()), true
			}
		}
	}
	return dsl.AnyAdapter{}, false
}

// applyListTypeRefine attaches list-type uniqueness refine if configured (and not already set).
func applyListTypeRefine(pp propertyPlan, name string, ps map[string]any) propertyPlan {
	if pp.refine != nil {
		return pp
	}
	if lt, ok := ps["x-kubernetes-list-type"].(string); ok {
		var keys []string
		if lmk, ok := ps["x-kubernetes-list-map-keys"].([]any); ok {
			for _, v := range lmk {
				if s, ok := v.(string); ok {
					keys = append(keys, s)
				}
			}
		}
		pp.refine = buildListUniquenessRefiner(name, lt, keys)
	}
	return pp
}

// buildObjectSchemaFromProperties constructs a nested object schema from an OpenAPI object schema
// with explicit properties/required. Supports recursive nested objects (local $defs already expanded upstream).
func buildObjectSchemaFromProperties(ps map[string]any, d *simpleDiag) (goskema.Schema[map[string]any], bool) {
	b := dsl.Object()
	pm, _ := ps["properties"].(map[string]any)
	for pname, raw := range pm {
		psch, _ := raw.(map[string]any)
		if psch == nil {
			continue
		}
		// int-or-string first
		if ad, ok := func() (dsl.AnyAdapter, bool) {
			if isIntOrString(psch) {
				ad := dsl.SchemaOf[json.Number](dsl.NumberJSON().CoerceFromString())
				if nullableTrue(psch) {
					ad = dsl.Nullable(ad)
				}
				return ad, true
			}
			return dsl.AnyAdapter{}, false
		}(); ok {
			b.Field(pname, ad)
			continue
		}
		// object/map/array/primitives via minimalAdapterFor + list refine
		ad, _ := minimalAdapterFor(psch)
		// nullable wrapper
		if nullableTrue(psch) {
			ad = dsl.Nullable(ad)
		}
		b.Field(pname, ad)
		// attach list-type uniqueness refine at nested level when applicable
		if lt, ok := psch["x-kubernetes-list-type"].(string); ok {
			var keys []string
			if lmk, ok := psch["x-kubernetes-list-map-keys"].([]any); ok {
				for _, v := range lmk {
					if s, ok := v.(string); ok {
						keys = append(keys, s)
					}
				}
			}
			if ref := buildListUniquenessRefiner(pname, lt, keys); ref != nil {
				b.Refine("list-uniqueness:"+pname, ref)
			}
		}
	}
	// unknown policy: honor additionalProperties
	switch ap := ps["additionalProperties"].(type) {
	case bool:
		if ap {
			b.UnknownStrip() // allow unknown (strip at runtime)
		} else {
			b.UnknownStrict()
		}
	case map[string]any:
		// if schema present, accept unknown but drop (approximation)
		b.UnknownStrip()
		if d != nil {
			d.warnf("additionalProperties as schema is treated as permissive for MVP")
		}
	default:
		// default prune
		b.UnknownStrip()
	}
	return b.MustBuild(), true
}

// buildObjectSchemaFromPropertiesWithRequired is like buildObjectSchemaFromProperties
// but additionally applies required fields listed in ps["required"]. Intended for
// nested array item objects where enforcing required is desirable, while keeping
// root-nested objects lenient for minimal spec acceptance.
func buildObjectSchemaFromPropertiesWithRequired(ps map[string]any, d *simpleDiag) (goskema.Schema[map[string]any], bool) {
	_, ok := buildObjectSchemaFromProperties(ps, d)
	if !ok {
		return nil, false
	}
	// Rebuild with required (object builder cannot be modified post-build),
	// so reconstruct to apply required. We duplicate logic to honor unknown policy.
	b := dsl.Object()
	pm, _ := ps["properties"].(map[string]any)
	for pname, raw := range pm {
		psch, _ := raw.(map[string]any)
		if psch == nil {
			continue
		}
		// int-or-string first
		if ad, ok := func() (dsl.AnyAdapter, bool) {
			if isIntOrString(psch) {
				ad := dsl.SchemaOf(dsl.NumberJSON().CoerceFromString())
				if nullableTrue(psch) {
					ad = dsl.Nullable(ad)
				}
				return ad, true
			}
			return dsl.AnyAdapter{}, false
		}(); ok {
			b.Field(pname, ad)
			continue
		}
		// Prefer building nested object schemas explicitly to enforce child required when present.
		if t2, _ := psch["type"].(string); t2 == "object" {
			if pm2, ok := psch["properties"].(map[string]any); ok && len(pm2) > 0 {
				var ns goskema.Schema[map[string]any]
				var ok2 bool
				if _, hasReq := psch["required"].([]any); hasReq {
					ns, ok2 = buildObjectSchemaFromPropertiesWithRequired(psch, d)
				} else {
					ns, ok2 = buildObjectSchemaFromProperties(psch, d)
				}
				if ok2 {
					ad := dsl.SchemaOf[map[string]any](ns)
					if nullableTrue(psch) {
						ad = dsl.Nullable(ad)
					}
					b.Field(pname, ad)
					// done with this field
					continue
				}
			}
		}
		ad, _ := minimalAdapterFor(psch)
		if nullableTrue(psch) {
			ad = dsl.Nullable(ad)
		}
		b.Field(pname, ad)
		if lt, ok := psch["x-kubernetes-list-type"].(string); ok {
			var keys []string
			if lmk, ok := psch["x-kubernetes-list-map-keys"].([]any); ok {
				for _, v := range lmk {
					if s, ok := v.(string); ok {
						keys = append(keys, s)
					}
				}
			}
			if ref := buildListUniquenessRefiner(pname, lt, keys); ref != nil {
				b.Refine("list-uniqueness:"+pname, ref)
			}
		}
	}
	if names := extractRequiredNames(ps); len(names) > 0 {
		b.Require(names...)
	}
	switch ap := ps["additionalProperties"].(type) {
	case bool:
		if ap {
			b.UnknownStrip()
		} else {
			b.UnknownStrict()
		}
	case map[string]any:
		b.UnknownStrip()
		if d != nil {
			d.warnf("additionalProperties as schema is treated as permissive for MVP")
		}
	default:
		b.UnknownStrip()
	}
	return b.MustBuild(), true
}

// buildObjectAdapterFromProperties constructs a nested object adapter from an OpenAPI object schema
// with explicit properties/required. Limited to one level (no deep $ref resolution here).
func buildObjectAdapterFromProperties(ps map[string]any, d *simpleDiag) (dsl.AnyAdapter, bool) {
	if s, ok := buildObjectSchemaFromProperties(ps, d); ok {
		return dsl.SchemaOf[map[string]any](s), true
	}
	return dsl.AnyAdapter{}, false
}
