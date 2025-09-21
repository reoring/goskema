package dsl_test

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"
	"testing"
	"time"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
	js "github.com/reoring/goskema/jsonschema"
)

// ---- Minimal helper schemas for tests ----

type float64Schema struct{}

func (float64Schema) Parse(ctx context.Context, v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: err.Error()}}
		}
		return f, nil
	default:
		return 0, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected number"}}
	}
}
func (float64Schema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[float64], error) {
	f, err := (float64Schema{}).Parse(ctx, v)
	return goskema.Decoded[float64]{Value: f, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (float64Schema) TypeCheck(ctx context.Context, v any) error         { return nil }
func (float64Schema) RuleCheck(ctx context.Context, v any) error         { return nil }
func (float64Schema) Validate(ctx context.Context, v any) error          { return nil }
func (float64Schema) ValidateValue(ctx context.Context, v float64) error { return nil }
func (float64Schema) JSONSchema() (*js.Schema, error)                    { return &js.Schema{}, nil }

type int64Schema struct{}

func (int64Schema) Parse(ctx context.Context, v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case json.Number:
		iv, err := t.Int64()
		if err != nil {
			return 0, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: err.Error()}}
		}
		return iv, nil
	case int64:
		return t, nil
	default:
		return 0, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected integer"}}
	}
}
func (int64Schema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[int64], error) {
	i, err := (int64Schema{}).Parse(ctx, v)
	return goskema.Decoded[int64]{Value: i, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (int64Schema) TypeCheck(ctx context.Context, v any) error       { return nil }
func (int64Schema) RuleCheck(ctx context.Context, v any) error       { return nil }
func (int64Schema) Validate(ctx context.Context, v any) error        { return nil }
func (int64Schema) ValidateValue(ctx context.Context, v int64) error { return nil }
func (int64Schema) JSONSchema() (*js.Schema, error)                  { return &js.Schema{}, nil }

type bigIntSchema struct{}

func (bigIntSchema) Parse(ctx context.Context, v any) (*big.Int, error) {
	switch t := v.(type) {
	case *big.Int:
		return t, nil
	default:
		return nil, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected *big.Int"}}
	}
}
func (bigIntSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[*big.Int], error) {
	b, err := (bigIntSchema{}).Parse(ctx, v)
	return goskema.Decoded[*big.Int]{Value: b, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (bigIntSchema) TypeCheck(ctx context.Context, v any) error          { return nil }
func (bigIntSchema) RuleCheck(ctx context.Context, v any) error          { return nil }
func (bigIntSchema) Validate(ctx context.Context, v any) error           { return nil }
func (bigIntSchema) ValidateValue(ctx context.Context, v *big.Int) error { return nil }
func (bigIntSchema) JSONSchema() (*js.Schema, error)                     { return &js.Schema{}, nil }

type bytesSchema struct{}

func (bytesSchema) Parse(ctx context.Context, v any) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	return nil, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected []byte"}}
}
func (bytesSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[[]byte], error) {
	b, err := (bytesSchema{}).Parse(ctx, v)
	return goskema.Decoded[[]byte]{Value: b, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (bytesSchema) TypeCheck(ctx context.Context, v any) error        { return nil }
func (bytesSchema) RuleCheck(ctx context.Context, v any) error        { return nil }
func (bytesSchema) Validate(ctx context.Context, v any) error         { return nil }
func (bytesSchema) ValidateValue(ctx context.Context, v []byte) error { return nil }
func (bytesSchema) JSONSchema() (*js.Schema, error)                   { return &js.Schema{}, nil }

type mapAnySchema struct{}

func (mapAnySchema) Parse(ctx context.Context, v any) (map[string]any, error) {
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	return nil, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected object"}}
}
func (mapAnySchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]any], error) {
	m, err := (mapAnySchema{}).Parse(ctx, v)
	return goskema.Decoded[map[string]any]{Value: m, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (mapAnySchema) TypeCheck(ctx context.Context, v any) error                { return nil }
func (mapAnySchema) RuleCheck(ctx context.Context, v any) error                { return nil }
func (mapAnySchema) Validate(ctx context.Context, v any) error                 { return nil }
func (mapAnySchema) ValidateValue(ctx context.Context, v map[string]any) error { return nil }
func (mapAnySchema) JSONSchema() (*js.Schema, error)                           { return &js.Schema{}, nil }

// ---- Custom codecs for tests ----

type stringToFloatCodec struct{}

func (stringToFloatCodec) In() goskema.Schema[string]   { return g.String() }
func (stringToFloatCodec) Out() goskema.Schema[float64] { return float64Schema{} }
func (stringToFloatCodec) Decode(ctx context.Context, s string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidFormat, Message: err.Error()}}
	}
	if err := (float64Schema{}).ValidateValue(ctx, f); err != nil {
		return 0, err
	}
	return f, nil
}
func (stringToFloatCodec) Encode(ctx context.Context, f float64) (string, error) {
	if err := (float64Schema{}).ValidateValue(ctx, f); err != nil {
		return "", err
	}
	return strconv.FormatFloat(f, 'g', -1, 64), nil
}
func (stringToFloatCodec) DecodeWithMeta(ctx context.Context, s string) (goskema.Decoded[float64], error) {
	f, err := (stringToFloatCodec{}).Decode(ctx, s)
	return goskema.Decoded[float64]{Value: f, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (stringToFloatCodec) EncodePreserving(ctx context.Context, df goskema.Decoded[float64]) (string, error) {
	return (stringToFloatCodec{}).Encode(ctx, df.Value)
}

type stringToBigIntCodec struct{}

func (stringToBigIntCodec) In() goskema.Schema[string]    { return g.String() }
func (stringToBigIntCodec) Out() goskema.Schema[*big.Int] { return bigIntSchema{} }
func (stringToBigIntCodec) Decode(ctx context.Context, s string) (*big.Int, error) {
	b := new(big.Int)
	if _, ok := b.SetString(s, 10); !ok {
		return nil, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidFormat, Message: "invalid bigint"}}
	}
	return b, nil
}
func (stringToBigIntCodec) Encode(ctx context.Context, b *big.Int) (string, error) {
	return b.String(), nil
}
func (stringToBigIntCodec) DecodeWithMeta(ctx context.Context, s string) (goskema.Decoded[*big.Int], error) {
	v, err := (stringToBigIntCodec{}).Decode(ctx, s)
	return goskema.Decoded[*big.Int]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (stringToBigIntCodec) EncodePreserving(ctx context.Context, db goskema.Decoded[*big.Int]) (string, error) {
	return (stringToBigIntCodec{}).Encode(ctx, db.Value)
}

type epochSecondsCodec struct{}

func (epochSecondsCodec) In() goskema.Schema[int64]      { return int64Schema{} }
func (epochSecondsCodec) Out() goskema.Schema[time.Time] { return timeSchemaTest{} }
func (epochSecondsCodec) Decode(ctx context.Context, sec int64) (time.Time, error) {
	return time.Unix(sec, 0).UTC(), nil
}
func (epochSecondsCodec) Encode(ctx context.Context, t time.Time) (int64, error) {
	return t.UTC().Unix(), nil
}
func (epochSecondsCodec) DecodeWithMeta(ctx context.Context, sec int64) (goskema.Decoded[time.Time], error) {
	v, err := (epochSecondsCodec{}).Decode(ctx, sec)
	return goskema.Decoded[time.Time]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (epochSecondsCodec) EncodePreserving(ctx context.Context, dt goskema.Decoded[time.Time]) (int64, error) {
	return (epochSecondsCodec{}).Encode(ctx, dt.Value)
}

type timeSchemaTest struct{}

func (timeSchemaTest) Parse(ctx context.Context, v any) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		return t, nil
	}
	return time.Time{}, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected time.Time"}}
}
func (timeSchemaTest) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[time.Time], error) {
	t, err := (timeSchemaTest{}).Parse(ctx, v)
	return goskema.Decoded[time.Time]{Value: t, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (timeSchemaTest) TypeCheck(ctx context.Context, v any) error           { return nil }
func (timeSchemaTest) RuleCheck(ctx context.Context, v any) error           { return nil }
func (timeSchemaTest) Validate(ctx context.Context, v any) error            { return nil }
func (timeSchemaTest) ValidateValue(ctx context.Context, v time.Time) error { return nil }
func (timeSchemaTest) JSONSchema() (*js.Schema, error)                      { return &js.Schema{}, nil }

type jsonStringToMapCodec struct{}

func (jsonStringToMapCodec) In() goskema.Schema[string]          { return g.String() }
func (jsonStringToMapCodec) Out() goskema.Schema[map[string]any] { return mapAnySchema{} }
func (jsonStringToMapCodec) Decode(ctx context.Context, s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidFormat, Message: err.Error()}}
	}
	return m, nil
}
func (jsonStringToMapCodec) Encode(ctx context.Context, m map[string]any) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: err.Error()}}
	}
	return string(b), nil
}
func (jsonStringToMapCodec) DecodeWithMeta(ctx context.Context, s string) (goskema.Decoded[map[string]any], error) {
	v, err := (jsonStringToMapCodec{}).Decode(ctx, s)
	return goskema.Decoded[map[string]any]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (jsonStringToMapCodec) EncodePreserving(ctx context.Context, dm goskema.Decoded[map[string]any]) (string, error) {
	return (jsonStringToMapCodec{}).Encode(ctx, dm.Value)
}

type utf8StringToBytesCodec struct{}

func (utf8StringToBytesCodec) In() goskema.Schema[string]  { return g.String() }
func (utf8StringToBytesCodec) Out() goskema.Schema[[]byte] { return bytesSchema{} }
func (utf8StringToBytesCodec) Decode(ctx context.Context, s string) ([]byte, error) {
	return []byte(s), nil
}
func (utf8StringToBytesCodec) Encode(ctx context.Context, b []byte) (string, error) {
	return string(b), nil
}
func (utf8StringToBytesCodec) DecodeWithMeta(ctx context.Context, s string) (goskema.Decoded[[]byte], error) {
	v, err := (utf8StringToBytesCodec{}).Decode(ctx, s)
	return goskema.Decoded[[]byte]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (utf8StringToBytesCodec) EncodePreserving(ctx context.Context, db goskema.Decoded[[]byte]) (string, error) {
	return (utf8StringToBytesCodec{}).Encode(ctx, db.Value)
}

type stringBoolCodec struct{ truthy, falsy map[string]struct{} }

func (c stringBoolCodec) In() goskema.Schema[string] { return g.String() }
func (c stringBoolCodec) Out() goskema.Schema[bool]  { return g.Bool() }
func (c stringBoolCodec) Decode(ctx context.Context, s string) (bool, error) {
	if _, ok := c.truthy[lower(s)]; ok {
		return true, nil
	}
	if _, ok := c.falsy[lower(s)]; ok {
		return false, nil
	}
	return false, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidEnum, Message: "invalid stringbool"}}
}
func (c stringBoolCodec) Encode(ctx context.Context, b bool) (string, error) {
	// deterministic: pick first in set
	if b {
		for k := range c.truthy {
			return k, nil
		}
	}
	for k := range c.falsy {
		return k, nil
	}
	return "", goskema.Issues{{Path: "/", Code: goskema.CodeParseError, Message: "no mapping"}}
}
func (c stringBoolCodec) DecodeWithMeta(ctx context.Context, s string) (goskema.Decoded[bool], error) {
	v, err := c.Decode(ctx, s)
	return goskema.Decoded[bool]{Value: v, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (c stringBoolCodec) EncodePreserving(ctx context.Context, db goskema.Decoded[bool]) (string, error) {
	return c.Encode(ctx, db.Value)
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if 'A' <= b[i] && b[i] <= 'Z' {
			b[i] = b[i] + 32
		}
	}
	return string(b)
}

// ---- Tests ----

func TestCodec_TimeRFC3339_NestedInObject_Decode(t *testing.T) {
	_ = context.Background()
	c := codec.TimeRFC3339()

	obj, _ := g.Object().
		Field("startDate", g.SchemaOf[time.Time](g.Codec[string, time.Time](c))).
		Field("title", g.StringOf[string]()).
		Require("startDate", "title").
		UnknownStrict().
		Build()

	jsb := []byte(`{"startDate":"2024-06-01T00:00:00Z","title":"ok"}`)
	v, err := goskema.ParseFrom(context.Background(), obj, goskema.JSONBytes(jsb))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := v["startDate"].(time.Time); !ok {
		t.Fatalf("expected time.Time in output, got: %#v", v["startDate"])
	}
}

func TestCodec_StringToFloat_Roundtrip(t *testing.T) {
	c := stringToFloatCodec{}
	ctx := context.Background()
	v, err := c.Decode(ctx, "3.14")
	if err != nil || v == 0 {
		t.Fatalf("decode err=%v v=%v", err, v)
	}
	s, err := c.Encode(ctx, v)
	if err != nil || s == "" {
		t.Fatalf("encode err=%v s=%q", err, s)
	}
}

func TestCodec_StringToBigInt_Roundtrip(t *testing.T) {
	c := stringToBigIntCodec{}
	ctx := context.Background()
	b, err := c.Decode(ctx, "12345678901234567890")
	if err != nil || b == nil {
		t.Fatalf("decode err=%v b=%v", err, b)
	}
	s, err := c.Encode(ctx, b)
	if err != nil || s != "12345678901234567890" {
		t.Fatalf("encode err=%v s=%q", err, s)
	}
}

func TestCodec_EpochSeconds_Roundtrip(t *testing.T) {
	c := epochSecondsCodec{}
	ctx := context.Background()
	t1, err := c.Decode(ctx, int64(1_700_000_000))
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	sec, err := c.Encode(ctx, t1)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	if sec != 1_700_000_000 {
		t.Fatalf("roundtrip mismatch: %d", sec)
	}
}

func TestCodec_JSONString_Object_Roundtrip(t *testing.T) {
	c := jsonStringToMapCodec{}
	ctx := context.Background()
	m, err := c.Decode(ctx, `{"name":"Alice","age":30}`)
	if err != nil || m["name"] != "Alice" {
		t.Fatalf("decode err=%v m=%v", err, m)
	}
	s, err := c.Encode(ctx, m)
	if err != nil || s == "" {
		t.Fatalf("encode err=%v s=%q", err, s)
	}
}

func TestCodec_UTF8String_Bytes_Roundtrip(t *testing.T) {
	c := utf8StringToBytesCodec{}
	ctx := context.Background()
	b, err := c.Decode(ctx, "hello")
	if err != nil || string(b) != "hello" {
		t.Fatalf("decode err=%v b=%v", err, b)
	}
	s, err := c.Encode(ctx, b)
	if err != nil || s != "hello" {
		t.Fatalf("encode err=%v s=%q", err, s)
	}
}

func TestCodec_StringBool_Truthiness(t *testing.T) {
	c := stringBoolCodec{truthy: map[string]struct{}{"yes": {}}, falsy: map[string]struct{}{"no": {}}}
	ctx := context.Background()
	v, err := c.Decode(ctx, "YES")
	if err != nil || v != true {
		t.Fatalf("decode truthy err=%v v=%v", err, v)
	}
	s, err := c.Encode(ctx, false)
	if err != nil || s == "" {
		t.Fatalf("encode falsy err=%v s=%q", err, s)
	}
}
