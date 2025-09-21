package user

import (
	"context"

	d "github.com/reoring/goskema/dsl"
)

// DSL returns a minimal user schema: {name:string, active:bool}
// name is required, active has default true.
var DSL = func() any {
	b := d.Object().
		Field("name", d.StringOf[string]()).Required().
		Field("active", d.BoolOf[bool]()).Default(true).
		UnknownStrict()
	s, _ := b.Build()
	// return as any to avoid compile-time coupling
	return s
}

// keep a reference to context to avoid unused import warning in minimal file
var _ = context.Background
