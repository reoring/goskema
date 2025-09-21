package dsl

import (
	"context"
	"sort"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

type objectBuilder struct {
	fields        map[string]AnyAdapter
	required      map[string]struct{}
	unknownPolicy goskema.UnknownPolicy
	unknownTarget string
	refines       []objRefine
	discriminator string
	variants      map[string]goskema.Schema[map[string]any]
	typedRules    []any // holds typedRule[T] values; retyped at Bind[T]
}

type fieldStep struct {
	b    *objectBuilder
	name string
}

// Object creates a new object builder with safe defaults (UnknownStrict).
func Object() *objectBuilder {
	return &objectBuilder{
		fields:        map[string]AnyAdapter{},
		required:      map[string]struct{}{},
		unknownPolicy: goskema.UnknownStrict,
		refines:       nil,
		discriminator: "",
		variants:      nil,
		typedRules:    nil,
	}
}

// Field registers a field with its adapter.
func (b *objectBuilder) Field(name string, ad AnyAdapter) *fieldStep {
	b.fields[name] = ad
	return &fieldStep{b: b, name: name}
}

// Required marks the field as required and returns the builder.
func (f *fieldStep) Required() *objectBuilder {
	f.b.required[f.name] = struct{}{}
	return f.b
}

// Optional marks the field as optional (default) and returns the builder.
func (f *fieldStep) Optional() *objectBuilder {
	delete(f.b.required, f.name)
	return f.b
}

// Deprecated: Prefer Field(...).Required() for a single field.
// Use the builder-level Require("a","b") for marking multiple fields at once.
// This method remains for backward compatibility but will be removed in a future major release.
func (f *fieldStep) Require(names ...string) *objectBuilder { return f.b.Require(names...) }
func (f *fieldStep) UnknownStrict() *objectBuilder          { return f.b.UnknownStrict() }
func (f *fieldStep) UnknownStrip() *objectBuilder           { return f.b.UnknownStrip() }
func (f *fieldStep) UnknownPassthrough(target string) *objectBuilder {
	return f.b.UnknownPassthrough(target)
}

// Default sets a default for the current field and exports it to JSON Schema.
func (f *fieldStep) Default(v any) *objectBuilder {
	ad := f.b.fields[f.name]
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
	f.b.fields[f.name] = ad
	return f.b
}
func (f *fieldStep) Refine(name string, fn func(context.Context, map[string]any) error) *objectBuilder {
	return f.b.Refine(name, fn)
}
func (f *fieldStep) Field(name string, ad AnyAdapter) *fieldStep    { return f.b.Field(name, ad) }
func (f *fieldStep) Build() (goskema.Schema[map[string]any], error) { return f.b.Build() }
func (f *fieldStep) MustBuild() goskema.Schema[map[string]any]      { return f.b.MustBuild() }

// Require marks one or more fields as required.
func (b *objectBuilder) Require(names ...string) *objectBuilder {
	for _, n := range names {
		b.required[n] = struct{}{}
	}
	return b
}

// UnknownStrict sets unknown policy to Strict.
func (b *objectBuilder) UnknownStrict() *objectBuilder {
	b.unknownPolicy = goskema.UnknownStrict
	b.unknownTarget = ""
	return b
}

// UnknownStrip sets unknown policy to Strip.
func (b *objectBuilder) UnknownStrip() *objectBuilder {
	b.unknownPolicy = goskema.UnknownStrip
	b.unknownTarget = ""
	return b
}

// UnknownPassthrough sets unknown policy to Passthrough with a target field.
func (b *objectBuilder) UnknownPassthrough(target string) *objectBuilder {
	b.unknownPolicy = goskema.UnknownPassthrough
	b.unknownTarget = target
	return b
}

// Refine adds an object-level refine function. It is executed after Normalize/ValidateValue.
func (b *objectBuilder) Refine(name string, fn func(context.Context, map[string]any) error) *objectBuilder {
	if fn == nil {
		return b
	}
	b.refines = append(b.refines, objRefine{name: name, fn: fn})
	return b
}

// addTypedRuleOpaque appends a typed rule instance stored in an opaque form (any).
// The value should be a typedRule[T] constructed by the typed builder.
func (b *objectBuilder) addTypedRuleOpaque(rule any) {
	b.typedRules = append(b.typedRules, rule)
}

// Discriminator sets the discriminator key for a discriminated union.
func (b *objectBuilder) Discriminator(key string) *objectBuilder {
	b.discriminator = key
	return b
}

// UnionVariant defines a named variant schema for discriminated unions.
type UnionVariant struct {
	name   string
	schema goskema.Schema[map[string]any]
}

// Variant constructs a UnionVariant.
func Variant(name string, s goskema.Schema[map[string]any]) UnionVariant {
	return UnionVariant{name: name, schema: s}
}

// OneOf registers union variants when a discriminator is set.
func (b *objectBuilder) OneOf(vars ...UnionVariant) *objectBuilder {
	if len(vars) == 0 {
		return b
	}
	if b.variants == nil {
		b.variants = make(map[string]goskema.Schema[map[string]any], len(vars))
	}
	for _, v := range vars {
		if v.name == "" || v.schema == nil {
			continue
		}
		b.variants[v.name] = v.schema
	}
	return b
}

// Build validates the builder and returns a Schema.
func (b *objectBuilder) Build() (goskema.Schema[map[string]any], error) {
	// If discriminator is configured, return a union schema
	if b.discriminator != "" && len(b.variants) > 0 {
		return &unionSchema{discriminator: b.discriminator, mapping: b.variants}, nil
	}
	// Validate unknown passthrough target
	if b.unknownPolicy == goskema.UnknownPassthrough {
		ad, ok := b.fields[b.unknownTarget]
		if !ok || b.unknownTarget == "" {
			return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: i18n.T(goskema.CodeParseError, nil), Hint: "unknown_target missing for passthrough"}}
		}
		// adapter must accept map[string]any (validateValue on empty map)
		if err := ad.validateValue(context.Background(), map[string]any{}); err != nil {
			return nil, goskema.Issues{goskema.Issue{Path: "/" + b.unknownTarget, Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "unknown_target must be map[string]any"}}
		}
	}
	// cache sorted keys for deterministic order without per-parse sorting
	kfs := make([]string, 0, len(b.fields))
	for k := range b.fields {
		kfs = append(kfs, k)
	}
	sort.Strings(kfs)
	return &objectSchema{fields: b.fields, required: b.required, unknownPolicy: b.unknownPolicy, unknownTarget: b.unknownTarget, refines: b.refines, typedRulesAny: b.typedRules, sortedKeys: kfs}, nil
}

// MustBuild is like Build but panics on error.
func (b *objectBuilder) MustBuild() goskema.Schema[map[string]any] {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}
	return s
}
