package dsl_test

import (
	"encoding/json"
	"reflect"
	"testing"

	g "github.com/reoring/goskema/dsl"
)

// normalize marshals v to JSON and unmarshals back into interface{} to remove ordering effects.
func normalize(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

func TestJSONSchema_Primitives(t *testing.T) {
	// string
	if s, err := g.String().JSONSchema(); err != nil {
		t.Fatalf("string JSONSchema err: %v", err)
	} else {
		got := normalize(s)
		want := normalize(map[string]any{"type": "string"})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("string schema mismatch\n got=%v\nwant=%v", got, want)
		}
	}

	// boolean
	if s, err := g.Bool().JSONSchema(); err != nil {
		t.Fatalf("bool JSONSchema err: %v", err)
	} else {
		got := normalize(s)
		want := normalize(map[string]any{"type": "boolean"})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("bool schema mismatch\n got=%v\nwant=%v", got, want)
		}
	}

	// number (json.Number)
	if s, err := g.NumberJSON().JSONSchema(); err != nil {
		t.Fatalf("number JSONSchema err: %v", err)
	} else {
		got := normalize(s)
		want := normalize(map[string]any{"type": "number"})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("number schema mismatch\n got=%v\nwant=%v", got, want)
		}
	}
}

func TestJSONSchema_Array(t *testing.T) {
	arr := g.Array[string](g.String()).Min(1).Max(2)
	s, err := arr.JSONSchema()
	if err != nil {
		t.Fatalf("array JSONSchema err: %v", err)
	}

	got := normalize(s)
	want := normalize(map[string]any{
		"type":     "array",
		"items":    map[string]any{"type": "string"},
		"minItems": 1,
		"maxItems": 2,
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("array schema mismatch\n got=%v\nwant=%v", got, want)
	}
}

func TestJSONSchema_Object_UnknownPolicies(t *testing.T) {
	// Strict → additionalProperties=false
	objStrict, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id").
		UnknownStrict().
		Build()
	s1, err := objStrict.JSONSchema()
	if err != nil {
		t.Fatalf("object(strict) JSONSchema err: %v", err)
	}
	got1 := normalize(s1)
	want1 := normalize(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":   map[string]any{"type": "string"},
			"name": map[string]any{"type": "string"},
		},
		"required":             []any{"id"},
		"additionalProperties": false,
	})
	if !reflect.DeepEqual(got1, want1) {
		t.Fatalf("object(strict) schema mismatch\n got=%v\nwant=%v", got1, want1)
	}

	// Strip → additionalProperties=true (accepts then discards extra fields)
	objStrip, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		Build()
	sStrip, err := objStrip.JSONSchema()
	if err != nil {
		t.Fatalf("object(strip) JSONSchema err: %v", err)
	}
	gotStrip := normalize(sStrip)
	wantStrip := normalize(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":   map[string]any{"type": "string"},
			"name": map[string]any{"type": "string"},
		},
		"required":             []any{"id"},
		"additionalProperties": true,
	})
	if !reflect.DeepEqual(gotStrip, wantStrip) {
		t.Fatalf("object(strip) schema mismatch\n got=%v\nwant=%v", gotStrip, wantStrip)
	}

	// Passthrough → additionalProperties=true
	objPass, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Field("extra", g.SchemaOf[map[string]any](g.MapAny())).
		Require("id").
		UnknownPassthrough("extra").
		Build()
	s2, err := objPass.JSONSchema()
	if err != nil {
		t.Fatalf("object(passthrough) JSONSchema err: %v", err)
	}
	got2 := normalize(s2)
	want2 := normalize(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":    map[string]any{"type": "string"},
			"name":  map[string]any{"type": "string"},
			"extra": map[string]any{"type": "object", "additionalProperties": true},
		},
		"required":             []any{"id"},
		"additionalProperties": true,
	})
	if !reflect.DeepEqual(got2, want2) {
		t.Fatalf("object(passthrough) schema mismatch\n got=%v\nwant=%v", got2, want2)
	}
}

func TestJSONSchema_Object_Defaults(t *testing.T) {
	// Default via fieldStep.Default(v)
	obj1, _ := g.Object().
		Field("active", g.BoolOf[bool]()).
		Field("name", g.StringOf[string]()).
		Require("name").
		UnknownStrict().
		// set default for active
		Field("active", g.BoolOf[bool]()).Default(true).
		Build()
	s1, err := obj1.JSONSchema()
	if err != nil {
		t.Fatalf("object(default) JSONSchema err: %v", err)
	}
	got1 := normalize(s1)
	wantActive := map[string]any{"type": "boolean", "default": true}
	props1, _ := got1.(map[string]any)["properties"].(map[string]any)
	if props1 == nil || !reflect.DeepEqual(props1["active"], normalize(wantActive)) {
		t.Fatalf("default(active) not projected\n got=%v", got1)
	}

	// Default applied via fieldStep.Default helper
	obj2, _ := g.Object().
		Field("active", g.BoolOf[bool]()).Default(true).
		UnknownStrict().
		Build()
	s2, err := obj2.JSONSchema()
	if err != nil {
		t.Fatalf("object(adaptWithDefault) JSONSchema err: %v", err)
	}
	got2 := normalize(s2)
	props2, _ := got2.(map[string]any)["properties"].(map[string]any)
	if props2 == nil || !reflect.DeepEqual(props2["active"], normalize(wantActive)) {
		t.Fatalf("default(active) via Of+Default not projected\n got=%v", got2)
	}
}
