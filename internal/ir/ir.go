package ir

// Package ir defines the minimal intermediate representation used by the
// code generator. This package is internal and not part of the public API.

// NodeKind identifies an IR node type.
type NodeKind int

const (
	NodePrimitive NodeKind = iota
	NodeArray
	NodeObject
	NodeOneOf
)

// Schema is the root IR node interface.
type Schema interface {
	Kind() NodeKind
}

// Primitive represents string/bool/number primitives.
type Primitive struct {
	Name string // "string"|"bool"|"number" (JSON compatible names)
}

func (p *Primitive) Kind() NodeKind { return NodePrimitive }

// Array represents an array of items.
type Array struct {
	Item Schema
}

func (a *Array) Kind() NodeKind { return NodeArray }

// Object represents an object with fields and policies.
type Object struct {
	Fields        []Field
	Required      map[string]struct{}
	UnknownPolicy int // mirrors goskema.UnknownPolicy; kept as int to decouple layers
	UnknownTarget string
	RefineHooks   []string // placeholder identifiers for refine functions
}

func (o *Object) Kind() NodeKind { return NodeObject }

// Field maps a JSON name to a Schema and optional binding info.
type Field struct {
	Name       string // JSON name (post-alias resolution)
	Schema     Schema
	DomainPath string // optional: binding target path in domain struct
	Aliases    []string
	Default    any // optional materialized default (wire shape)
}

// OneOf represents a discriminated union.
type OneOf struct {
	Discriminator string
	Mapping       map[string]Schema // discriminator value -> variant schema
}

func (u *OneOf) Kind() NodeKind { return NodeOneOf }
