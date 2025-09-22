package rules

import (
	"fmt"
	"reflect"
	"strings"

	goskema "github.com/reoring/goskema"
)

// Op defines simple comparison operators for If(...).Then(...)
type Op int

const (
	Eq Op = iota
	Ne
	Lt
	Le
	Gt
	Ge
)

// Conditional composes conditional execution of rules.
type Conditional[T any] struct {
	path string
	op   Op
	want any
	all  []Conditional[T] // composite AND
	any  []Conditional[T] // composite OR
}

// If builds a conditional that evaluates a path against a value using an operator.
// The path is a JSON Pointer like "/status" using DSL keys. Top-level only is required for MVP.
func If[T any](path string, op Op, want any) Conditional[T] {
	return Conditional[T]{path: normalizePath(path), op: op, want: want}
}

// IfAll builds a conditional that requires all conditions to hold.
func IfAll[T any](conds ...Conditional[T]) Conditional[T] { return Conditional[T]{all: conds} }

// IfAny builds a conditional that requires any condition to hold.
func IfAny[T any](conds ...Conditional[T]) Conditional[T] { return Conditional[T]{any: conds} }

// And combines the receiver with additional conditions using logical AND.
func (c Conditional[T]) And(others ...Conditional[T]) Conditional[T] {
	conds := append([]Conditional[T]{c}, others...)
	return IfAll(conds...)
}

// Or combines the receiver with additional conditions using logical OR.
func (c Conditional[T]) Or(others ...Conditional[T]) Conditional[T] {
	conds := append([]Conditional[T]{c}, others...)
	return IfAny(conds...)
}

// Then attaches rules to run when the condition is satisfied.
func (c Conditional[T]) Then(rules ...func(goskema.DomainCtx[T], T) []goskema.Issue) func(goskema.DomainCtx[T], T) []goskema.Issue {
	return func(d goskema.DomainCtx[T], v T) []goskema.Issue {
		if !evalConditional(d, v, c) {
			return nil
		}
		var all []goskema.Issue
		for _, r := range rules {
			if r == nil {
				continue
			}
			if iss := r(d, v); len(iss) > 0 {
				all = append(all, iss...)
			}
			if len(all) > 0 && goskema.IsFailFast(d.Ctx) {
				return all
			}
		}
		return all
	}
}

// AtLeastOne ensures the collection at collectionPath has at least 1 element.
func AtLeastOne[T any](collectionPath string) func(goskema.DomainCtx[T], T) []goskema.Issue {
	p := normalizePath(collectionPath)
	return func(d goskema.DomainCtx[T], v T) []goskema.Issue {
		val, ok := valueAtPath(v, p)
		if !ok {
			return nil
		}
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			if rv.Len() == 0 {
				return []goskema.Issue{d.Ref.At(p).Issue(goskema.CodeTooShort, "at least 1 item is required", "minItems", 1)}
			}
		default:
			// Not a collection; do not issue error here to avoid noise
		}
		return nil
	}
}

// UniqueBy ensures elements in a collection have unique key values.
// collectionPath is JSON Pointer to a slice field (e.g., "/items").
// keyPath is a relative path inside each element (e.g., "sku" or "/sku").
// Note: Prefer a stable, comparable key type (e.g., string). Mixed-type keys may stringify
// to identical values and cause false positives. Align schema so the key is a single type.
func UniqueBy[T any](collectionPath, keyPath string) func(goskema.DomainCtx[T], T) []goskema.Issue {
	cp := normalizePath(collectionPath)
	kp := strings.TrimPrefix(keyPath, "/")
	return func(d goskema.DomainCtx[T], v T) []goskema.Issue {
		val, ok := valueAtPath(v, cp)
		if !ok {
			return nil
		}
		rv := reflect.ValueOf(val)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return nil
		}
		seen := map[string]int{}
		var out []goskema.Issue
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			kv, ok := valueAtPathWithin(elem, kp)
			if !ok {
				continue
			}
			key := fmt.Sprint(kv)
			if j, dup := seen[key]; dup {
				out = append(out, d.Ref.At(cp).Index(i).Field(kp).Issue(
					goskema.CodeUniqueness,
					"duplicate value",
					"first", j, "dup", i, "key", key,
				))
			} else {
				seen[key] = i
			}
		}
		return out
	}
}

// ------- helpers -------

func normalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	if p[0] != '/' {
		return "/" + p
	}
	return p
}

func evalConditional[T any](d goskema.DomainCtx[T], v T, c Conditional[T]) bool {
	// composite AND
	if len(c.all) > 0 {
		for _, it := range c.all {
			if !evalConditional(d, v, it) {
				return false
			}
		}
		return true
	}
	// composite OR
	if len(c.any) > 0 {
		for _, it := range c.any {
			if evalConditional(d, v, it) {
				return true
			}
		}
		return false
	}
	// simple predicate
	cur, ok := valueAtPath(v, c.path)
	if !ok {
		return false
	}
	return compare(cur, c.op, c.want)
}

