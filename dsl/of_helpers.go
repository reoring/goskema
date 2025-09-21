package dsl

import (
	goskema "github.com/reoring/goskema"
)

// SchemaOf converts an arbitrary Schema[T] into an AnyAdapter helper.
// Use it as the drop-in replacement for the legacy API.
func SchemaOf[T any](s goskema.Schema[T]) AnyAdapter { return anyAdapterFromSchema[T](s) }

// ArrayOfSchema converts a constrained ArrayBuilder[E] into an AnyAdapter.
// Example: Field("tags", ArrayOfSchema[string](Array(String()).Min(2)))
func ArrayOfSchema[E any](ab ArrayBuilder[E]) AnyAdapter { return anyAdapterFromSchema[[]E](ab) }
