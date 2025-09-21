package goskema_test

import (
	"bytes"
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// noop schema for tests
type noopSchema struct{}

func (noopSchema) Parse(ctx context.Context, v any) (struct{}, error) { return struct{}{}, nil }
func (noopSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[struct{}], error) {
	return goskema.Decoded[struct{}]{Value: struct{}{}, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, nil
}
func (noopSchema) TypeCheck(ctx context.Context, v any) error          { return nil }
func (noopSchema) RuleCheck(ctx context.Context, v any) error          { return nil }
func (noopSchema) Validate(ctx context.Context, v any) error           { return nil }
func (noopSchema) ValidateValue(ctx context.Context, v struct{}) error { return nil }
func (noopSchema) JSONSchema() (*js.Schema, error)                     { return &js.Schema{}, nil }

func TestStreamParse_DuplicateKey_Error(t *testing.T) {
	jsb := []byte(`{"a":1,"a":2}`)
	opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}
	_, err := goskema.StreamParse(context.Background(), noopSchema{}, bytes.NewReader(jsb), opt)
	if err == nil {
		t.Fatalf("expected error for duplicate key")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeDuplicateKey {
			t.Fatalf("expected duplicate_key issue, got: %v", iss)
		} else if iss[0].Path != "/a" {
			t.Fatalf("expected path=/a, got: %s", iss[0].Path)
		}
	} else {
		t.Fatalf("expected Issues error, got: %v", err)
	}
}

func TestStreamParse_DuplicateKey_NestedPath(t *testing.T) {
	jsb := []byte(`[{"a":1,"a":2}]`)
	opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}
	_, err := goskema.StreamParse(context.Background(), noopSchema{}, bytes.NewReader(jsb), opt)
	if err == nil {
		t.Fatalf("expected error for duplicate key")
	}
	iss, ok := goskema.AsIssues(err)
	if !ok || len(iss) == 0 {
		t.Fatalf("expected Issues, got: %v", err)
	}
	if iss[0].Path != "/0/a" {
		t.Fatalf("expected path=/0/a, got: %s", iss[0].Path)
	}
}

func TestStreamParse_MaxDepth_Exceeded(t *testing.T) {
	// depth = 3 for { a: { b: { c: 1 } } }
	jsb := []byte(`{"a":{"b":{"c":1}}}`)
	opt := goskema.ParseOpt{MaxDepth: 2}
	_, err := goskema.StreamParse(context.Background(), noopSchema{}, bytes.NewReader(jsb), opt)
	if err == nil {
		t.Fatalf("expected error for max depth exceeded")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Path != "/a/b" {
			t.Fatalf("expected path=/a/b for max depth, got: %v", iss)
		}
	}
}

func TestStreamParse_MaxBytes_Exceeded(t *testing.T) {
	// Provide N bytes > MaxBytes and any valid JSON prefix to ensure read path
	data := append([]byte("{}"), bytes.Repeat([]byte("x"), 1024)...)
	r := bytes.NewReader(data)
	opt := goskema.ParseOpt{MaxBytes: 2} // smaller than data
	_, err := goskema.StreamParse(context.Background(), noopSchema{}, r, opt)
	if err == nil {
		t.Fatalf("expected error for max bytes exceeded")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeTruncated {
			t.Fatalf("expected truncated issue, got: %v", iss)
		}
		if iss[0].Path != "" && iss[0].Path != "/" {
			t.Fatalf("expected truncated path empty or root, got: %s", iss[0].Path)
		}
	}
}
