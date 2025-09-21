package kubeopenapi

import (
	"context"

	goskema "github.com/reoring/goskema"
)

// buildAnyOfRefiner returns a refine that validates an instance against multiple
// candidate schemas and applies ambiguity strategy. MVP: supports primitive type
// branches (via type field) and object branches that only set required fields.
func buildAnyOfRefiner(fieldName string, branches []map[string]any, strat AmbiguityStrategy) func(ctx context.Context, m map[string]any) error {
	// Prepare light-weight branch checkers
	type checker func(v any) bool
	var checks []checker
	for _, b := range branches {
		// prefer type, otherwise required on object
		if t, _ := b["type"].(string); t != "" {
			typ := t
			checks = append(checks, func(v any) bool { return valueMatchesType(v, typ) })
			continue
		}
		if req, ok := b["required"].([]any); ok {
			var names []string
			for _, r := range req {
				if s, ok := r.(string); ok {
					names = append(names, s)
				}
			}
			checks = append(checks, func(v any) bool {
				mv, _ := v.(map[string]any)
				if mv == nil {
					return false
				}
				for _, k := range names {
					if _, ok := mv[k]; !ok {
						return false
					}
				}
				return true
			})
			continue
		}
		// fallback: accept
		checks = append(checks, func(v any) bool { return true })
	}
	return func(ctx context.Context, m map[string]any) error {
		v := m[fieldName]
		if v == nil {
			return nil
		}
		matched := 0
		for _, c := range checks {
			if c(v) {
				matched++
			}
		}
		switch strat {
		case AmbiguityError:
			if matched != 1 {
				return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: "ambiguous_match", Message: "ambiguous anyOf/oneOf match"}}
			}
		case AmbiguityFirstMatch:
			if matched == 0 {
				return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: "no_match", Message: "no branch matched"}}
			}
		case AmbiguityScoreBest:
			if matched == 0 {
				return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: "no_match", Message: "no branch matched"}}
			}
			// MVP: same as FirstMatch for now
		}
		return nil
	}
}
