package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Array_Contains_MinMax_Primitive(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"vals": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "number"},
				"contains":    map[string]any{"type": "number"},
				"minContains": 2,
				"maxContains": 3,
			},
		},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	ok := []byte(`{"vals":[1,2,0]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(ok)); err != nil {
		t.Fatalf("expected within min/max contains: %v", err)
	}
	low := []byte(`{"vals":[1]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(low)); err == nil {
		t.Fatalf("expected too_short due to minContains")
	}
	high := []byte(`{"vals":[1,2,3,4]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(high)); err == nil {
		t.Fatalf("expected too_long due to maxContains")
	}
}

func TestImport_Array_Contains_Object_Required(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
				"contains": map[string]any{
					"type":     "object",
					"required": []any{"name"},
				},
				"minContains": 1,
			},
		},
		"additionalProperties": false,
	}

	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}

	ok := []byte(`{"items":[{"name":"a"},{"x":1}]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(ok)); err != nil {
		t.Fatalf("expected contains object with required field: %v", err)
	}

	bad := []byte(`{"items":[{"x":1}]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected too_short due to missing required match")
	}
}
