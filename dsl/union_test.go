package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

const testCardNumber = "4111111111111111"

func TestUnion_Discriminator_HappyPath(t *testing.T) {
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

	// card
	v, err := u.Parse(ctx, map[string]any{"type": "card", "number": testCardNumber})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v["number"] != testCardNumber {
		t.Fatalf("unexpected value: %#v", v)
	}

	// bank
	v2, err := u.Parse(ctx, map[string]any{"type": "bank", "iban": "DE89 3704 0044 0532 0130 00"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v2["iban"] == nil {
		t.Fatalf("iban missing: %#v", v2)
	}
}

func TestUnion_Discriminator_Missing(t *testing.T) {
	ctx := context.Background()

	card, _ := g.Object().
		Field("type", g.StringOf[string]()).
		Field("number", g.StringOf[string]()).
		Require("number").
		UnknownStrict().
		Build()

	u := g.Object().
		Discriminator("type").
		OneOf(g.Variant("card", card)).
		MustBuild()

	_, err := u.Parse(ctx, map[string]any{"number": "x"})
	if err == nil {
		t.Fatalf("expected discriminator_missing")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeDiscriminatorMissing {
			t.Fatalf("expected discriminator_missing, got: %v", iss)
		}
	}
}

func TestUnion_Discriminator_Unknown(t *testing.T) {
	ctx := context.Background()

	card, _ := g.Object().
		Field("type", g.StringOf[string]()).
		Field("number", g.StringOf[string]()).
		Require("number").
		UnknownStrict().
		Build()

	u := g.Object().
		Discriminator("type").
		OneOf(g.Variant("card", card)).
		MustBuild()

	_, err := u.Parse(ctx, map[string]any{"type": "legacy"})
	if err == nil {
		t.Fatalf("expected discriminator_unknown")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeDiscriminatorUnknown {
			t.Fatalf("expected discriminator_unknown, got: %v", iss)
		}
	}
}

func TestUnion_JSONSchema_OneOf(t *testing.T) {
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

	js, err := u.JSONSchema()
	if err != nil {
		t.Fatalf("jsonschema err: %v", err)
	}
	if js.OneOf == nil || len(js.OneOf) != 2 {
		t.Fatalf("expected oneOf with 2 variants, got: %#v", js)
	}
}
