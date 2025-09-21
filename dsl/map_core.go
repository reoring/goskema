package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	eng "github.com/reoring/goskema/internal/engine"
	str "github.com/reoring/goskema/internal/stream"
	js "github.com/reoring/goskema/jsonschema"
)

// MapAny returns a minimal Schema[map[string]any] useful for passthrough targets or loose maps.
func MapAny() goskema.Schema[map[string]any] { return mapAnySchema{} }

type mapAnySchema struct{}

func (mapAnySchema) Parse(ctx context.Context, v any) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	return m, nil
}
func (mapAnySchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]any], error) {
	m, err := (mapAnySchema{}).Parse(ctx, v)
	return goskema.Decoded[map[string]any]{Value: m, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (mapAnySchema) TypeCheck(ctx context.Context, v any) error                { return nil }
func (mapAnySchema) RuleCheck(ctx context.Context, v any) error                { return nil }
func (mapAnySchema) Validate(ctx context.Context, v any) error                 { return nil }
func (mapAnySchema) ValidateValue(ctx context.Context, v map[string]any) error { return nil }
func (mapAnySchema) JSONSchema() (*js.Schema, error) {
	return &js.Schema{Type: "object", AdditionalProperties: true}, nil
}

// Map returns a schema for JSON objects where all properties are validated by elem schema.
// It decodes into map[string]V and validates each value using elem.
func Map[V any](elem goskema.Schema[V]) goskema.Schema[map[string]V] { return mapSchema[V]{val: elem} }

// MapOf adapts Map[V] to AnyAdapter for use in typed object builders.
func MapOf[V any](elem goskema.Schema[V]) AnyAdapter {
	return anyAdapterFromSchema[map[string]V](Map[V](elem))
}

type mapSchema[V any] struct{ val goskema.Schema[V] }

func (m mapSchema[V]) Parse(ctx context.Context, v any) (map[string]V, error) {
	switch src := v.(type) {
	case map[string]V:
		for k, vv := range src {
			if err := m.val.ValidateValue(ctx, vv); err != nil {
				if iss, ok := goskema.AsIssues(err); ok {
					var out goskema.Issues
					base := "/" + k
					for _, it := range iss {
						p := it.Path
						if p == "" || p == "/" {
							p = base
						} else if p[0] == '/' {
							p = base + p
						} else {
							p = base + "/" + p
						}
						out = goskema.AppendIssues(out, goskema.Issue{Path: p, Code: it.Code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
					}
					return nil, out
				}
				return nil, goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
			}
		}
		nn, err := goskema.ApplyNormalize[map[string]V](ctx, src, m)
		if err != nil {
			return nil, err
		}
		if err := goskema.ApplyRefine[map[string]V](ctx, nn, m); err != nil {
			return nil, err
		}
		return nn, nil
	case map[string]any:
		out := make(map[string]V, len(src))
		for k, anyVal := range src {
			vv, err := m.val.Parse(ctx, anyVal)
			if err != nil {
				if iss, ok := goskema.AsIssues(err); ok {
					var outIss goskema.Issues
					base := "/" + k
					for _, it := range iss {
						p := it.Path
						if p == "" || p == "/" {
							p = base
						} else if p[0] == '/' {
							p = base + p
						} else {
							p = base + "/" + p
						}
						outIss = goskema.AppendIssues(outIss, goskema.Issue{Path: p, Code: it.Code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
					}
					return nil, outIss
				}
				return nil, goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
			}
			out[k] = vv
		}
		if err := m.ValidateValue(ctx, out); err != nil {
			return nil, err
		}
		nn, err := goskema.ApplyNormalize[map[string]V](ctx, out, m)
		if err != nil {
			return nil, err
		}
		if err := goskema.ApplyRefine[map[string]V](ctx, nn, m); err != nil {
			return nil, err
		}
		return nn, nil
	default:
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
}

func (m mapSchema[V]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]V], error) {
	mv, err := m.Parse(ctx, v)
	return goskema.Decoded[map[string]V]{Value: mv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

// ---- streaming SPI ----
func (m mapSchema[V]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (map[string]V, error) {
	engSrc := goskema.EngineTokenSource(src)
	tok, err := engSrc.NextToken()
	if err != nil {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if tok.Kind != eng.KindBeginObject {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	out := make(map[string]V)
	for {
		t, err := engSrc.NextToken()
		if err != nil {
			return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
		}
		if t.Kind == eng.KindEndObject {
			break
		}
		if t.Kind != eng.KindKey {
			return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: "unexpected token in object"}}
		}
		k := t.String
		sub := str.NewSubtreeSource(engSrc)
		var anyVal any
		if src.NumberMode() == goskema.NumberFloat64 { // NOTE: keep consistent with number handling above
			anyVal, err = eng.DecodeAnyFromSourceAsFloat64(sub)
		} else {
			anyVal, err = eng.DecodeAnyFromSource(sub)
		}
		if err != nil {
			return nil, goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
		}
		vv, perr := m.val.Parse(ctx, anyVal)
		if perr != nil {
			if iss, ok := goskema.AsIssues(perr); ok {
				var outIss goskema.Issues
				base := "/" + k
				for _, it := range iss {
					p := it.Path
					if p == "" || p == "/" {
						p = base
					} else if p[0] == '/' {
						p = base + p
					} else {
						p = base + "/" + p
					}
					outIss = goskema.AppendIssues(outIss, goskema.Issue{Path: p, Code: it.Code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
				}
				return nil, outIss
			}
			return nil, goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: perr.Error(), Cause: perr}}
		}
		out[k] = vv
	}
	if err := m.ValidateValue(ctx, out); err != nil {
		return nil, err
	}
	nn, err := goskema.ApplyNormalize[map[string]V](ctx, out, m)
	if err != nil {
		return nil, err
	}
	if err := goskema.ApplyRefine[map[string]V](ctx, nn, m); err != nil {
		return nil, err
	}
	return nn, nil
}

func (m mapSchema[V]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[map[string]V], error) {
	mv, err := m.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[map[string]V]{Value: mv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (m mapSchema[V]) TypeCheck(ctx context.Context, v any) error {
	switch v.(type) {
	case map[string]V, map[string]any:
		return nil
	default:
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
}

func (m mapSchema[V]) RuleCheck(ctx context.Context, v any) error { return nil }

func (m mapSchema[V]) Validate(ctx context.Context, v any) error {
	if err := m.TypeCheck(ctx, v); err != nil {
		return err
	}
	return m.RuleCheck(ctx, v)
}

func (m mapSchema[V]) ValidateValue(ctx context.Context, v map[string]V) error {
	for k, vv := range v {
		if err := m.val.ValidateValue(ctx, vv); err != nil {
			if iss, ok := goskema.AsIssues(err); ok {
				return iss
			}
			return goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
		}
	}
	return nil
}

func (m mapSchema[V]) JSONSchema() (*js.Schema, error) {
	vs, err := m.val.JSONSchema()
	if err != nil {
		return nil, err
	}
	return &js.Schema{Type: "object", AdditionalProperties: vs}, nil
}
