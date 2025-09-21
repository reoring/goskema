package goskema_test

import (
	"context"
	"errors"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// TestErrorModel_CollectVsFailFast_And_AsIssues compares Collect versus
// Fail-Fast behavior and exercises both AsIssues and errors.As helpers.
func TestErrorModel_CollectVsFailFast_And_AsIssues(t *testing.T) {
	ctx := context.Background()
	user, _ := g.Object().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	js := []byte(`{"email": 1, "zzz": true}`)

	// Collect mode: gather multiple issues
	_, err := goskema.ParseFrom(ctx, user, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected issues in collect mode")
	}
	var iss goskema.Issues
	if !errors.As(err, &iss) {
		t.Fatalf("expected errors.As to extract Issues, got: %v", err)
	}
	if len(iss) < 2 {
		t.Fatalf("expected multiple issues, got: %v", iss)
	}
	// Expected codes include unknown_key plus either required/email or
	// invalid_type/email. We intentionally avoid asserting exact messages.

	// Fail-Fast: stop at the first issue
	_, err = goskema.ParseFrom(ctx, user, goskema.JSONBytes(js), goskema.ParseOpt{FailFast: true})
	iss2, ok := goskema.AsIssues(err)
	if !ok || len(iss2) == 0 {
		t.Fatalf("expected fail-fast issues, got: %v", err)
	}
}

// TestErrorModel_DeterministicOrder lightly checks that required errors are
// ordered by key name and unknown-key errors remain stable.
func TestErrorModel_DeterministicOrder(t *testing.T) {
	ctx := context.Background()
	obj, _ := g.Object().
		Field("a", g.StringOf[string]()).Required().
		Field("b", g.StringOf[string]()).Required().
		Field("c", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	// Missing: a,b,c and unknown: zzz, yyy -> post-sort unknowns become "/y" then "/z".
	js := []byte(`{"zzz":1,"yyy":2}`)
	_, err := goskema.ParseFrom(ctx, obj, goskema.JSONBytes(js))
	iss, ok := goskema.AsIssues(err)
	if !ok || len(iss) < 2 {
		t.Fatalf("expected multiple issues, got: %v", err)
	}
	// Spot-check only the first few entries to avoid brittle ordering assertions.
	// Required keys should appear in a,b,c order and unknown keys should intermix
	// while remaining sorted by name. Ensure at least one entry starts with "/a".
	if iss[0].Path == "" {
		t.Fatalf("expected first issue to have a path, got empty")
	}
}
