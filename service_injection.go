package goskema

import "context"

// serviceKey is a unique key per type parameter T for context storage.
type serviceKey[T any] struct{}

// WithService stores a typed service instance in the context for use by Context rules.
func WithService[T any](ctx context.Context, svc T) context.Context {
	return context.WithValue(ctx, serviceKey[T]{}, any(svc))
}

// Service retrieves a typed service instance from context.
func Service[T any](ctx context.Context) (T, bool) {
	var zero T
	v := ctx.Value(serviceKey[T]{})
	if v == nil {
		return zero, false
	}
	if tv, ok := v.(T); ok {
		return tv, true
	}
	return zero, false
}

// RequireService returns the service or an error suitable for bubbling into Issues by callers.
func RequireService[T any](ctx context.Context) (T, error) {
	if v, ok := Service[T](ctx); ok {
		return v, nil
	}
	var zero T
	return zero, Issues{Issue{Code: CodeDependencyUnavailable, Message: "service not provided"}}
}
