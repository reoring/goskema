package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	js "github.com/reoring/goskema/jsonschema"
)

// IdentitySchema exposes Decode/Encode methods on top of Schema[T].
// This is a thin view that delegates to codec.Identity(inner).
type IdentitySchema[T any] interface {
	goskema.Schema[T]
	Decode(ctx context.Context, v T) (T, error)
	Encode(ctx context.Context, v T) (T, error)
	DecodeWithMeta(ctx context.Context, v T) (goskema.Decoded[T], error)
	EncodePreserving(ctx context.Context, dv goskema.Decoded[T]) (T, error)
}

// WithIdentity wraps a Schema[T] and provides Decode/Encode sugar similar to Zod.
func WithIdentity[T any](s goskema.Schema[T]) IdentitySchema[T] {
	return identitySchemaView[T]{inner: s}
}

type identitySchemaView[T any] struct{ inner goskema.Schema[T] }

// ---- Schema[T] methods (forward to inner) ----

func (w identitySchemaView[T]) Parse(ctx context.Context, v any) (T, error) {
	return w.inner.Parse(ctx, v)
}

func (w identitySchemaView[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	return w.inner.ParseWithMeta(ctx, v)
}

func (w identitySchemaView[T]) TypeCheck(ctx context.Context, v any) error {
	return w.inner.TypeCheck(ctx, v)
}

func (w identitySchemaView[T]) RuleCheck(ctx context.Context, v any) error {
	return w.inner.RuleCheck(ctx, v)
}

func (w identitySchemaView[T]) Validate(ctx context.Context, v any) error {
	return w.inner.Validate(ctx, v)
}

func (w identitySchemaView[T]) ValidateValue(ctx context.Context, v T) error {
	return w.inner.ValidateValue(ctx, v)
}

func (w identitySchemaView[T]) JSONSchema() (*js.Schema, error) { return w.inner.JSONSchema() }

// ---- identity codec sugar ----

func (w identitySchemaView[T]) Decode(ctx context.Context, v T) (T, error) {
	return codec.Identity(w.inner).Decode(ctx, v)
}

func (w identitySchemaView[T]) Encode(ctx context.Context, v T) (T, error) {
	return codec.Identity(w.inner).Encode(ctx, v)
}

func (w identitySchemaView[T]) DecodeWithMeta(ctx context.Context, v T) (goskema.Decoded[T], error) {
	return codec.Identity(w.inner).DecodeWithMeta(ctx, v)
}

func (w identitySchemaView[T]) EncodePreserving(ctx context.Context, dv goskema.Decoded[T]) (T, error) {
	return codec.Identity(w.inner).EncodePreserving(ctx, dv)
}
