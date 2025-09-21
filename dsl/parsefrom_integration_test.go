package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestParseFrom_FailFast_StopsAtFirstIssue_UnknownKey(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrict().
		Build()

	js := []byte(`{"id": 1, "zzz": true}`)
	// FailFast: true
	_, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js), goskema.ParseOpt{FailFast: true})
	iss, ok := goskema.AsIssues(err)
	if !ok || len(iss) == 0 {
		t.Fatalf("expected issues, got %v", err)
	}
}

func TestParseFrom_Collect_GathersMultipleIssues(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id", "name").
		UnknownStrict().
		Build()

	js := []byte(`{"zzz": true}`)
	// FailFast: false (collect)
	_, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js), goskema.ParseOpt{})
	iss, ok := goskema.AsIssues(err)
	if !ok || len(iss) < 2 { // at least: unknown_key for zzz, required for id/name
		t.Fatalf("expected multiple issues, got %v", err)
	}
}

// Below: existing tests in this file

func TestParseFrom_DSL_Object_JSONBytes_Strict(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrict().
		Build()

	// ok
	js := []byte(`{"id":"u_1"}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["id"] != "u_1" {
		t.Fatalf("unexpected value: %#v", v)
	}

	// unknown key error
	js2 := []byte(`{"id":"u_1","x":1}`)
	_, err = goskema.ParseFrom(ctx, s, goskema.JSONBytes(js2))
	if err == nil {
		t.Fatalf("expected unknown_key error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeUnknownKey {
			t.Fatalf("expected unknown_key, got %v", iss)
		}
	}
}

func TestParseFrom_DSL_Object_Defaults_Applied(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("name", g.StringOf[string]()).Default("anon").
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("name").
		UnknownStrict().
		Build()

	js := []byte(`{}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["name"] != "anon" || v["active"] != true {
		t.Fatalf("defaults not applied: %#v", v)
	}
}
