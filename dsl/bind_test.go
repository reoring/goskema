package dsl_test

import (
	"context"
	"testing"

	g "github.com/reoring/goskema/dsl"
)

type userBind struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `goskema:"name=nickname"`
}

func TestBind_Basic_KeyResolution(t *testing.T) {
	ctx := context.Background()
	b := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Field("nickname", g.StringOf[string]()).
		Require("id", "name").
		UnknownStrict()

	s, err := g.Bind[userBind](b)
	if err != nil {
		t.Fatalf("bind err: %v", err)
	}

	m := map[string]any{"id": "u1", "name": "Reo", "nickname": "R"}
	v, err := s.Parse(ctx, m)
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if v.ID != "u1" || v.Name != "Reo" || v.Alias != "R" {
		t.Fatalf("unexpected value: %+v", v)
	}
}

func TestBind_ValidateValue_TypedZeroValuesTreatedAsPresent(t *testing.T) {
	ctx := context.Background()
	b := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id").
		UnknownStrict()

	s, err := g.Bind[userBind](b)
	if err != nil {
		t.Fatalf("bind err: %v", err)
	}
	// Typed zero values are treated as present, so the required check does not fire.
	err = s.ValidateValue(ctx, userBind{Name: "n"})
	if err != nil {
		t.Fatalf("unexpected error for zero-valued required field: %v", err)
	}
}
