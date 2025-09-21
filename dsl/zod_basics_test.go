package dsl_test

import (
	"context"
	"testing"
	"time"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
	js "github.com/reoring/goskema/jsonschema"
)

// TestZodBasics_Minimal_Primitives covers minimal schema definitions for
// string, bool, and number.
func TestZodBasics_Minimal_Primitives(t *testing.T) {
	ctx := context.Background()

	// string success and failure cases
	if v, err := g.String().Parse(ctx, "hello"); err != nil || v != "hello" {
		t.Fatalf("string parse ok expected, got v=%v err=%v", v, err)
	}
	if _, err := g.String().Parse(ctx, 1); err == nil {
		t.Fatalf("expected invalid_type for non-string")
	}

	// bool success and failure cases
	if v, err := g.Bool().Parse(ctx, true); err != nil || v != true {
		t.Fatalf("bool parse ok expected, got v=%v err=%v", v, err)
	}
	if _, err := g.Bool().Parse(ctx, "nope"); err == nil {
		t.Fatalf("expected invalid_type for non-bool")
	}

	// number(json.Number) success and failure (float64 allowed, string rejected)
	if _, err := g.NumberJSON().Parse(ctx, 1.23); err != nil {
		t.Fatalf("number parse from float64 expected ok, err=%v", err)
	}
	if _, err := g.NumberJSON().Parse(ctx, "1.0"); err == nil {
		t.Fatalf("expected invalid_type for string input to number")
	}
}

