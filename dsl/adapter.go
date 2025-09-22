package dsl

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

// AnyAdapter adapts Schema[T] to an any-typed DSL wrapper.
// It keeps the original schema to support default application, JSON Schema
// augmentation, and IR conversion.
type AnyAdapter struct {
	parse           func(context.Context, any) (any, error)
	validateValue   func(context.Context, any) error
	parseFromSource func(context.Context, goskema.Source, goskema.ParseOpt) (any, error)
	applyDefault    func(context.Context) (any, error)
	jsonSchema      func() (*js.Schema, error)
	orig            any
}

// anyAdapterFromSchema wraps a strongly typed Schema[T] as AnyAdapter for Field builders.
func anyAdapterFromSchema[T any](s goskema.Schema[T]) AnyAdapter {
	ad := AnyAdapter{
		parse: func(ctx context.Context, v any) (any, error) { return s.Parse(ctx, v) },
		validateValue: func(ctx context.Context, v any) error {
			tv, ok := v.(T)
			if !ok {
				return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: "invalid field type"}}
			}
			return s.ValidateValue(ctx, tv)
		},
		jsonSchema: s.JSONSchema,
		orig:       s,
	}

	type parseFromSourceLike[T any] interface {
		ParseFromSource(context.Context, goskema.Source, goskema.ParseOpt) (T, error)
	}
	if pf, ok := any(s).(parseFromSourceLike[T]); ok {
		ad.parseFromSource = func(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (any, error) {
			v, err := pf.ParseFromSource(ctx, src, opt)
			if err != nil {
				return nil, err
			}
			return any(v), nil
		}
	}

	return ad
}

// Orig returns the original underlying Schema[T] or builder object used to create this adapter.
// It is intended for advanced integrations (e.g., enabling streaming hints) and may change.
func (ad AnyAdapter) Orig() any { return ad.orig }

// Nullable wraps an AnyAdapter to accept nulls (JSON null) for both parse and validate.
// When the input value is nil, parsing succeeds and returns nil; validation also succeeds.
// JSON Schema export is left to the underlying adapter as our minimal Schema does not
// yet model union types; callers can post-process if needed.
func Nullable(ad AnyAdapter) AnyAdapter {
	prevParse := ad.parse
	prevValidate := ad.validateValue
	prevJSON := ad.jsonSchema
	out := ad
	out.parse = func(ctx context.Context, v any) (any, error) {
		if v == nil {
			return nil, nil
		}
		if prevParse == nil {
			return v, nil
		}
		return prevParse(ctx, v)
	}
	out.validateValue = func(ctx context.Context, v any) error {
		if v == nil {
			return nil
		}
		if prevValidate == nil {
			if prevParse == nil {
				return nil
			}
			_, err := prevParse(ctx, v)
			return err
		}
		return prevValidate(ctx, v)
	}
	out.jsonSchema = func() (*js.Schema, error) {
		if prevJSON == nil {
			return &js.Schema{}, nil
		}
		return prevJSON()
	}
	return out
}

// Nullable enables fluent chaining: g.StringOf[T]().Nullable()
func (ad AnyAdapter) Nullable() AnyAdapter { return Nullable(ad) }

