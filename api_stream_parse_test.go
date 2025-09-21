package goskema_test

import (
	"context"
	"strings"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestStreamParse_Array_Typed_Success(t *testing.T) {
	ctx := context.Background()
	type Item struct {
		ID string `json:"id"`
	}
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		MustBind()

	arr := g.Array[Item](item)
	r := strings.NewReader(`[{"id":"a"},{"id":"b"}]`)
	vals, err := goskema.StreamParse(ctx, arr, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("want 2, got %d", len(vals))
	}
}

func TestStreamParse_Array_Typed_CollectErrors(t *testing.T) {
	ctx := context.Background()
	type Item struct {
		ID string `json:"id"`
	}
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		MustBind()

	arr := g.Array[Item](item)
	r := strings.NewReader(`[{"id":"ok"},{"id":1},{"id":"ok2"}]`)
	_, err := goskema.StreamParse(ctx, arr, r)
	if err == nil {
		t.Fatalf("expected issues, got nil")
	}
	iss, ok := goskema.AsIssues(err)
	if !ok || len(iss) == 0 {
		t.Fatalf("expected Issues, got: %v", err)
	}
	// first error should reference index 1 (allow nested field path)
	if !(iss[0].Path == "/1" || iss[0].Path == "/1/id") {
		t.Fatalf("want path=/1 or /1/id, got %s", iss[0].Path)
	}
}

func TestStreamParse_MaxBytes_Truncated(t *testing.T) {
	ctx := context.Background()
	type Item struct {
		ID string `json:"id"`
	}
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		MustBind()

	arr := g.Array[Item](item)
	// input ~16 bytes; set cap to 8 to trigger truncated
	r := strings.NewReader(`[{"id":"x"}]`)
	_, err := goskema.StreamParse(ctx, arr, r, goskema.ParseOpt{MaxBytes: 8})
	if err == nil {
		t.Fatalf("expected truncated error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		if len(iss) == 0 || iss[0].Code != goskema.CodeTruncated {
			t.Fatalf("want truncated, got: %v", iss)
		}
	} else {
		t.Fatalf("expected Issues, got: %v", err)
	}
}
