package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// This test verifies that EncodePreservingObject drops fields that were materialized
// only by defaults (PresenceDefaultApplied without PresenceSeen/WasNull), while keeping
// explicitly provided fields and nulls.
func TestEncodePreserving_Object_DropsDefaultMaterialized(t *testing.T) {
	ctx := context.Background()
	obj, _ := g.Object().
		Field("name", g.StringOf[string]()).
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("name").
		UnknownStrict().
		Build()

	// WithMeta parse: omit "active" so default applies
	dm, err := goskema.ParseFromWithMeta(ctx, obj, goskema.JSONBytes([]byte(`{"name":"alice"}`)))
	if err != nil {
		t.Fatalf("unexpected parse err: %v", err)
	}

	// Canonical would include active=true. Preserving should drop it.
	out := goskema.EncodePreservingObject(dm)
	if _, ok := out["active"]; ok {
		t.Fatalf("preserving should drop default-materialized field 'active': %#v", out)
	}
	if out["name"] != "alice" {
		t.Fatalf("expected name preserved, got: %#v", out)
	}
}
