package kubeopenapi

import (
	"encoding/json"
	"errors"
	"fmt"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/dsl"
)

// Import compiles a minimal subset of OpenAPI v3 schema (object/properties/required/unknown) into a goskema schema.
// The input can be either a decoded map[string]any or raw JSON bytes.
func Import(schema any, opts Options) (goskema.Schema[map[string]any], Diag, error) {
	d := &simpleDiag{}
	if opts.Profile == "" {
		opts.Profile = ProfileStructuralV1
	}
	if schema == nil {
		return nil, d, errors.New("kubeopenapi: nil schema")
	}
	var root map[string]any
	switch t := schema.(type) {
	case []byte:
		if err := json.Unmarshal(t, &root); err != nil {
			return nil, d, fmt.Errorf("kubeopenapi: invalid JSON: %w", err)
		}
	case map[string]any:
		root = t
	default:
		// try json.Marshaler style
		b, err := json.Marshal(t)
		if err != nil {
			return nil, d, fmt.Errorf("kubeopenapi: cannot marshal input: %w", err)
		}
		if err := json.Unmarshal(b, &root); err != nil {
			return nil, d, fmt.Errorf("kubeopenapi: invalid marshaled JSON: %w", err)
		}
	}

	// Accept direct schema (openAPIV3Schema) or unwrap CRD root (spec.versions[].schema.openAPIV3Schema)
	if spec, ok := root["openAPIV3Schema"].(map[string]any); ok {
		root = spec
	} else if unwrapped := unwrapCRDSchema(root); unwrapped != nil {
		root = unwrapped
	}

	// Index local $defs
	defs := extractDefs(root)

	// Resolve local $ref recursively (properties/items only for MVP)
	visited := make(map[string]bool)
	resolveRefsInPlace(root, defs, d, visited)

	s, err := importObject(root, opts, d)
	return s, d, err
}

// unwrapCRDSchema tries to extract openAPIV3Schema from a Kubernetes CRD document.
// It looks for spec.versions[].schema.openAPIV3Schema (preferring served=true),
// then falls back to spec.validation.openAPIV3Schema for legacy specs.
func unwrapCRDSchema(root map[string]any) map[string]any {
	// spec.versions[].schema.openAPIV3Schema
	if spec, ok := root["spec"].(map[string]any); ok {
		if vers, ok := spec["versions"].([]any); ok {
			var firstFound map[string]any
			for _, v := range vers {
				vm, _ := v.(map[string]any)
				if vm == nil {
					continue
				}
				served := true
				if sv, ok := vm["served"].(bool); ok {
					served = sv
				}
				if sch, ok := vm["schema"].(map[string]any); ok {
					if oas, ok := sch["openAPIV3Schema"].(map[string]any); ok {
						if served {
							return oas
						}
						if firstFound == nil {
							firstFound = oas
						}
					}
				}
			}
			if firstFound != nil {
				return firstFound
			}
		}
		// legacy: spec.validation.openAPIV3Schema
		if val, ok := spec["validation"].(map[string]any); ok {
			if oas, ok := val["openAPIV3Schema"].(map[string]any); ok {
				return oas
			}
		}
	}
	return nil
}

// NOTE: helper implementations for $ref are in refs.go

func importObject(doc map[string]any, opts Options, d *simpleDiag) (goskema.Schema[map[string]any], error) {
	// Assume type=object (minimal implementation)
	warnNonObjectRoot(doc, d)

	b := dsl.Object()

	// Apply handling policy for unknown fields
	mode, passthrough := planUnknownBehavior(doc, d)
	switch mode {
	case UnknownPreserve:
		// Temporary: stash passthrough destination into a reserved field (must exist)
		b.Field("_unknown", dsl.SchemaOf[map[string]any](dsl.MapAny()))
		b.UnknownPassthrough(passthrough)
	case UnknownStrict:
		b.UnknownStrict()
	default: // equivalent to UnknownPrune
		b.UnknownStrip()
	}

	// Reflect properties
	for _, p := range planProperties(doc, d, opts) {
		b.Field(p.name, p.adapter)
		if p.refine != nil {
			b.Refine("list-uniqueness:"+p.name, p.refine)
		}
	}

	// Reflect required
	if names := extractRequiredNames(doc); len(names) > 0 {
		b.Require(names...)
	}

	return b.Build()
}

// warnNonObjectRoot warns when the root declares a non-object type.
func warnNonObjectRoot(doc map[string]any, d *simpleDiag) {
	if t, _ := doc["type"].(string); t != "object" && t != "" {
		d.warnf("non-object at root treated as object-compatible: type=%q", t)
	}
}

// planUnknownBehavior determines how to handle unknown fields.
func planUnknownBehavior(doc map[string]any, d *simpleDiag) (UnknownBehavior, string) {
	if v, ok := doc["x-kubernetes-preserve-unknown-fields"].(bool); ok && v {
		return UnknownPreserve, "_unknown"
	}
	switch ap := doc["additionalProperties"].(type) {
	case bool:
		if ap {
			return UnknownPrune, ""
		}
		return UnknownStrict, ""
	case map[string]any:
		// TODO: support map schema. For now, accept and drop (permissive for MVP).
		d.warnf("additionalProperties as schema is treated as permissive for MVP")
		return UnknownPrune, ""
	default:
		// default equivalent to Prune
		return UnknownPrune, ""
	}
}

// planProperties generates reflection plans from the properties map.
func planProperties(doc map[string]any, d *simpleDiag, opts Options) []propertyPlan {
	pm, ok := doc["properties"].(map[string]any)
	if !ok || len(pm) == 0 {
		return nil
	}
	plans := make([]propertyPlan, 0, len(pm))
	for name, raw := range pm {
		ps, _ := raw.(map[string]any)
		plans = append(plans, planProperty(name, ps, d, opts))
	}
	return plans
}

// planProperty plans the adapter for a single property and a Refine when needed.
func planProperty(name string, ps map[string]any, d *simpleDiag, opts Options) propertyPlan {
	if pp, ok := planIntOrStringProperty(name, ps); ok {
		return applyEmbeddedResourceRefine(applyContainsRefine(applyListTypeRefine(pp, name, ps), name, ps, d), name, ps, opts)
	}
	if pp, ok := planObjectProperty(name, ps, d); ok {
		return applyEmbeddedResourceRefine(applyContainsRefine(applyListTypeRefine(pp, name, ps), name, ps, d), name, ps, opts)
	}
	pp := planMinimalProperty(name, ps, d, opts)
	return applyEmbeddedResourceRefine(applyContainsRefine(applyListTypeRefine(pp, name, ps), name, ps, d), name, ps, opts)
}

// list/check helpers are moved to list.go

// extractRequiredNames retrieves property names listed under required.
func extractRequiredNames(doc map[string]any) []string {
	if req, ok := doc["required"].([]any); ok {
		var names []string
		for _, r := range req {
			if s, ok := r.(string); ok {
				names = append(names, s)
			}
		}
		return names
	}
	return nil
}