// Min sets a numeric minimum (inclusive) constraint at runtime and in JSON Schema.
// Non-numeric values are ignored by this guard (type errors are handled elsewhere).
func (ad AnyAdapter) Min(n float64) AnyAdapter {
	prevParse := ad.parse
	prevValidate := ad.validateValue
	prevJSON := ad.jsonSchema
	prevSrc := ad.parseFromSource
	out := ad
	out.parse = func(ctx context.Context, v any) (any, error) {
		if prevParse != nil {
			val, err := prevParse(ctx, v)
			if err != nil {
				return nil, err
			}
			if err := minCheck(val, n); err != nil {
				return nil, err
			}
			return val, nil
		}
		if err := minCheck(v, n); err != nil {
			return nil, err
		}
		return v, nil
	}
	out.validateValue = func(ctx context.Context, v any) error {
		if prevValidate != nil {
			if err := prevValidate(ctx, v); err != nil {
				return err
			}
		}
		return minCheck(v, n)
	}
	out.jsonSchema = func() (*js.Schema, error) {
		s := &js.Schema{}
		if prevJSON != nil {
			ps, err := prevJSON()
			if err != nil {
				return nil, err
			}
			if ps != nil {
				s = ps
			}
		}
		s.Minimum = jsPtrFloat(n)
		if s.Type == "" {
			s.Type = "number"
		}
		return s, nil
	}
	if prevSrc != nil {
		out.parseFromSource = func(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (any, error) {
			val, err := prevSrc(ctx, src, opt)
			if err != nil {
				return nil, err
			}
			if err := minCheck(val, n); err != nil {
				return nil, err
			}
			return val, nil
		}
	}
	return out
}

// Max sets a numeric maximum (inclusive) constraint at runtime and in JSON Schema.
func (ad AnyAdapter) Max(n float64) AnyAdapter {
	prevParse := ad.parse
	prevValidate := ad.validateValue
	prevJSON := ad.jsonSchema
	prevSrc := ad.parseFromSource
	out := ad
	out.parse = func(ctx context.Context, v any) (any, error) {
		if prevParse != nil {
			val, err := prevParse(ctx, v)
			if err != nil {
				return nil, err
			}
			if err := maxCheck(val, n); err != nil {
				return nil, err
			}
			return val, nil
		}
		if err := maxCheck(v, n); err != nil {
			return nil, err
		}
		return v, nil
	}
	out.validateValue = func(ctx context.Context, v any) error {
		if prevValidate != nil {
			if err := prevValidate(ctx, v); err != nil {
				return err
			}
		}
		return maxCheck(v, n)
	}
	out.jsonSchema = func() (*js.Schema, error) {
		s := &js.Schema{}
		if prevJSON != nil {
			ps, err := prevJSON()
			if err != nil {
				return nil, err
			}
			if ps != nil {
				s = ps
			}
		}
		s.Maximum = jsPtrFloat(n)
		if s.Type == "" {
			s.Type = "number"
		}
		return s, nil
	}
	if prevSrc != nil {
		out.parseFromSource = func(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (any, error) {
			val, err := prevSrc(ctx, src, opt)
			if err != nil {
				return nil, err
			}
			if err := maxCheck(val, n); err != nil {
				return nil, err
			}
			return val, nil
		}
	}
	return out
}

// ---- helpers ----
func jsPtrFloat(v float64) *float64 { return &v }

func minCheck(v any, min float64) error {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int:
		if float64(n) < min {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooSmall, Message: i18n.T(goskema.CodeTooSmall, nil)}}
		}
	case int8, int16, int32, int64:
		if reflect.ValueOf(n).Int() < int64(min) {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooSmall, Message: i18n.T(goskema.CodeTooSmall, nil)}}
		}
	case uint, uint8, uint16, uint32, uint64:
		if float64(reflect.ValueOf(n).Uint()) < min {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooSmall, Message: i18n.T(goskema.CodeTooSmall, nil)}}
		}
	case float32, float64:
		if reflect.ValueOf(n).Convert(reflect.TypeOf(float64(0))).Float() < min {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooSmall, Message: i18n.T(goskema.CodeTooSmall, nil)}}
		}
	case json.Number:
		if f, err := strconv.ParseFloat(string(n), 64); err == nil {
			if f < min {
				return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooSmall, Message: i18n.T(goskema.CodeTooSmall, nil)}}
			}
		}
	}
	return nil
}

func maxCheck(v any, max float64) error {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int:
		if float64(n) > max {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooBig, Message: i18n.T(goskema.CodeTooBig, nil)}}
		}
	case int8, int16, int32, int64:
		if reflect.ValueOf(n).Int() > int64(max) {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooBig, Message: i18n.T(goskema.CodeTooBig, nil)}}
		}
	case uint, uint8, uint16, uint32, uint64:
		if float64(reflect.ValueOf(n).Uint()) > max {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooBig, Message: i18n.T(goskema.CodeTooBig, nil)}}
		}
	case float32, float64:
		if reflect.ValueOf(n).Convert(reflect.TypeOf(float64(0))).Float() > max {
			return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooBig, Message: i18n.T(goskema.CodeTooBig, nil)}}
		}
	case json.Number:
		if f, err := strconv.ParseFloat(string(n), 64); err == nil {
			if f > max {
				return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooBig, Message: i18n.T(goskema.CodeTooBig, nil)}}
			}
		}
	}
	return nil
}
