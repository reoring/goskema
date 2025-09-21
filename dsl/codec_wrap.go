package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// Codec adapts a Codec[A,B] into a Schema[B] that accepts wire A and produces domain B.
// Parse: In.Parse -> Decode -> Out.Normalize -> Out.ValidateValue -> Out.Refine
// Type/Rule/Validate (wire input): delegate to In().
// ValidateValue (domain value): delegate to Out().
// JSONSchema: delegate to Out().JSONSchema().
func Codec[A, B any](c goskema.Codec[A, B]) goskema.Schema[B] { return codecSchema[A, B]{c: c} }

type codecSchema[A, B any] struct{ c goskema.Codec[A, B] }

func (s codecSchema[A, B]) Parse(ctx context.Context, v any) (B, error) {
	// wire -> A
	a, err := s.c.In().Parse(ctx, v)
	if err != nil {
		var zero B
		if iss, ok := goskema.AsIssues(err); ok {
			return zero, iss
		}
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	// A -> B
	b, err := s.c.Decode(ctx, a)
	if err != nil {
		var zero B
		if iss, ok := goskema.AsIssues(err); ok {
			return zero, iss
		}
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	// Normalize -> ValidateValue -> Refine on Out schema
	b2, err := goskema.ApplyNormalize[B](ctx, b, s.c.Out())
	if err != nil {
		var zero B
		if iss, ok := goskema.AsIssues(err); ok {
			return zero, iss
		}
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if err := s.c.Out().ValidateValue(ctx, b2); err != nil {
		var zero B
		return zero, err
	}
	if err := goskema.ApplyRefine[B](ctx, b2, s.c.Out()); err != nil {
		var zero B
		if iss, ok := goskema.AsIssues(err); ok {
			return zero, iss
		}
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	return b2, nil
}

func (s codecSchema[A, B]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[B], error) {
	b, err := s.Parse(ctx, v)
	return goskema.Decoded[B]{Value: b, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s codecSchema[A, B]) TypeCheck(ctx context.Context, v any) error {
	return s.c.In().TypeCheck(ctx, v)
}
func (s codecSchema[A, B]) RuleCheck(ctx context.Context, v any) error {
	return s.c.In().RuleCheck(ctx, v)
}
func (s codecSchema[A, B]) Validate(ctx context.Context, v any) error {
	return s.c.In().Validate(ctx, v)
}
func (s codecSchema[A, B]) ValidateValue(ctx context.Context, v B) error {
	return s.c.Out().ValidateValue(ctx, v)
}
func (s codecSchema[A, B]) JSONSchema() (*js.Schema, error) { return s.c.Out().JSONSchema() }
