package goskema

import (
	"reflect"
	"strings"
)

// FieldToken identifies a top-level struct field of T using the DSL key name.
// Obtain it via FieldOf to ensure compile-time linkage to the struct field.
// It intentionally supports only top-level fields of T.
type FieldToken[T any] struct {
	key string
}

// Key returns the DSL key name associated with this field token.
func (t FieldToken[T]) Key() string { return t.key }

// FieldPathToken identifies a nested struct field path of T using DSL key names.
// Produced by PathOf. Keys are top-level-first.
type FieldPathToken[T any] struct {
	keys []string
}

// Keys returns the DSL key path segments.
func (t FieldPathToken[T]) Keys() []string { return append([]string(nil), t.keys...) }

// FieldNameOf returns the DSL key name for a top-level field of S selected by selector.
// Example: FieldNameOf[OrderItem](func(i *OrderItem) *string { return &i.SKU }) -> "sku".
func FieldNameOf[S any, F any](selector func(*S) *F) string {
	if selector == nil {
		panic("goskema.FieldNameOf: selector must not be nil")
	}
	var zero S
	fp := reflect.ValueOf(selector(&zero)).Pointer()
	rv := reflect.ValueOf(&zero).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := rv.Field(i)
		if fv.CanAddr() && fv.Addr().Pointer() == fp {
			name := ResolveStructKey(sf)
			if name == "" || name == "-" {
				panic("goskema.FieldNameOf: selected field is not exported or disabled")
			}
			return name
		}
	}
	panic("goskema.FieldNameOf: selector must return address of a top-level field")
}

// FieldOf builds a FieldToken for a top-level field of T.
// The selector must return the address of a top-level field, e.g.:
//
//	FieldOf[Order](func(o *Order) *string { return &o.Status })
//
// This guarantees compile-time errors if the field is renamed/removed.
func FieldOf[T any, F any](selector func(*T) *F) FieldToken[T] {
	if selector == nil {
		panic("goskema.FieldOf: selector must not be nil")
	}
	var zero T
	// Get pointer to selected field within zero value of T
	fp := reflect.ValueOf(selector(&zero)).Pointer()

	rv := reflect.ValueOf(&zero).Elem()
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		if !fv.CanAddr() {
			continue
		}
		if fv.Addr().Pointer() == fp {
			name := ResolveStructKey(rt.Field(i))
			if name == "" || name == "-" {
				panic("goskema.FieldOf: selected field is not exported or disabled")
			}
			return FieldToken[T]{key: name}
		}
	}
	panic("goskema.FieldOf: selector must return address of a top-level field of T")
}

// PathOf builds a FieldPathToken for an arbitrary nested field of T.
// The selector must return the address of a nested field, e.g.:
//
//	PathOf[Order, string](func(o *Order) *string { return &o.User.UserID })
//
// Limitations: Only descends through struct fields (non-pointer). Pointer hops
// are not supported in this initial version.
func PathOf[T any, F any](selector func(*T) *F) FieldPathToken[T] {
	if selector == nil {
		panic("goskema.PathOf: selector must not be nil")
	}
	var zero T
	target := reflect.ValueOf(selector(&zero)).Pointer()
	keys, ok := findPathKeys[T](reflect.ValueOf(&zero).Elem(), target, 0)
	if !ok || len(keys) == 0 {
		panic("goskema.PathOf: selector must address a nested struct field (non-pointer)")
	}
	return FieldPathToken[T]{keys: keys}
}

const _maxPathDepth = 32

func findPathKeys[T any](v reflect.Value, target uintptr, depth int) ([]string, bool) {
	if depth > _maxPathDepth {
		return nil, false
	}
	t := v.Type()
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		fv := v.Field(i)
		if fv.CanAddr() && fv.Addr().Pointer() == target {
			name := ResolveStructKey(sf)
			if name == "" || name == "-" {
				return nil, false
			}
			return []string{name}, true
		}
		// Recurse into nested structs only (skip pointers for safety)
		if fv.Kind() == reflect.Struct {
			if rest, ok := findPathKeys[T](fv, target, depth+1); ok {
				name := ResolveStructKey(sf)
				if name == "" || name == "-" {
					return nil, false
				}
				return append([]string{name}, rest...), true
			}
		}
	}
	return nil, false
}

// Seen reports whether the given field was present in input (PresenceSeen).
func (dc DomainCtx[T]) Seen(field FieldToken[T]) bool {
	if dc.Presence == nil {
		return false
	}
	return dc.Presence["/"+field.key]&PresenceSeen != 0
}

// WasNull reports whether the given field was explicitly null in input.
func (dc DomainCtx[T]) WasNull(field FieldToken[T]) bool {
	if dc.Presence == nil {
		return false
	}
	return dc.Presence["/"+field.key]&PresenceWasNull != 0
}

// DefaultApplied reports whether the given field value was materialized from a default.
func (dc DomainCtx[T]) DefaultApplied(field FieldToken[T]) bool {
	if dc.Presence == nil {
		return false
	}
	return dc.Presence["/"+field.key]&PresenceDefaultApplied != 0
}

// AnySeen reports whether any of the given fields were present.
func (dc DomainCtx[T]) AnySeen(fields ...FieldToken[T]) bool {
	for _, f := range fields {
		if dc.Seen(f) {
			return true
		}
	}
	return false
}

// AnySeenDeep reports whether any of the given top-level fields or their descendants
// were present in the input. This is useful for PATCH-safe gating when a nested
// property under a field changed.
func (dc DomainCtx[T]) AnySeenDeep(fields ...FieldToken[T]) bool {
	if dc.Presence == nil {
		return false
	}
	for _, f := range fields {
		base := "/" + f.key
		// direct field presence
		if dc.Presence[base]&PresenceSeen != 0 {
			return true
		}
		// descendant presence (e.g., /field/0, /field/name)
		for k, v := range dc.Presence {
			if v&PresenceSeen == 0 {
				continue
			}
			if strings.HasPrefix(k, base+"/") {
				return true
			}
		}
	}
	return false
}

// Path returns a PathRef anchored at the given top-level field token for T.
// This provides a typed alternative to At("/field").
func (dc DomainCtx[T]) Path(field FieldToken[T]) PathRef {
	if dc.Ref == nil {
		return NewRef(dc.Presence).Root().Field(field.key)
	}
	return dc.Ref.Root().Field(field.key)
}

// PathTo returns a PathRef anchored at the given nested field token for T.
func (dc DomainCtx[T]) PathTo(token FieldPathToken[T]) PathRef {
	var pr PathRef
	if dc.Ref == nil {
		pr = NewRef(dc.Presence).Root()
	} else {
		pr = dc.Ref.Root()
	}
	for _, k := range token.keys {
		pr = pr.Field(k)
	}
	return pr
}
