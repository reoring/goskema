package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
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
