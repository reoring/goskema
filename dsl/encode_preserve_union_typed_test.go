package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestPreserve_TypedBind_Object_DefaultDropped(t *testing.T) {
	type User struct {
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}
	ctx := context.Background()
	b := g.Object().
		Field("name", g.StringOf[string]()).
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("name").
		UnknownStrict()
	s, err := g.Bind[User](b)
	if err != nil {
		t.Fatalf("bind err: %v", err)
	}

	dm, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes([]byte(`{"name":"alice"}`)))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	// Project back to wire map and apply preserving helper
	inner := goskema.Decoded[map[string]any]{Value: map[string]any{"name": dm.Value.Name, "active": dm.Value.Active}, Presence: dm.Presence}
	out := goskema.EncodePreservingObject(inner)
	if _, ok := out["active"]; ok {
		t.Fatalf("typed preserving should drop default-materialized 'active': %#v", out)
	}
}

func TestPreserve_Union_ParseWithMeta_CarriesPresence(t *testing.T) {
	ctx := context.Background()
	card, _ := g.Object().
		Field("type", g.StringOf[string]()).
		Field("number", g.StringOf[string]()).
		Require("type", "number").
		UnknownStrict().
		Build()
	u := g.Object().
		Discriminator("type").
		OneOf(g.Variant("card", card)).
		MustBuild()

	dm, err := goskema.ParseFromWithMeta(ctx, u, goskema.JSONBytes([]byte(`{"type":"card","number":"n"}`)))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if _, ok := dm.Presence["/number"]; !ok {
		t.Fatalf("expected presence for variant fields to be retained, got: %#v", dm.Presence)
	}
}
