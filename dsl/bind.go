package dsl

import (
	"context"
	"reflect"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// Bind builds an object schema and binds it to struct type T (free function for Go version compatibility).
func Bind[T any](b *objectBuilder) (goskema.Schema[T], error) {
	s, err := b.Build()
	if err != nil {
		var zero goskema.Schema[T]
		return zero, err
	}
	os, ok := s.(*objectSchema)
	if !ok {
		var zero goskema.Schema[T]
		return zero, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: "unexpected schema type for Bind"}}
	}
	return newTypedObjectSchema[T](os)
}

// MustBind is like Bind but panics on error (free function for Go version compatibility).
func MustBind[T any](b *objectBuilder) goskema.Schema[T] {
	s, err := Bind[T](b)
	if err != nil {
		panic(err)
	}
	return s
}

// typedObjectSchema adapts an objectSchema to a typed struct T using key resolution.
type typedObjectSchema[T any] struct {
	inner       *objectSchema
	t           reflect.Type
	fieldByKey  map[string]int // DSL key -> struct field index
	typedRules  []typedRule[T]
	typedRulesE []typedRuleE[T]
}

func newTypedObjectSchema[T any](os *objectSchema) (goskema.Schema[T], error) {
	var zero goskema.Schema[T]
	var t T
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return zero, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: "Bind[T] requires struct T"}}
	}
	idxByName := make(map[string]int)
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		name := goskema.ResolveStructKey(sf)
		if name == "-" || name == "" {
			continue
		}
		idxByName[name] = i
	}
	fm := make(map[string]int)
	for k := range os.fields {
		if i, ok := idxByName[k]; ok {
			fm[k] = i
		}
	}
	// rehydrate typed rules for T
	var trs []typedRule[T]
	var trse []typedRuleE[T]
	if raw, ok := os.typedRulesAny.([]any); ok && len(raw) > 0 {
		trs = make([]typedRule[T], 0, len(raw))
		trse = make([]typedRuleE[T], 0, len(raw))
		for _, it := range raw {
			if tr, ok := it.(typedRule[T]); ok {
				trs = append(trs, tr)
				continue
			}
			if tre, ok := it.(typedRuleE[T]); ok {
				trse = append(trse, tre)
				continue
			}
		}
	}
	return &typedObjectSchema[T]{inner: os, t: rt, fieldByKey: fm, typedRules: trs, typedRulesE: trse}, nil
}

// Parse maps wire -> map via inner, then into struct fields by mapping.
func (s *typedObjectSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	var zero T
	m, err := s.inner.Parse(ctx, v)
	if err != nil {
		return zero, err
	}
	rv := reflect.New(s.t).Elem()
	for key, idx := range s.fieldByKey {
		if val, ok := m[key]; ok {
			fv := rv.Field(idx)
			if !fv.CanSet() {
				continue
			}
			// Gracefully handle nulls for nillable fields
			if val == nil {
				switch fv.Kind() {
				case reflect.Interface, reflect.Pointer, reflect.Slice, reflect.Map, reflect.Func, reflect.Chan:
					fv.Set(reflect.Zero(fv.Type()))
				default:
					// leave zero value for non-nillable fields
				}
				continue
			}
			vv := reflect.ValueOf(val)
			if vv.Type().AssignableTo(fv.Type()) {
				fv.Set(vv)
			} else if vv.Type().ConvertibleTo(fv.Type()) {
				fv.Set(vv.Convert(fv.Type()))
			} else {
				return zero, goskema.Issues{goskema.Issue{Path: "/" + key, Code: goskema.CodeInvalidType, Message: "field type mismatch"}}
			}
		}
	}
	out := rv.Interface().(T)
	// Execute typed rules also on Parse path (without returning presence). We reconstruct
	// a minimal PresenceMap from the wire map to enable presence-gated rules.
	if (len(s.typedRules) > 0 || len(s.typedRulesE) > 0) && !goskema.IsSkipTypedRules(ctx) {
		pm := goskema.PresenceMap{"/": goskema.PresenceSeen}
		// minimally mark seen keys at top-level from the wire map
		for k := range m {
			pm["/"+k] |= goskema.PresenceSeen
		}
		if len(s.typedRules) > 0 {
			if iss := runTypedRules[T](ctx, out, pm, s.typedRules, goskema.PhaseDomain); len(iss) > 0 {
				return zero, iss
			}
		}
		if len(s.typedRules) > 0 {
			if iss := runTypedRules[T](ctx, out, pm, s.typedRules, goskema.PhaseContext); len(iss) > 0 {
				return zero, iss
			}
		}
		if len(s.typedRulesE) > 0 {
			if iss, e := runTypedRulesE[T](ctx, out, pm, s.typedRulesE, goskema.PhaseContext); e != nil {
				return zero, e
			} else if len(iss) > 0 {
				return zero, iss
			}
		}
	}
	return out, nil
}

func (s *typedObjectSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	var zero goskema.Decoded[T]
	dm, err := s.inner.ParseWithMeta(ctx, v)
	if err != nil {
		return zero, err
	}
	// At this point, ParseFromWithMeta has set the skip flag so that s.Parse won't execute typed rules.
	out, err := s.Parse(ctx, dm.Value)
	if err != nil {
		return zero, err
	}
	// typed rules: domain phase, then context phase (stop on issues across phases)
	if len(s.typedRules) > 0 || len(s.typedRulesE) > 0 {
		if len(s.typedRules) > 0 {
			if iss := runTypedRules[T](ctx, out, dm.Presence, s.typedRules, goskema.PhaseDomain); len(iss) > 0 {
				return goskema.Decoded[T]{Value: out, Presence: dm.Presence}, iss
			}
		}
		if len(s.typedRules) > 0 {
			if iss := runTypedRules[T](ctx, out, dm.Presence, s.typedRules, goskema.PhaseContext); len(iss) > 0 {
				return goskema.Decoded[T]{Value: out, Presence: dm.Presence}, iss
			}
		}
		if len(s.typedRulesE) > 0 {
			if iss, err := runTypedRulesE[T](ctx, out, dm.Presence, s.typedRulesE, goskema.PhaseContext); err != nil {
				return goskema.Decoded[T]{Value: out, Presence: dm.Presence}, err
			} else if len(iss) > 0 {
				return goskema.Decoded[T]{Value: out, Presence: dm.Presence}, iss
			}
		}
	}
	// Note: Presence keys are in wire (map) shape using DSL keys. Bind maps DSL keys to struct fields,
	// so we can carry PresenceMap as-is for preserving encode decisions at object level.
	return goskema.Decoded[T]{Value: out, Presence: dm.Presence}, nil
}

func (s *typedObjectSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return s.inner.TypeCheck(ctx, v)
}
func (s *typedObjectSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return s.inner.RuleCheck(ctx, v)
}
func (s *typedObjectSchema[T]) Validate(ctx context.Context, v any) error {
	return s.inner.Validate(ctx, v)
}

func (s *typedObjectSchema[T]) ValidateValue(ctx context.Context, v T) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	m := make(map[string]any, len(s.fieldByKey))
	for key, idx := range s.fieldByKey {
		fv := rv.Field(idx)
		if !fv.IsValid() {
			continue
		}
		// Treat zero values as present to avoid false required errors for typed values
		m[key] = fv.Interface()
	}
	return s.inner.ValidateValue(ctx, m)
}

func (s *typedObjectSchema[T]) JSONSchema() (*js.Schema, error) { return s.inner.JSONSchema() }
