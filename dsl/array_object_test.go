package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestArraySchema_Parse_MinMax_AndElementValidation(t *testing.T) {
	arr := g.Array[string](g.String()).Min(2).Max(3)

	ctx := context.Background()

	// ok case (len=2)
	got, err := arr.Parse(ctx, []any{"a", "b"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected value: %#v", got)
	}

	// too short (len=1)
	_, err = arr.Parse(ctx, []any{"a"})
	if err == nil {
		t.Fatalf("expected too_short error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeTooShort {
			t.Fatalf("expected too_short, got %v", iss)
		}
	}

	// element invalid type
	_, err = arr.Parse(ctx, []any{"a", 1})
	if err == nil {
		t.Fatalf("expected element invalid_type")
	}
}

func TestArraySchema_InvalidType(t *testing.T) {
	arr := g.Array[string](g.String())
	ctx := context.Background()
	_, err := arr.Parse(ctx, "not array")
	if err == nil {
		t.Fatalf("expected invalid_type for non-array input")
	}
}

func TestObjectSchema_Required_And_UnknownPolicies(t *testing.T) {
	ctx := context.Background()

	// required missing
	objReq, _ := g.Object().
		Field("name", g.StringOf[string]()).
		Require("name").
		UnknownStrict().
		Build()
	_, err := objReq.Parse(ctx, map[string]any{})
	if err == nil {
		t.Fatalf("expected required error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeRequired {
			t.Fatalf("expected required, got %v", iss)
		}
	}

	// UnknownStrict
	objStrict, _ := g.Object().
		Field("name", g.StringOf[string]()).
		UnknownStrict().
		Build()
	_, err = objStrict.Parse(ctx, map[string]any{"name": "a", "x": 1})
	if err == nil {
		t.Fatalf("expected unknown_key error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeUnknownKey {
			t.Fatalf("expected unknown_key, got %v", iss)
		}
	}

	// UnknownStrip
	objStrip, _ := g.Object().
		Field("name", g.StringOf[string]()).
		UnknownStrip().
		Build()
	v, err := objStrip.Parse(ctx, map[string]any{"name": "a", "x": 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := v["x"]; ok {
		t.Fatalf("expected unknown key to be stripped, got: %v", v)
	}

	// UnknownPassthrough with unknownTarget
	objPass, _ := g.Object().
		Field("name", g.StringOf[string]()).
		Field("extra", g.SchemaOf[map[string]any](g.MapAny())).
		UnknownPassthrough("extra").
		Build()
	// Parse (non-WithMeta)
	v2, err := objPass.Parse(ctx, map[string]any{"name": "a", "x": 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	extra, ok := v2["extra"].(map[string]any)
	if !ok || extra["x"] != 1 {
		t.Fatalf("expected passthrough into extra, got: %#v", v2)
	}

	// ParseFromWithMeta should record presence under /extra and /extra/x
	dm, err := goskema.ParseFromWithMeta(ctx, objPass, goskema.JSONBytes([]byte(`{"name":"a","x":1}`)))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, ok := dm.Presence["/extra"]; !ok {
		t.Fatalf("expected presence at /extra, got: %#v", dm.Presence)
	}
	if _, ok := dm.Presence["/extra/x"]; !ok {
		t.Fatalf("expected presence at /extra/x, got: %#v", dm.Presence)
	}
}

func TestObjectSchema_Defaults_Applied_OnMissing(t *testing.T) {
	ctx := context.Background()

	obj, _ := g.Object().
		Field("name", g.StringOf[string]()).Default("anon").
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("name").
		UnknownStrict().
		Build()

	// missing fields should be filled by default and satisfy required
	v, err := obj.Parse(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["name"] != "anon" || v["active"] != true {
		t.Fatalf("defaults not applied: %#v", v)
	}

	// provided value should not be overridden by default
	v2, err := obj.Parse(ctx, map[string]any{"name": "bob"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v2["name"] != "bob" {
		t.Fatalf("expected provided value to win, got: %#v", v2)
	}

	// present null should NOT trigger default; it should be a type error
	_, err = obj.Parse(ctx, map[string]any{"name": nil})
	if err == nil {
		t.Fatalf("expected invalid_type when field present as null")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeInvalidType {
			t.Fatalf("expected invalid_type, got %v", iss)
		}
	}
}