// TestZodBasics_Object_Required_Optional_Default exercises required, optional,
// and default handling on objects.
func TestZodBasics_Object_Required_Optional_Default(t *testing.T) {
	ctx := context.Background()
	user, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Field("nickname", g.StringOf[string]()). // Optional field.
		Field("age", g.BoolOf[bool]()).Default(true).
		Require("id", "name").
		UnknownStrict().
		Build()

	// success: nickname omitted, age receives the default value
	v, err := user.Parse(ctx, map[string]any{"id": "u_1", "name": "Reo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["age"] != true {
		t.Fatalf("expected default age=true, got: %#v", v)
	}

	// failure: missing required field
	if _, err := user.Parse(ctx, map[string]any{"id": "u_1"}); err == nil {
		t.Fatalf("expected required error for missing name")
	}
}

// TestZodBasics_Array_String_MinLen validates string arrays with a minimum
// length of one (element constraints are length-only in MVP).
func TestZodBasics_Array_String_MinLen(t *testing.T) {
	ctx := context.Background()
	tags := g.Array(g.String()).Min(1)

	// ok
	if v, err := tags.Parse(ctx, []any{"dev"}); err != nil || len(v) != 1 || v[0] != "dev" {
		t.Fatalf("array parse expected ok, v=%v err=%v", v, err)
	}
	// failure: empty array
	if _, err := tags.Parse(ctx, []any{}); err == nil {
		t.Fatalf("expected too_short error for empty array")
	}
}

// 8) Correlation check equivalent to superRefine (password === confirm).
// Wrap dsl.Object with a thin layer implementing Refiner to apply custom logic.
type signupSchema struct {
	inner goskema.Schema[map[string]any]
}

func (s signupSchema) Parse(ctx context.Context, v any) (map[string]any, error) {
	out, err := s.inner.Parse(ctx, v)
	if err != nil {
		return nil, err
	}
	if err := goskema.ApplyRefine(ctx, out, s); err != nil {
		return nil, err
	}
	return out, nil
}
func (s signupSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]any], error) {
	m, err := s.Parse(ctx, v)
	return goskema.Decoded[map[string]any]{Value: m, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (s signupSchema) TypeCheck(ctx context.Context, v any) error { return s.inner.TypeCheck(ctx, v) }
func (s signupSchema) RuleCheck(ctx context.Context, v any) error { return s.inner.RuleCheck(ctx, v) }
func (s signupSchema) Validate(ctx context.Context, v any) error  { return s.inner.Validate(ctx, v) }
func (s signupSchema) ValidateValue(ctx context.Context, v map[string]any) error {
	return s.inner.ValidateValue(ctx, v)
}
func (s signupSchema) JSONSchema() (*js.Schema, error) { return s.inner.JSONSchema() }

// Implement Refiner for signupSchema
func (signupSchema) Refine(ctx context.Context, v map[string]any) error {
	pw, _ := v["password"].(string)
	cf, _ := v["confirm"].(string)
	if pw != cf {
		return goskema.Issues{{Path: "/confirm", Code: "custom", Message: "password mismatch"}}
	}
	return nil
}

func TestZodBasics_Refine_PasswordConfirm(t *testing.T) {
	ctx := context.Background()
	base, _ := g.Object().
		Field("email", g.StringOf[string]()).
		Field("password", g.StringOf[string]()).
		Field("confirm", g.StringOf[string]()).
		Require("email", "password", "confirm").
		UnknownStrict().
		Build()

	s := signupSchema{inner: base}

	// failure: mismatch
	if _, err := s.Parse(ctx, map[string]any{"email": "a@b", "password": "x", "confirm": "y"}); err == nil {
		t.Fatalf("expected refine custom error for mismatch")
	}
	// success: match
	if _, err := s.Parse(ctx, map[string]any{"email": "a@b", "password": "x", "confirm": "x"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// TestZodBasics_Codec_TimeRFC3339 demonstrates using the RFC3339 codec
// (time.Time <-> string).
func TestZodBasics_Codec_TimeRFC3339(t *testing.T) {
	c := codec.TimeRFC3339()
	ctx := context.Background()

	// Decode
	t1, err := c.Decode(ctx, "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if !t1.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected time: %v", t1)
	}
	// Encode (canonical, UTC)
	s, err := c.Encode(ctx, t1)
	if err != nil || s == "" {
		t.Fatalf("encode err or empty: %v %q", err, s)
	}
}

// TestZodBasics_TypeGuardLike_IsUser emulates runtime checks and type guards.
func TestZodBasics_TypeGuardLike_IsUser(t *testing.T) {
	ctx := context.Background()
	user, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id", "name").
		UnknownStrict().
		Build()

	isUser := func(v any) bool {
		_, err := user.Parse(ctx, v)
		return err == nil
	}

	if !isUser(map[string]any{"id": "u", "name": "n"}) {
		t.Fatalf("expected isUser==true")
	}
	if isUser(map[string]any{"id": "u"}) {
		t.Fatalf("expected isUser==false for missing required field")
	}
}

// TestZodBasics_Optional_Nullable_Default_Semantics clarifies optional vs
// nullable vs default behaviors.
func TestZodBasics_Optional_Nullable_Default_Semantics(t *testing.T) {
	ctx := context.Background()
	user, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Field("nickname", g.StringOf[string]()). // Optional field.
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("id", "name").
		UnknownStrict().
		Build()

	// optional: missing nickname is accepted
	v, err := user.Parse(ctx, map[string]any{"id": "u1", "name": "Reo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["active"] != true {
		t.Fatalf("expected default active=true, got: %#v", v)
	}

	// nullable: null is not allowed here, so invalid_type (nickname = null)
	_, err = user.Parse(ctx, map[string]any{"id": "u1", "name": "Reo", "nickname": nil})
	if err == nil {
		t.Fatalf("expected invalid_type for nullable without Nullable wrapper")
	}

	// default field provided as null -> invalid_type (defaults apply only to absence)
	_, err = user.Parse(ctx, map[string]any{"id": "u1", "name": "Reo", "active": nil})
	if err == nil {
		t.Fatalf("expected invalid_type when default field provided as null")
	}
}

// TestZodBasics_UnknownPolicy_Basics covers Strict/Strip/Passthrough behaviors.
func TestZodBasics_UnknownPolicy_Basics(t *testing.T) {
	ctx := context.Background()
	// Strict: extra keys raise unknown_key
	sStrict, _ := g.Object().
		Field("name", g.StringOf[string]()).
		UnknownStrict().
		Build()
	if _, err := sStrict.Parse(ctx, map[string]any{"name": "a", "x": 1}); err == nil {
		t.Fatalf("expected unknown_key under Strict")
	}

	// Strip: extra keys are discarded
	sStrip, _ := g.Object().
		Field("name", g.StringOf[string]()).
		UnknownStrip().
		Build()
	v, err := sStrip.Parse(ctx, map[string]any{"name": "a", "x": 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := v["x"]; ok {
		t.Fatalf("expected unknown key stripped, got: %#v", v)
	}

	// Passthrough: extra keys are collected under unknown_target
	sPass, _ := g.Object().
		Field("name", g.StringOf[string]()).
		Field("extra", g.SchemaOf[map[string]any](g.MapAny())).
		UnknownPassthrough("extra").
		Build()
	v2, err := sPass.Parse(ctx, map[string]any{"name": "a", "x": 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	ex, ok := v2["extra"].(map[string]any)
	if !ok || ex["x"] != 1 {
		t.Fatalf("expected passthrough into extra, got: %#v", v2)
	}
}

// TestZodBasics_TypeCheck_vs_RuleCheck visualizes TypeCheck vs RuleCheck phases.
func TestZodBasics_TypeCheck_vs_RuleCheck(t *testing.T) {
	ctx := context.Background()

	// Custom minimal schema: string type required (TypeCheck), length >= 5 (RuleCheck)
	s := min5Schema{}

	// TypeCheck should produce a type error
	if err := s.TypeCheck(ctx, 123); err == nil {
		t.Fatalf("expected invalid_type from TypeCheck")
	}
	// RuleCheck should produce a constraint error
	if err := s.RuleCheck(ctx, "abc"); err == nil {
		t.Fatalf("expected too_short from RuleCheck")
	}
	// Validate runs TypeCheck followed by RuleCheck
	if err := s.Validate(ctx, "abcd"); err == nil {
		t.Fatalf("expected too_short from Validate")
	}
	if err := s.Validate(ctx, "abcde"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// ---- helpers for TypeCheck vs RuleCheck demo ----
type min5Schema struct{}

func (min5Schema) Parse(ctx context.Context, v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected string"}}
	}
	if len(s) < 5 {
		return "", goskema.Issues{{Path: "/", Code: goskema.CodeTooShort, Message: "min length 5"}}
	}
	return s, nil
}
func (min5Schema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[string], error) {
	s, err := (min5Schema{}).Parse(ctx, v)
	return goskema.Decoded[string]{Value: s, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (min5Schema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(string); !ok {
		return goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected string"}}
	}
	return nil
}
func (min5Schema) RuleCheck(ctx context.Context, v any) error {
	s, _ := v.(string)
	if s != "" && len(s) < 5 {
		return goskema.Issues{{Path: "/", Code: goskema.CodeTooShort, Message: "min length 5"}}
	}
	return nil
}
func (min5Schema) Validate(ctx context.Context, v any) error {
	if err := (min5Schema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (min5Schema{}).RuleCheck(ctx, v)
}
func (min5Schema) ValidateValue(ctx context.Context, v string) error {
	if len(v) < 5 {
		return goskema.Issues{{Path: "/", Code: goskema.CodeTooShort, Message: "min length 5"}}
	}
	return nil
}
func (min5Schema) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }
