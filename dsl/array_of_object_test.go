package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// TestArrayOfObject_NoConstraint covers ArrayOf[Object] without additional constraints.
func TestArrayOfObject_NoConstraint(t *testing.T) {
	ctx := context.Background()

	item, _ := g.Object().
		Field("id", g.StringOf[string]()).Required().
		Field("name", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	s, _ := g.Object().
		Field("items", g.ArrayOf[map[string]any](item)).
		UnknownStrict().
		Build()

	v, err := s.Parse(ctx, map[string]any{
		"items": []any{
			map[string]any{"id": "1", "name": "A"},
			map[string]any{"id": "2", "name": "B"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected err(ArrayOf): %v", err)
	}
	if got := len(v["items"].([]map[string]any)); got != 2 {
		t.Fatalf("items len want=2 got=%d", got)
	}
}

// TestArrayOfObject_MinConstraint covers ArrayOfSchema with a Min=1 constraint.
func TestArrayOfObject_MinConstraint(t *testing.T) {
	ctx := context.Background()

	item, _ := g.Object().
		Field("id", g.StringOf[string]()).Required().
		Field("name", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	ab := g.Array(item).Min(1)
	s, _ := g.Object().
		Field("items", g.ArrayOfSchema(ab)).
		UnknownStrict().
		Build()

	// ok
	if _, err := s.Parse(ctx, map[string]any{
		"items": []any{map[string]any{"id": "1", "name": "A"}},
	}); err != nil {
		t.Fatalf("unexpected err(ArrayOfSchema ok): %v", err)
	}
	// failure: too_short
	if _, err := s.Parse(ctx, map[string]any{"items": []any{}}); err == nil {
		t.Fatalf("expected too_short for empty items")
	} else if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeTooShort {
			t.Fatalf("want too_short, got: %v", iss)
		}
	}
}

// TestArrayOfObject_DiscriminatedUnion demonstrates array elements as a
// discriminated union (card/bank).
func TestArrayOfObject_DiscriminatedUnion(t *testing.T) {
	ctx := context.Background()

	card, _ := g.Object().
		Field("type", g.StringOf[string]()).
		Field("number", g.StringOf[string]()).
		Require("number").
		UnknownStrict().
		Build()

	bank, _ := g.Object().
		Field("type", g.StringOf[string]()).
		Field("iban", g.StringOf[string]()).
		Require("iban").
		UnknownStrict().
		Build()

	u := g.Object().
		Discriminator("type").
		OneOf(
			g.Variant("card", card),
			g.Variant("bank", bank),
		).
		MustBuild()

	ab := g.Array(u).Min(1)
	s, _ := g.Object().
		Field("payments", g.ArrayOfSchema[map[string]any](ab)).
		UnknownStrict().
		Build()

	// success: mix of card and bank entries
	_, err := s.Parse(ctx, map[string]any{
		"payments": []any{
			map[string]any{"type": "card", "number": "4111111111111111"},
			map[string]any{"type": "bank", "iban": "DE89370400440532013000"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// failure: discriminator_missing
	if _, err := s.Parse(ctx, map[string]any{
		"payments": []any{
			map[string]any{"number": "4111111111111111"},
		},
	}); err == nil {
		t.Fatalf("expected discriminator_missing")
	} else if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeDiscriminatorMissing {
			t.Fatalf("want discriminator_missing, got: %v", iss)
		}
	}
}