// valueAtPath navigates v (struct/map) by JSON Pointer using DSL keys for struct fields.
func valueAtPath[T any](v T, pointer string) (any, bool) {
	return valueAtPathWithin(v, strings.TrimPrefix(pointer, "/"))
}

func valueAtPathWithin(v any, rel string) (any, bool) {
	if rel == "" {
		return v, true
	}
	cur := reflect.ValueOf(v)
	parts := strings.Split(rel, "/")
	for _, seg := range parts {
		if !cur.IsValid() {
			return nil, false
		}
		if cur.Kind() == reflect.Pointer {
			if cur.IsNil() {
				return nil, false
			}
			cur = cur.Elem()
		}
		switch cur.Kind() {
		case reflect.Struct:
			found := false
			rt := cur.Type()
			for i := 0; i < rt.NumField(); i++ {
				sf := rt.Field(i)
				if !sf.IsExported() {
					continue
				}
				name := goskema.ResolveStructKey(sf)
				if name == seg {
					cur = cur.Field(i)
					found = true
					break
				}
			}
			if !found {
				return nil, false
			}
		case reflect.Map:
			key := reflect.ValueOf(seg)
			mv := cur.MapIndex(key)
			if !mv.IsValid() {
				return nil, false
			}
			cur = mv
		case reflect.Slice, reflect.Array:
			// For collection, seg should be an index
			idx, ok := tryParseInt(seg)
			if !ok {
				return nil, false
			}
			if idx < 0 || idx >= cur.Len() {
				return nil, false
			}
			cur = cur.Index(idx)
		default:
			return nil, false
		}
	}
	if cur.Kind() == reflect.Pointer && !cur.IsNil() {
		cur = cur.Elem()
	}
	return cur.Interface(), true
}

func compare(cur any, op Op, want any) bool {
	switch op {
	case Eq:
		return reflect.DeepEqual(cur, want)
	case Ne:
		return !reflect.DeepEqual(cur, want)
	case Lt, Le, Gt, Ge:
		// Support ints and floats only for MVP
		return compareOrdered(cur, op, want)
	default:
		return false
	}
}

func compareOrdered(cur any, op Op, want any) bool {
	c := reflect.ValueOf(cur)
	w := reflect.ValueOf(want)
	if isIntLike(c.Kind()) && isIntLike(w.Kind()) {
		a := toInt64(c)
		b := toInt64(w)
		switch op {
		case Lt:
			return a < b
		case Le:
			return a <= b
		case Gt:
			return a > b
		case Ge:
			return a >= b
		}
	}
	if isFloatLike(c.Kind()) && isFloatLike(w.Kind()) {
		a := toFloat64(c)
		b := toFloat64(w)
		switch op {
		case Lt:
			return a < b
		case Le:
			return a <= b
		case Gt:
			return a > b
		case Ge:
			return a >= b
		}
	}
	return false
}

func isIntLike(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloatLike(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func toInt64(v reflect.Value) int64 {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint())
	default:
		return 0
	}
}

func toFloat64(v reflect.Value) float64 {
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return 0
	}
}

func tryParseInt(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	neg := false
	for i, r := range s {
		if i == 0 && r == '-' {
			neg = true
			continue
		}
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}

// ---------- Rule combinators ----------

// Rule is an alias for a typed rule function.
type Rule[T any] = func(goskema.DomainCtx[T], T) []goskema.Issue

// And executes all rules and concatenates Issues. FailFast なら途中短絡。
func And[T any](rules ...Rule[T]) Rule[T] {
	return func(d goskema.DomainCtx[T], v T) []goskema.Issue {
		var out []goskema.Issue
		for _, r := range rules {
			if r == nil {
				continue
			}
			if iss := r(d, v); len(iss) > 0 {
				out = append(out, iss...)
				if goskema.IsFailFast(d.Ctx) {
					return out
				}
			}
		}
		return out
	}
}

// Or succeeds if any rule returns no Issues. 全失敗のときは最小Issue数の枝を返す。
func Or[T any](rules ...Rule[T]) Rule[T] {
	return func(d goskema.DomainCtx[T], v T) []goskema.Issue {
		var best []goskema.Issue
		bestSet := false
		for _, r := range rules {
			if r == nil {
				continue
			}
			iss := r(d, v)
			if len(iss) == 0 {
				return nil
			}
			if !bestSet || len(iss) < len(best) {
				best = iss
				bestSet = true
			}
		}
		if bestSet {
			return best
		}
		return nil
	}
}
