package dsl_test

import (
	"context"
	"encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestStringSchema_Basic(t *testing.T) {
	s := g.String()
	ctx := context.Background()

	// ok
	v, err := s.Parse(ctx, "hello")
	if err != nil || v != "hello" {
		t.Fatalf("parse ok expected, got v=%v err=%v", v, err)
	}

	// invalid type
	_, err = s.Parse(ctx, 1)
	if err == nil {
		t.Fatalf("expected error for invalid type")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeInvalidType {
			t.Fatalf("expected invalid_type, got %v", iss)
		}
	} else {
		t.Fatalf("expected Issues error, got %v", err)
	}

	// WithMeta presence
	dv, err := s.ParseWithMeta(ctx, "x")
	if err != nil {
		t.Fatalf("parse with meta err: %v", err)
	}
	if dv.Presence["/"]&goskema.PresenceSeen == 0 {
		t.Fatalf("expected PresenceSeen at root")
	}
}

func TestBoolSchema_Basic(t *testing.T) {
	s := g.Bool()
	ctx := context.Background()

	v, err := s.Parse(ctx, true)
	if err != nil || v != true {
		t.Fatalf("parse ok expected, got v=%v err=%v", v, err)
	}
	_, err = s.Parse(ctx, "nope")
	if err == nil {
		t.Fatalf("expected error for invalid type")
	}
}

func TestNumberJSONSchema_Basic(t *testing.T) {
	s := g.NumberJSON()
	ctx := context.Background()

	// json.Number input
	n := json.Number("123.45")
	v, err := s.Parse(ctx, n)
	if err != nil || v != n {
		t.Fatalf("expected roundtrip json.Number, got v=%v err=%v", v, err)
	}

	// float64 input coerced
	v2, err := s.Parse(ctx, float64(1.23))
	if err != nil || string(v2) == "" {
		t.Fatalf("expected formatted number, got v=%v err=%v", v2, err)
	}

	// invalid type (string not allowed by default)
	_, err = s.Parse(ctx, "1.0")
	if err == nil {
		t.Fatalf("expected error for invalid type")
	}
}

func TestNumberJSONSchema_CoerceFromString(t *testing.T) {
	s := g.NumberJSON().CoerceFromString()
	ctx := context.Background()

	v, err := s.Parse(ctx, "1.0")
	if err != nil {
		t.Fatalf("expected coerce from string, err=%v", err)
	}
	if string(v) != "1" && string(v) != "1.0" { // canonicalization via float formatting produces "1"
		t.Fatalf("unexpected coerced value: %q", v)
	}

	// TypeCheck should accept string under coerce mode
	if err := s.TypeCheck(ctx, "2.5"); err != nil {
		t.Fatalf("TypeCheck should accept string when coerce enabled: %v", err)
	}
}
