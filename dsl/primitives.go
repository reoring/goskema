package dsl

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"strconv"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	eng "github.com/reoring/goskema/internal/engine"
	js "github.com/reoring/goskema/jsonschema"
)

// String returns the minimal string schema implementation.
func String() goskema.Schema[string] { return stringSchema{} }

// Bool returns the minimal bool schema implementation.
func Bool() goskema.Schema[bool] { return boolSchema{} }

// StringOf returns an AnyAdapter for a string wire schema projected to domain type T.
// Wraps String() schema for domain-specific string projection.
// stringAsSchema wraps stringSchema and projects to a domain type T with underlying string.
type stringAsSchema[T ~string] struct{}

func (stringAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	s, err := (stringSchema{}).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(s), nil
}

func (stringAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	ds, err := (stringSchema{}).ParseWithMeta(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(ds.Value), Presence: ds.Presence}, nil
}

// ---- streaming SPI ----
func (stringAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	s, err := (stringSchema{}).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(s), nil
}
func (stringAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	ds, err := (stringSchema{}).ParseFromSourceWithMeta(ctx, src, opt)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(ds.Value), Presence: ds.Presence}, nil
}

func (stringAsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (stringSchema{}).TypeCheck(ctx, v)
}
func (stringAsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (stringSchema{}).RuleCheck(ctx, v)
}
func (stringAsSchema[T]) Validate(ctx context.Context, v any) error {
	return (stringSchema{}).Validate(ctx, v)
}
func (stringAsSchema[T]) ValidateValue(ctx context.Context, v T) error {
	return (stringSchema{}).ValidateValue(ctx, string(v))
}
func (stringAsSchema[T]) JSONSchema() (*js.Schema, error) { return (stringSchema{}).JSONSchema() }

func StringOf[T ~string]() AnyAdapter {
	ad := anyAdapterFromSchema[T](stringAsSchema[T]{})
	ad.orig = stringSchema{}
	return ad
}

// BoolOf returns an AnyAdapter for a bool wire schema projected to domain type T.
// Wraps Bool() schema for domain-specific bool projection.
// boolAsSchema wraps boolSchema and projects to a domain type T with underlying bool.
type boolAsSchema[T ~bool] struct{}

func (boolAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	b, err := (boolSchema{}).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(b), nil
}
func (boolAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	db, err := (boolSchema{}).ParseWithMeta(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(db.Value), Presence: db.Presence}, nil
}

// ---- streaming SPI ----
func (boolAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	b, err := (boolSchema{}).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(b), nil
}
func (boolAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	db, err := (boolSchema{}).ParseFromSourceWithMeta(ctx, src, opt)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(db.Value), Presence: db.Presence}, nil
}
func (boolAsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (boolSchema{}).TypeCheck(ctx, v)
}
func (boolAsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (boolSchema{}).RuleCheck(ctx, v)
}
func (boolAsSchema[T]) Validate(ctx context.Context, v any) error {
	return (boolSchema{}).Validate(ctx, v)
}
func (boolAsSchema[T]) ValidateValue(ctx context.Context, v T) error {
	return (boolSchema{}).ValidateValue(ctx, bool(v))
}
func (boolAsSchema[T]) JSONSchema() (*js.Schema, error) { return (boolSchema{}).JSONSchema() }

func BoolOf[T ~bool]() AnyAdapter {
	ad := anyAdapterFromSchema[T](boolAsSchema[T]{})
	ad.orig = boolSchema{}
	return ad
}

// NumberBuilder exposes chaining options for number schemas while implementing Schema[json.Number].
type NumberBuilder interface {
	goskema.Schema[json.Number]
	CoerceFromString() NumberBuilder
}

// NumberJSON returns the minimal json.Number schema implementation (no string coerce by default).
func NumberJSON() NumberBuilder { return &numberJSONSchema{} }

type stringSchema struct{}

type boolSchema struct{}

// numberJSONSchema implements NumberBuilder with optional string coercion.
type numberJSONSchema struct{ coerceFromString bool }

func (n *numberJSONSchema) CoerceFromString() NumberBuilder {
	n.coerceFromString = true
	return n
}

