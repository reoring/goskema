package kubeopenapi

import (
	"context"
	"reflect"
	"strconv"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/dsl"
)

// applyContainsRefine chains a contains/minContains/maxContains refine onto the property plan
// when the schema declares type: array and a contains schema. It preserves existing refine
// by running it first, then the contains refine.
func applyContainsRefine(pp propertyPlan, name string, ps map[string]any, d *simpleDiag) propertyPlan {
	t, _ := ps["type"].(string)
	if t != "array" {
		return pp
	}
	contains, _ := ps["contains"].(map[string]any)
	if contains == nil {
		return pp
	}
	var minC, maxC int
	minC = -1
	maxC = -1
	if v, ok := ps["minContains"]; ok {
		switch n := v.(type) {
		case float64:
			minC = int(n)
		case int:
			minC = n
		}
	}
	if v, ok := ps["maxContains"]; ok {
		switch n := v.(type) {
		case float64:
			maxC = int(n)
		case int:
			maxC = n
		}
	}
	ref := buildContainsRefiner(name, contains, minC, maxC)
	if ref == nil {
		return pp
	}
	// Try enabling streaming contains when the adapter is an ArraySchema and the contains
	// can be evaluated with our lightweight predicate. We only support type/required subset.
	if pp.adapter.Orig() != nil {
		if arr, ok := pp.adapter.Orig().(interface {
			WithStreamContainsAny(int, int, func(any) bool)
		}); ok {
			pred := func(v any) bool { return containsMatch(v, contains) }
			arr.WithStreamContainsAny(minC, maxC, pred)
			// Keep object-level refine as fallback for non-streaming path
		}
		// generics: attempt concrete ArraySchema[...] via type switch
		switch a := pp.adapter.Orig().(type) {
		case *dsl.ArraySchema[map[string]any]:
			a.WithStreamContainsAny(minC, maxC, func(v any) bool { return containsMatch(v, contains) })
		case *dsl.ArraySchema[any]:
			a.WithStreamContainsAny(minC, maxC, func(v any) bool { return containsMatch(v, contains) })
		}
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

// buildContainsRefiner returns an object-level refine that counts elements in the array field
// matching the provided contains schema (MVP: type check and required keys for objects) and
// enforces minContains/maxContains.
func buildContainsRefiner(fieldName string, contains map[string]any, minC, maxC int) func(ctx context.Context, m map[string]any) error {
	match := func(el any) bool { return containsMatch(el, contains) }
	return func(ctx context.Context, m map[string]any) error {
		v, ok := m[fieldName]
		if !ok || v == nil {
			return nil
		}
		// Iterate generically over arrays/slices
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return nil
		}
		count := 0
		var indices []int
		for i := 0; i < rv.Len(); i++ {
			if match(rv.Index(i).Interface()) {
				count++
				if len(indices) < 8 { // cap to keep hint concise
					indices = append(indices, i)
				}
				// Early hint: if maxC is set and we already exceeded, we can still
				// provide the index that exceeded in the hint below.
			}
		}
		var iss goskema.Issues
		if minC >= 0 && count < minC {
			hint := "matched=" + strconv.Itoa(count)
			if len(indices) > 0 {
				hint += "; indices=" + intsToCSV(indices)
			}
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName, Code: goskema.CodeTooShort, Message: "contains matched less than minContains", Hint: hint})
		}
		if maxC >= 0 && count > maxC {
			hint := "matched=" + strconv.Itoa(count)
			if len(indices) > 0 {
				hint += "; indices=" + intsToCSV(indices)
			}
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName, Code: goskema.CodeTooLong, Message: "contains matched more than maxContains", Hint: hint})
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

func intsToCSV(xs []int) string {
	if len(xs) == 0 {
		return ""
	}
	// simple and small: allocate once
	s := strconv.Itoa(xs[0])
	for i := 1; i < len(xs); i++ {
		s += "," + strconv.Itoa(xs[i])
	}
	return s
}

func containsMatch(v any, contains map[string]any) bool {
	if contains == nil {
		return true
	}
	if t, _ := contains["type"].(string); t != "" {
		if !valueMatchesType(v, t) {
			return false
		}
		if t == "object" {
			// Optional: honor required for object elements
			if req, ok := contains["required"].([]any); ok && len(req) > 0 {
				mv, _ := v.(map[string]any)
				if mv == nil {
					return false
				}
				for _, r := range req {
					if k, ok := r.(string); ok {
						if _, exists := mv[k]; !exists {
							return false
						}
					}
				}
			}
		}
		return true
	}
	// No recognized constraint -> accept (MVP)
	return true
}
