package middleware

import (
	"context"

	goskema "github.com/reoring/goskema"
)

// ctxKeyDecoded is a typed context key for storing Decoded[T].
// Using a generic struct type ensures uniqueness per T.
type ctxKeyDecoded[T any] struct{}

// ContextWithDecoded attaches a Decoded[T] to the context.
func ContextWithDecoded[T any](ctx context.Context, db goskema.Decoded[T]) context.Context {
	return context.WithValue(ctx, ctxKeyDecoded[T]{}, db)
}

// DecodedFromContext retrieves a Decoded[T] from context.
func DecodedFromContext[T any](ctx context.Context) (goskema.Decoded[T], bool) {
	v, ok := ctx.Value(ctxKeyDecoded[T]{}).(goskema.Decoded[T])
	return v, ok
}

// DefaultParseOpt returns a recommended default for HTTP JSON boundaries.
// - Duplicate keys are errors
// - Presence is collected for preserve-friendly semantics
func DefaultParseOpt() goskema.ParseOpt {
	return goskema.ParseOpt{
		Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
		Presence:   goskema.PresenceOpt{Collect: true},
	}
}

// ErrorPayload shapes Issues for JSON responses.
func ErrorPayload(issues []goskema.Issue) map[string]any {
	return map[string]any{"issues": issues}
}
