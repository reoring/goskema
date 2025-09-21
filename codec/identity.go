package codec

import (
	"context"

	goskema "github.com/reoring/goskema"
)

// Identity returns a Codec[T,T] that performs identity transformations.
// In() and Out() are the provided schema s. Decode/Encode validate via Out()/In() respectively.
func Identity[T any](s goskema.Schema[T]) goskema.Codec[T, T] {
	return &identityCodec[T]{in: s, out: s}
}

type identityCodec[T any] struct {
	in  goskema.Schema[T]
	out goskema.Schema[T]
}

func (c *identityCodec[T]) In() goskema.Schema[T]  { return c.in }
func (c *identityCodec[T]) Out() goskema.Schema[T] { return c.out }

func (c *identityCodec[T]) Decode(ctx context.Context, a T) (T, error) {
	// Validate on Out schema to ensure domain-side constraints
	if err := c.out.ValidateValue(ctx, a); err != nil {
		var zero T
		return zero, err
	}
	return a, nil
}

func (c *identityCodec[T]) Encode(ctx context.Context, b T) (T, error) {
	// Out.ValidateValue -> In.ValidateValue for identity (no transformation)
	if err := c.out.ValidateValue(ctx, b); err != nil {
		var zero T
		return zero, err
	}
	if err := c.in.ValidateValue(ctx, b); err != nil {
		var zero T
		return zero, err
	}
	return b, nil
}

func (c *identityCodec[T]) DecodeWithMeta(ctx context.Context, a T) (goskema.Decoded[T], error) {
	v, err := c.Decode(ctx, a)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (c *identityCodec[T]) EncodePreserving(ctx context.Context, db goskema.Decoded[T]) (T, error) {
	// For identity scalar values, presence does not change encoding outcome.
	return c.Encode(ctx, db.Value)
}