func (stringSchema) Parse(ctx context.Context, v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	// Normalize -> ValidateValue -> Refine
	ns, err := goskema.ApplyNormalize[string](ctx, s, stringSchema{})
	if err != nil {
		return "", err
	}
	s = ns
	if err := (stringSchema{}).ValidateValue(ctx, s); err != nil {
		return "", err
	}
	if err := goskema.ApplyRefine[string](ctx, s, stringSchema{}); err != nil {
		return "", err
	}
	return s, nil
}

func (stringSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[string], error) {
	s, err := (stringSchema{}).Parse(ctx, v)
	return goskema.Decoded[string]{Value: s, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

// ---- streaming SPI ----
func (stringSchema) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (string, error) {
	engSrc := goskema.EngineTokenSource(src)
	tok, err := engSrc.NextToken()
	if err != nil {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if tok.Kind != eng.KindString {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	s := tok.String
	ns, nerr := goskema.ApplyNormalize[string](ctx, s, stringSchema{})
	if nerr != nil {
		return "", nerr
	}
	if err := (stringSchema{}).ValidateValue(ctx, ns); err != nil {
		return "", err
	}
	if err := goskema.ApplyRefine[string](ctx, ns, stringSchema{}); err != nil {
		return "", err
	}
	return ns, nil
}

func (stringSchema) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[string], error) {
	v, err := (stringSchema{}).ParseFromSource(ctx, src, opt)
	return goskema.Decoded[string]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (stringSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(string); !ok {
		return goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	return nil
}

func (stringSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (stringSchema) Validate(ctx context.Context, v any) error {
	if err := (stringSchema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (stringSchema{}).RuleCheck(ctx, v)
}

func (stringSchema) ValidateValue(ctx context.Context, v string) error { return nil }

func (stringSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{Type: "string"}, nil }

func (boolSchema) Parse(ctx context.Context, v any) (bool, error) {
	b, ok := v.(bool)
	if !ok {
		return false, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	nb, err := goskema.ApplyNormalize[bool](ctx, b, boolSchema{})
	if err != nil {
		return false, err
	}
	b = nb
	if err := (boolSchema{}).ValidateValue(ctx, b); err != nil {
		return false, err
	}
	if err := goskema.ApplyRefine[bool](ctx, b, boolSchema{}); err != nil {
		return false, err
	}
	return b, nil
}

func (boolSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[bool], error) {
	b, err := (boolSchema{}).Parse(ctx, v)
	return goskema.Decoded[bool]{Value: b, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

// ---- streaming SPI ----
func (boolSchema) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (bool, error) {
	engSrc := goskema.EngineTokenSource(src)
	tok, err := engSrc.NextToken()
	if err != nil {
		return false, goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if tok.Kind != eng.KindBool {
		return false, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	b := tok.Bool
	nb, nerr := goskema.ApplyNormalize[bool](ctx, b, boolSchema{})
	if nerr != nil {
		return false, nerr
	}
	if err := (boolSchema{}).ValidateValue(ctx, nb); err != nil {
		return false, err
	}
	if err := goskema.ApplyRefine[bool](ctx, nb, boolSchema{}); err != nil {
		return false, err
	}
	return nb, nil
}

func (boolSchema) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[bool], error) {
	v, err := (boolSchema{}).ParseFromSource(ctx, src, opt)
	return goskema.Decoded[bool]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (boolSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(bool); !ok {
		return goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
	return nil
}

func (boolSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (boolSchema) Validate(ctx context.Context, v any) error {
	if err := (boolSchema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (boolSchema{}).RuleCheck(ctx, v)
}

func (boolSchema) ValidateValue(ctx context.Context, v bool) error { return nil }

func (boolSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{Type: "boolean"}, nil }

// ---------------- NumberOf[T] ----------------
// numberAsSchema wraps numberJSONSchema and projects to a domain type T with underlying string.
// json.Number is defined over string, so we project through string.
type numberAsSchema[T ~string] struct{ n numberJSONSchema }

func (s numberAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(string(num)), nil
}

func (s numberAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	dn, err := (&s.n).ParseWithMeta(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(string(dn.Value)), Presence: dn.Presence}, nil
}

// ---- streaming SPI ----
func (s numberAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	return T(string(num)), nil
}
func (s numberAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	dn, err := (&s.n).ParseFromSourceWithMeta(ctx, src, opt)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: T(string(dn.Value)), Presence: dn.Presence}, nil
}

func (s numberAsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s numberAsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s numberAsSchema[T]) Validate(ctx context.Context, v any) error { return (&s.n).Validate(ctx, v) }
func (s numberAsSchema[T]) ValidateValue(ctx context.Context, v T) error {
	return (&s.n).ValidateValue(ctx, json.Number(string(v)))
}
func (s numberAsSchema[T]) JSONSchema() (*js.Schema, error) { return (&s.n).JSONSchema() }

// NumberOf returns an AnyAdapter for a json.Number wire schema projected to domain type T.
// Wraps NumberJSON() schema for domain-specific number projection.
func NumberOf[T ~string]() AnyAdapter {
	ad := anyAdapterFromSchema[T](numberAsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- IntOf[T] ----------------
// intAsSchema wraps numberJSONSchema and projects to a domain type T with underlying int.
// It accepts JSON number on the wire and converts to int with integer-only semantics.
type intAsSchema[T ~int] struct{ n numberJSONSchema }

func (s intAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	// Allow direct int for default application ergonomics.
	switch t := v.(type) {
	case int:
		return T(t), nil
	case int8:
		return T(int(t)), nil
	case int16:
		return T(int(t)), nil
	case int32:
		return T(int(t)), nil
	case int64:
		return T(int(t)), nil
	case uint:
		return T(int(t)), nil
	case uint8:
		return T(int(t)), nil
	case uint16:
		return T(int(t)), nil
	case uint32:
		return T(int(t)), nil
	case uint64:
		// Best-effort downcast; overflow will be caught by Go's int range.
		return T(int(t)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	i64, perr := num.Int64()
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(int(i64)), nil
}

func (s intAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

// ---- streaming SPI ----
func (s intAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	i64, perr := num.Int64()
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(int(i64)), nil
}
func (s intAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s intAsSchema[T]) TypeCheck(ctx context.Context, v any) error   { return (&s.n).TypeCheck(ctx, v) }
func (s intAsSchema[T]) RuleCheck(ctx context.Context, v any) error   { return (&s.n).RuleCheck(ctx, v) }
func (s intAsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s intAsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s intAsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

// IntOf returns an AnyAdapter for a json.Number wire schema projected to domain type T(~int).
// It accepts JSON numbers like 1 or 2 (not strings unless NumberMode coerces) and decodes to T.
func IntOf[T ~int]() AnyAdapter {
	ad := anyAdapterFromSchema[T](intAsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- FloatOf[T] ----------------
// floatAsSchema wraps numberJSONSchema and projects to a domain type T with underlying float64.
// It accepts JSON numbers on the wire and converts to float64 using strconv.ParseFloat when necessary.
type floatAsSchema[T ~float64] struct{ n numberJSONSchema }

func (s floatAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	// Accept direct float64 for default application ergonomics
	if f, ok := v.(float64); ok {
		return T(f), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	// json.Number -> float64
	f64, perr := strconv.ParseFloat(num.String(), 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(f64), nil
}

func (s floatAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

// ---- streaming SPI ----
func (s floatAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	f64, perr := strconv.ParseFloat(num.String(), 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(f64), nil
}
func (s floatAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s floatAsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s floatAsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s floatAsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s floatAsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s floatAsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "number"}, nil }

// FloatOf returns an AnyAdapter for a json.Number wire schema projected to domain type T(~float64).
func FloatOf[T ~float64]() AnyAdapter {
	ad := anyAdapterFromSchema[T](floatAsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- UintOf[T] ----------------
// uintAsSchema projects json.Number to domain type T with underlying uint64.
// Accepts JSON numbers on the wire and converts using json.Number's parsing.
type uintAsSchema[T ~uint64] struct{ n numberJSONSchema }

func (s uintAsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	// Accept common unsigned ints directly for defaults/validation convenience
	switch t := v.(type) {
	case uint, uint8, uint16, uint32, uint64:
		return T(reflect.ValueOf(t).Convert(reflect.TypeOf(uint64(0))).Uint()), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	// Use ParseUint to avoid float precision/overflow pitfalls and reject negatives/fractions
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(u64), nil
}

func (s uintAsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

// ---- streaming SPI ----
func (s uintAsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	return T(u64), nil
}
func (s uintAsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s uintAsSchema[T]) TypeCheck(ctx context.Context, v any) error   { return (&s.n).TypeCheck(ctx, v) }
func (s uintAsSchema[T]) RuleCheck(ctx context.Context, v any) error   { return (&s.n).RuleCheck(ctx, v) }
func (s uintAsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s uintAsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s uintAsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

// UintOf returns an AnyAdapter for a json.Number wire schema projected to domain type T(~uint64).
func UintOf[T ~uint64]() AnyAdapter {
	ad := anyAdapterFromSchema[T](uintAsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// helper to produce *float64 for JSONSchema Minimum
func ptrFloat(v float64) *float64 { return &v }

// ---------------- Int32Of[T] ----------------
// int32AsSchema projects json.Number to domain type T with underlying int32.
type int32AsSchema[T ~int32] struct{ n numberJSONSchema }

func (s int32AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	// Accept direct ints if in range
	switch t := v.(type) {
	case int, int8, int16, int32, int64:
		i64 := reflect.ValueOf(t).Int()
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int32 overflow"}}
		}
		return T(int32(i64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	// Prefer Int64 when possible
	if i64, perr := num.Int64(); perr == nil {
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int32 overflow"}}
		}
		return T(int32(i64)), nil
	}
	// Fallback via float
	f64, perr := strconv.ParseFloat(num.String(), 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if math.Trunc(f64) != f64 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "fractional part not allowed for int32"}}
	}
	i64 := int64(f64)
	if i64 < math.MinInt32 || i64 > math.MaxInt32 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int32 overflow"}}
	}
	return T(int32(i64)), nil
}

func (s int32AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

// ---- streaming SPI ----
func (s int32AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	if i64, perr := strconv.ParseInt(num.String(), 10, 64); perr == nil {
		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int32 overflow"}}
		}
		return T(int32(i64)), nil
	}
	var zero T
	return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
}
func (s int32AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s int32AsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s int32AsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s int32AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s int32AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s int32AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

// Int32Of returns an AnyAdapter for a json.Number wire schema projected to domain type T(~int32).
func Int32Of[T ~int32]() AnyAdapter {
	ad := anyAdapterFromSchema[T](int32AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- Uint32Of[T] ----------------
// uint32AsSchema projects json.Number to domain type T with underlying uint32.
type uint32AsSchema[T ~uint32] struct{ n numberJSONSchema }

func (s uint32AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	switch t := v.(type) {
	case uint, uint8, uint16, uint32, uint64:
		u64 := reflect.ValueOf(t).Convert(reflect.TypeOf(uint64(0))).Uint()
		if u64 > math.MaxUint32 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint32 overflow"}}
		}
		return T(uint32(u64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint32 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint32 overflow"}}
	}
	return T(uint32(u64)), nil
}

func (s uint32AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

// ---- streaming SPI ----
func (s uint32AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint32 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint32 overflow"}}
	}
	return T(uint32(u64)), nil
}
func (s uint32AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s uint32AsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s uint32AsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s uint32AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s uint32AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s uint32AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

// Uint32Of returns an AnyAdapter for a json.Number wire schema projected to domain type T(~uint32).
func Uint32Of[T ~uint32]() AnyAdapter {
	ad := anyAdapterFromSchema[T](uint32AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- Int16Of[T] ----------------
type int16AsSchema[T ~int16] struct{ n numberJSONSchema }

func (s int16AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	switch t := v.(type) {
	case int, int8, int16, int32, int64:
		i64 := reflect.ValueOf(t).Int()
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int16 overflow"}}
		}
		return T(int16(i64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	if i64, perr := num.Int64(); perr == nil {
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int16 overflow"}}
		}
		return T(int16(i64)), nil
	}
	f64, perr := strconv.ParseFloat(num.String(), 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if math.Trunc(f64) != f64 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "fractional part not allowed for int16"}}
	}
	i64 := int64(f64)
	if i64 < math.MinInt16 || i64 > math.MaxInt16 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int16 overflow"}}
	}
	return T(int16(i64)), nil
}

func (s int16AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

func (s int16AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	if i64, perr := strconv.ParseInt(num.String(), 10, 64); perr == nil {
		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int16 overflow"}}
		}
		return T(int16(i64)), nil
	}
	var zero T
	return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
}

func (s int16AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s int16AsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s int16AsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s int16AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s int16AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s int16AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

func Int16Of[T ~int16]() AnyAdapter {
	ad := anyAdapterFromSchema[T](int16AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- Uint16Of[T] ----------------
type uint16AsSchema[T ~uint16] struct{ n numberJSONSchema }

func (s uint16AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	switch t := v.(type) {
	case uint, uint8, uint16, uint32, uint64:
		u64 := reflect.ValueOf(t).Convert(reflect.TypeOf(uint64(0))).Uint()
		if u64 > math.MaxUint16 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint16 overflow"}}
		}
		return T(uint16(u64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint16 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint16 overflow"}}
	}
	return T(uint16(u64)), nil
}

func (s uint16AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

func (s uint16AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint16 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint16 overflow"}}
	}
	return T(uint16(u64)), nil
}

func (s uint16AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s uint16AsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s uint16AsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s uint16AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s uint16AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s uint16AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

func Uint16Of[T ~uint16]() AnyAdapter {
	ad := anyAdapterFromSchema[T](uint16AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- Int8Of[T] ----------------
type int8AsSchema[T ~int8] struct{ n numberJSONSchema }

func (s int8AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	switch t := v.(type) {
	case int, int8, int16, int32, int64:
		i64 := reflect.ValueOf(t).Int()
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int8 overflow"}}
		}
		return T(int8(i64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	if i64, perr := num.Int64(); perr == nil {
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int8 overflow"}}
		}
		return T(int8(i64)), nil
	}
	f64, perr := strconv.ParseFloat(num.String(), 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if math.Trunc(f64) != f64 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "fractional part not allowed for int8"}}
	}
	i64 := int64(f64)
	if i64 < math.MinInt8 || i64 > math.MaxInt8 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int8 overflow"}}
	}
	return T(int8(i64)), nil
}

func (s int8AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

func (s int8AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	if i64, perr := strconv.ParseInt(num.String(), 10, 64); perr == nil {
		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "int8 overflow"}}
		}
		return T(int8(i64)), nil
	}
	var zero T
	return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
}

func (s int8AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s int8AsSchema[T]) TypeCheck(ctx context.Context, v any) error   { return (&s.n).TypeCheck(ctx, v) }
func (s int8AsSchema[T]) RuleCheck(ctx context.Context, v any) error   { return (&s.n).RuleCheck(ctx, v) }
func (s int8AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s int8AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s int8AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

func Int8Of[T ~int8]() AnyAdapter {
	ad := anyAdapterFromSchema[T](int8AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

// ---------------- Uint8Of[T] ----------------
type uint8AsSchema[T ~uint8] struct{ n numberJSONSchema }

func (s uint8AsSchema[T]) Parse(ctx context.Context, v any) (T, error) {
	switch t := v.(type) {
	case uint, uint8, uint16, uint32, uint64:
		u64 := reflect.ValueOf(t).Convert(reflect.TypeOf(uint64(0))).Uint()
		if u64 > math.MaxUint8 {
			var zero T
			return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint8 overflow"}}
		}
		return T(uint8(u64)), nil
	}
	num, err := (&s.n).Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint8 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint8 overflow"}}
	}
	return T(uint8(u64)), nil
}

func (s uint8AsSchema[T]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[T], error) {
	tv, err := s.Parse(ctx, v)
	if err != nil {
		var zero goskema.Decoded[T]
		return zero, err
	}
	return goskema.Decoded[T]{Value: tv, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}

func (s uint8AsSchema[T]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (T, error) {
	num, err := (&s.n).ParseFromSource(ctx, src, opt)
	if err != nil {
		var zero T
		return zero, err
	}
	u64, perr := strconv.ParseUint(num.String(), 10, 64)
	if perr != nil {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
	}
	if u64 > math.MaxUint8 {
		var zero T
		return zero, goskema.Issues{{Path: "/", Code: goskema.CodeOverflow, Message: "uint8 overflow"}}
	}
	return T(uint8(u64)), nil
}

func (s uint8AsSchema[T]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[T], error) {
	v, err := s.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[T]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (s uint8AsSchema[T]) TypeCheck(ctx context.Context, v any) error {
	return (&s.n).TypeCheck(ctx, v)
}
func (s uint8AsSchema[T]) RuleCheck(ctx context.Context, v any) error {
	return (&s.n).RuleCheck(ctx, v)
}
func (s uint8AsSchema[T]) Validate(ctx context.Context, v any) error    { return (&s.n).Validate(ctx, v) }
func (s uint8AsSchema[T]) ValidateValue(ctx context.Context, v T) error { return nil }
func (s uint8AsSchema[T]) JSONSchema() (*js.Schema, error)              { return &js.Schema{Type: "integer"}, nil }

func Uint8Of[T ~uint8]() AnyAdapter {
	ad := anyAdapterFromSchema[T](uint8AsSchema[T]{})
	ad.orig = numberJSONSchema{}
	return ad
}

func (n *numberJSONSchema) Parse(ctx context.Context, v any) (json.Number, error) {
	switch t := v.(type) {
	case json.Number:
		num := t
		nn, err := goskema.ApplyNormalize[json.Number](ctx, num, n)
		if err != nil {
			return json.Number(""), err
		}
		num = nn
		if err := n.ValidateValue(ctx, num); err != nil {
			return json.Number(""), err
		}
		if err := goskema.ApplyRefine[json.Number](ctx, num, n); err != nil {
			return json.Number(""), err
		}
		return num, nil
	case float64:
		return json.Number(strconvFormatFloat(t)), nil
	case string:
		if n.coerceFromString {
			if _, err := strconv.ParseFloat(t, 64); err != nil {
				return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: err}}
			}
			// Canonicalize via float64 formatting for consistency with float64 input
			if f, err := strconv.ParseFloat(t, 64); err == nil {
				return json.Number(strconvFormatFloat(f)), nil
			}
			return json.Number(t), nil
		}
		return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	default:
		return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
}

func (n *numberJSONSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[json.Number], error) {
	num, err := n.Parse(ctx, v)
	return goskema.Decoded[json.Number]{Value: num, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

// ---- streaming SPI ----
func (n *numberJSONSchema) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (json.Number, error) {
	engSrc := goskema.EngineTokenSource(src)
	tok, err := engSrc.NextToken()
	if err != nil {
		return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	switch tok.Kind {
	case eng.KindNumber:
		// honor NumberMode: JSONNumber vs float64 already handled at source driver level
		if src.NumberMode() == goskema.NumberFloat64 {
			// format float back to canonical string to preserve contract
			if f, perr := strconv.ParseFloat(tok.Number, 64); perr == nil {
				return json.Number(strconvFormatFloat(f)), nil
			}
		}
		return json.Number(tok.Number), nil
	case eng.KindString:
		if n.coerceFromString {
			if _, perr := strconv.ParseFloat(tok.String, 64); perr != nil {
				return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Cause: perr}}
			}
			if f, perr := strconv.ParseFloat(tok.String, 64); perr == nil {
				return json.Number(strconvFormatFloat(f)), nil
			}
			return json.Number(tok.String), nil
		}
		return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	default:
		return json.Number(""), goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
}

func (n *numberJSONSchema) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[json.Number], error) {
	v, err := n.ParseFromSource(ctx, src, opt)
	return goskema.Decoded[json.Number]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (n *numberJSONSchema) TypeCheck(ctx context.Context, v any) error {
	switch v.(type) {
	case json.Number, float64:
		return nil
	case string:
		if n.coerceFromString {
			return nil
		}
		return goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	default:
		return goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil)}}
	}
}

func (n *numberJSONSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (n *numberJSONSchema) Validate(ctx context.Context, v any) error {
	if err := n.TypeCheck(ctx, v); err != nil {
		return err
	}
	return n.RuleCheck(ctx, v)
}

func (n *numberJSONSchema) ValidateValue(ctx context.Context, v json.Number) error { return nil }

func (n *numberJSONSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{Type: "number"}, nil }

// strconvFormatFloat mirrors the canonical JSON-like float formatting.
func strconvFormatFloat(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }
