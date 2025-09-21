package goskema

import "context"

// ApplyNormalize calls Normalizer[T] if implemented.
func ApplyNormalize[T any](ctx context.Context, v T, s Schema[T]) (T, error) {
	if n, ok := any(s).(Normalizer[T]); ok {
		return n.Normalize(ctx, v)
	}
	return v, nil
}

// ApplyRefine calls Refiner[T] if implemented.
func ApplyRefine[T any](ctx context.Context, v T, s Schema[T]) error {
	if r, ok := any(s).(Refiner[T]); ok {
		return r.Refine(ctx, v)
	}
	return nil
}
