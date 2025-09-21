package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// ObjectTyped returns a typed object builder that supports fluent Bind()/MustBind().
// This achieves chain-style API without method type parameters by parameterizing the builder type itself.
func ObjectTyped[T any]() *objectBuilderT[T] { return &objectBuilderT[T]{inner: Object()} }

type objectBuilderT[T any] struct{ inner *objectBuilder }

// fieldStepT is a typed variant of fieldStep that enables
// chain-friendly APIs like Field(...).Required().
type fieldStepT[T any] struct {
	tb   *objectBuilderT[T]
	name string
}

// Field registers a field and returns a typed field step for chaining.
func (tb *objectBuilderT[T]) Field(name string, ad AnyAdapter) *fieldStepT[T] {
	tb.inner.Field(name, ad)
	return &fieldStepT[T]{tb: tb, name: name}
}
func (tb *objectBuilderT[T]) Require(names ...string) *objectBuilderT[T] {
	tb.inner.Require(names...)
	return tb
}
func (tb *objectBuilderT[T]) UnknownStrict() *objectBuilderT[T] { tb.inner.UnknownStrict(); return tb }
func (tb *objectBuilderT[T]) UnknownStrip() *objectBuilderT[T]  { tb.inner.UnknownStrip(); return tb }
func (tb *objectBuilderT[T]) UnknownPassthrough(target string) *objectBuilderT[T] {
	tb.inner.UnknownPassthrough(target)
	return tb
}
func (tb *objectBuilderT[T]) Refine(name string, fn func(context.Context, map[string]any) error) *objectBuilderT[T] {
	tb.inner.Refine(name, fn)
	return tb
}

// RefineT registers a typed domain rule (default PhaseDomain).
func (tb *objectBuilderT[T]) RefineT(name string, fn func(goskema.DomainCtx[T], T) []goskema.Issue, opt ...goskema.RefineOpt[T]) *objectBuilderT[T] {
	if fn == nil {
		return tb
	}
	var o goskema.RefineOpt[T]
	if len(opt) > 0 {
		o = opt[len(opt)-1]
	}
	tb.inner.addTypedRuleOpaque(typedRule[T]{name: name, fn: fn, opt: o})
	return tb
}

// RefineCtx registers a typed context rule (I/O allowed; PhaseContext).
func (tb *objectBuilderT[T]) RefineCtx(name string, fn func(goskema.DomainCtx[T], T) []goskema.Issue, opt ...goskema.RefineOpt[T]) *objectBuilderT[T] {
	if fn == nil {
		return tb
	}
	o := goskema.RefineOpt[T]{}
	if len(opt) > 0 {
		o = opt[len(opt)-1]
	}
	o.Phase = goskema.PhaseContext
	tb.inner.addTypedRuleOpaque(typedRule[T]{name: name, fn: fn, opt: o})
	return tb
}

// RefineCtxE registers a typed context rule that can return a fatal error in addition to Issues.
// A non-nil error signals dependency outage or other temporary failure (map to 5xx at API layer).
func (tb *objectBuilderT[T]) RefineCtxE(name string, fn func(goskema.DomainCtx[T], T) ([]goskema.Issue, error), opt ...goskema.RefineOpt[T]) *objectBuilderT[T] {
	if fn == nil {
		return tb
	}
	o := goskema.RefineOpt[T]{}
	if len(opt) > 0 {
		o = opt[len(opt)-1]
	}
	o.Phase = goskema.PhaseContext
	tb.inner.addTypedRuleOpaque(typedRuleE[T]{name: name, fn: fn, opt: o})
	return tb
}

// Bind builds and binds to T.
func (tb *objectBuilderT[T]) Bind() (goskema.Schema[T], error) { return Bind[T](tb.inner) }

// MustBind builds and binds to T, panicking on error.
func (tb *objectBuilderT[T]) MustBind() goskema.Schema[T] { return MustBind[T](tb.inner) }

// ObjectOf is an alias of ObjectTyped for naming consistency with other *Of[T] helpers.
func ObjectOf[T any]() *objectBuilderT[T] { return ObjectTyped[T]() }

// ----- fieldStepT methods -----

// Required marks the current field as required and returns the typed builder.
func (f *fieldStepT[T]) Required() *objectBuilderT[T] {
	f.tb.inner.Require(f.name)
	return f.tb
}

// Optional marks the current field as optional and returns the typed builder.
func (f *fieldStepT[T]) Optional() *objectBuilderT[T] {
	delete(f.tb.inner.required, f.name)
	return f.tb
}

// Deprecated: Prefer Field(...).Required() for a single field.
// Use the builder-level Require("a","b") for marking multiple fields at once.
// This method remains for backward compatibility but will be removed in a future major release.
func (f *fieldStepT[T]) Require(names ...string) *objectBuilderT[T] { return f.tb.Require(names...) }
func (f *fieldStepT[T]) UnknownStrict() *objectBuilderT[T]          { return f.tb.UnknownStrict() }
func (f *fieldStepT[T]) UnknownStrip() *objectBuilderT[T]           { return f.tb.UnknownStrip() }
func (f *fieldStepT[T]) UnknownPassthrough(target string) *objectBuilderT[T] {
	return f.tb.UnknownPassthrough(target)
}

// Default sets a default for the current field and exports it to JSON Schema.
func (f *fieldStepT[T]) Default(v any) *objectBuilderT[T] {
	ad := f.tb.inner.fields[f.name]
	// Apply default by parsing via the field schema to leverage Normalize/Validate/Refine
	ad.applyDefault = func(ctx context.Context) (any, error) { return ad.parse(ctx, v) }
	prev := ad.jsonSchema
	ad.jsonSchema = func() (*js.Schema, error) {
		if prev == nil {
			return &js.Schema{Default: v}, nil
		}
		s, err := prev()
		if err != nil {
			return nil, err
		}
		if s == nil {
			s = &js.Schema{}
		}
		s.Default = v
		return s, nil
	}
	f.tb.inner.fields[f.name] = ad
	return f.tb
}

func (f *fieldStepT[T]) Refine(name string, fn func(context.Context, map[string]any) error) *objectBuilderT[T] {
	return f.tb.Refine(name, fn)
}

// Forward helpers to keep chaining ergonomics.
func (f *fieldStepT[T]) Field(name string, ad AnyAdapter) *fieldStepT[T] { return f.tb.Field(name, ad) }
func (f *fieldStepT[T]) Bind() (goskema.Schema[T], error)                { return f.tb.Bind() }
func (f *fieldStepT[T]) MustBind() goskema.Schema[T]                     { return f.tb.MustBind() }
