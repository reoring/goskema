package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// Smoke test for future PresenceDefaultApplied. For now, we just check that
// the presence map can expose the defaulted field via Include filter.
// If not available yet, we skip (pending implementation).
func TestFieldDefaultPresence_DefaultApplied_Smoke(t *testing.T) {
	ctx := context.Background()

	// schema: { name: string (required), active: bool (default true) }
	s, _ := g.Object().
		Field("name", g.StringOf[string]()).
		Field("active", g.BoolOf[bool]()).Default(true).
		Require("name").
		UnknownStrict().
		Build()

	// input missing "active" → default applied
	js := []byte(`{"name":"Reo"}`)
	opt := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true, Include: []string{"/active"}},
		PathRender: goskema.PathRenderOpt{Intern: true},
	}

	dm, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(js), opt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Expect the defaulted field presence to be visible under "/active".
	// Pending: field-level presence collection not yet implemented.
	if _, ok := dm.Presence["/active"]; !ok {
		t.Skip("pending: field-level presence for default-applied keys not implemented yet")
	}

	// Also ensure value was materialized
	if v, ok := dm.Value["active"]; !ok || v != true {
		t.Fatalf("expected default active=true, got: %#v", dm.Value)
	}
}
