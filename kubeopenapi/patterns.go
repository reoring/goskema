package kubeopenapi

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"regexp"

	goskema "github.com/reoring/goskema"
)

// buildKeyPatternRefiner builds an object-level refine that enforces key regex on a map field.
func buildKeyPatternRefiner(fieldName string, pattern string) func(ctx context.Context, m map[string]any) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Fail closed: invalid regex must report an issue rather than accept everything
		invalidErr := err
		return func(ctx context.Context, m map[string]any) error {
			return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: goskema.CodePattern, Message: "invalid regex pattern", Cause: invalidErr}}
		}
	}
	return func(ctx context.Context, m map[string]any) error {
		v, ok := m[fieldName]
		if !ok || v == nil {
			return nil
		}
		var iss goskema.Issues
		// reflect over map[string]T (T varies) to read keys
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			for _, mk := range rv.MapKeys() {
				k := mk.String()
				if !re.MatchString(k) {
					iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodePattern, Message: "key does not match pattern"})
				}
			}
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

// buildKeyPatternsRefiner enforces that all keys match at least one of the provided regex patterns.
func buildKeyPatternsRefiner(fieldName string, patterns []string) func(ctx context.Context, m map[string]any) error {
	// Precompile; invalid patterns cause a single error at refine time
	res := make([]*regexp.Regexp, 0, len(patterns))
	var invalid error
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			invalid = err
			// keep going to collect other valid patterns but remember the error
			continue
		}
		res = append(res, re)
	}
	return func(ctx context.Context, m map[string]any) error {
		if invalid != nil {
			return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: goskema.CodePattern, Message: "invalid regex pattern", Cause: invalid}}
		}
		v, ok := m[fieldName]
		if !ok || v == nil {
			return nil
		}
		var iss goskema.Issues
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			for _, mk := range rv.MapKeys() {
				k := mk.String()
				matched := false
				for _, re := range res {
					if re.MatchString(k) {
						matched = true
						break
					}
				}
				if !matched {
					iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodePattern, Message: "key does not match any allowed pattern"})
				}
			}
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

// buildPatternPropertiesValueTypeRefiner enforces value types for keys that
// match any of the provided regex patterns, and optionally enforces a type for
// unmatched keys when additionalProperties is a schema.
//
// If patternType is empty, values for matched keys are not type-checked.
// If apType is empty, unmatched keys are not type-checked.
func buildPatternPropertiesValueTypeRefiner(fieldName string, patterns []string, patternType string, apType string) func(ctx context.Context, m map[string]any) error {
	// Precompile patterns; invalid ones are reported as an error
	res := make([]*regexp.Regexp, 0, len(patterns))
	var invalid error
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			invalid = err
			continue
		}
		res = append(res, re)
	}
	return func(ctx context.Context, m map[string]any) error {
		if invalid != nil {
			return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: goskema.CodePattern, Message: "invalid regex pattern", Cause: invalid}}
		}
		v, ok := m[fieldName]
		if !ok || v == nil {
			return nil
		}
		var iss goskema.Issues
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			for _, mk := range rv.MapKeys() {
				k := mk.String()
				val := rv.MapIndex(mk).Interface()
				matched := false
				for _, re := range res {
					if re.MatchString(k) {
						matched = true
						break
					}
				}
				if matched {
					if patternType != "" && !valueMatchesType(val, patternType) {
						iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodeInvalidType, Message: "value type mismatch for patternProperties"})
					}
				} else {
					if apType != "" && !valueMatchesType(val, apType) {
						iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodeInvalidType, Message: "value type mismatch for additionalProperties"})
					}
				}
			}
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

// buildPatternPropertiesValueTypesRefiner enforces value types for keys that match
// one or more regex patterns. All matched pattern type constraints are applied
// (intersection semantics like JSON Schema). If a key matches no pattern and
// additionalProperties is a schema, apType is applied.
//
// patternToType maps regex pattern -> expected JSON type name ("string", "boolean",
// "number", "integer", "object"). Empty type means no check for that pattern.
func buildPatternPropertiesValueTypesRefiner(fieldName string, patternToType map[string]string, apType string) func(ctx context.Context, m map[string]any) error {
	// Precompile regexps; invalid ones produce an error
	type compiled struct {
		re  *regexp.Regexp
		typ string
	}
	var cs []compiled
	var invalid error
	for p, t := range patternToType {
		re, err := regexp.Compile(p)
		if err != nil {
			invalid = err
			continue
		}
		cs = append(cs, compiled{re: re, typ: t})
	}
	return func(ctx context.Context, m map[string]any) error {
		if invalid != nil {
			return goskema.Issues{goskema.Issue{Path: "/" + fieldName, Code: goskema.CodePattern, Message: "invalid regex pattern", Cause: invalid}}
		}
		v, ok := m[fieldName]
		if !ok || v == nil {
			return nil
		}
		var iss goskema.Issues
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			for _, mk := range rv.MapKeys() {
				k := mk.String()
				val := rv.MapIndex(mk).Interface()
				matched := false
				for _, c := range cs {
					if c.re.MatchString(k) {
						matched = true
						if c.typ != "" && !valueMatchesType(val, c.typ) {
							iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodeInvalidType, Message: "value type mismatch for patternProperties"})
						}
					}
				}
				if !matched {
					if apType != "" && !valueMatchesType(val, apType) {
						iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + k, Code: goskema.CodeInvalidType, Message: "value type mismatch for additionalProperties"})
					}
				}
			}
		}
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}

func valueMatchesType(v any, want string) bool {
	switch want {
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "number":
		return isNumeric(v)
	case "integer":
		return isInteger(v)
	case "object":
		// Accept any map value as object
		rv := reflect.ValueOf(v)
		return rv.Kind() == reflect.Map
	default:
		// Unknown types are treated as pass for MVP
		return true
	}
}

func isNumeric(v any) bool {
	switch t := v.(type) {
	case json.Number:
		// Accept any number representation
		if _, err := t.Float64(); err == nil {
			return true
		}
		if _, err := t.Int64(); err == nil {
			return true
		}
		return false
	case float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isInteger(v any) bool {
	switch t := v.(type) {
	case json.Number:
		if _, err := t.Int64(); err == nil {
			return true
		}
		// fall back to float check
		if f, err := t.Float64(); err == nil {
			return math.Trunc(f) == f
		}
		return false
	case float64:
		return math.Trunc(t) == t
	case float32:
		return math.Trunc(float64(t)) == float64(t)
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}
