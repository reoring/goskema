package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestParseFrom_Array_FailFast_StopsAtFirstIssue_ElementError(t *testing.T) {
	ctx := context.Background()
	s := g.Array[string](g.String())

	js := []byte(`[1,"ok",2]`)
	// FailFast: true -> stop at the first element (index 0)
	_, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js), goskema.ParseOpt{FailFast: true})
	if err == nil {
		t.Fatalf("expected error for first element parse failure")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 {
			t.Fatalf("expected at least one issue, got none")
		}
		if iss[0].Path != "/0" {
			t.Fatalf("expected first error path /0, got %s", iss[0].Path)
		}
		// element parse error is surfaced as parse_error at index
		if iss[0].Code != goskema.CodeParseError {
			t.Fatalf("expected parse_error for element failure, got %s", iss[0].Code)
		}
	} else {
		t.Fatalf("expected Issues error, got: %v", err)
	}
}

func TestParseFrom_Array_Collect_GathersMultipleElementErrors(t *testing.T) {
	ctx := context.Background()
	s := g.Array[string](g.String())

	js := []byte(`[1,"ok",2]`)
	// Collect (FailFast=false) -> gather errors across multiple elements
	_, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected issues for invalid elements")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) < 2 {
			t.Fatalf("expected at least two issues, got %v", iss)
		}
		// Expect issues at /0 and /2 (preserving input order)
		if iss[0].Path != "/0" || iss[0].Code != goskema.CodeParseError {
			t.Fatalf("expected first issue at /0 parse_error, got %v", iss[0])
		}
		if iss[1].Path != "/2" || iss[1].Code != goskema.CodeParseError {
			t.Fatalf("expected second issue at /2 parse_error, got %v", iss[1])
		}
	} else {
		t.Fatalf("expected Issues error, got: %v", err)
	}
}

func TestParseFromWithMeta_Array_Presence_WasNullAndSeen(t *testing.T) {
	ctx := context.Background()
	s := g.Array[string](g.String())

	js := []byte(`[null,"x"]`)
	dm, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected error due to null element for string schema")
	}
	// Presence metadata is returned even on error
	pm := dm.Presence
	if pm == nil {
		t.Fatalf("expected presence map, got nil")
	}
	if pm["/"]&goskema.PresenceSeen == 0 {
		t.Fatalf("root presence not marked seen")
	}
	if pm["/0"]&goskema.PresenceSeen == 0 {
		t.Fatalf("index 0 not marked seen")
	}
	if pm["/0"]&goskema.PresenceWasNull == 0 {
		t.Fatalf("index 0 not marked wasNull")
	}
	if pm["/1"]&goskema.PresenceSeen == 0 {
		t.Fatalf("index 1 not marked seen")
	}
}
