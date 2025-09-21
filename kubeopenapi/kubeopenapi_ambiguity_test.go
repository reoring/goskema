package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_AnyOf_Ambiguity_Error(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
					map[string]any{"type": "string"},
				},
			},
		},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{Ambiguity: kubeopenapi.AmbiguityError})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"x":"a"}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err == nil {
		t.Fatalf("expected ambiguous_match error in AmbiguityError mode")
	}
}

func TestImport_AnyOf_FirstMatch_OK(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{
				"anyOf": []any{
					map[string]any{"type": "string"},
					map[string]any{"type": "number"},
				},
			},
		},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{Ambiguity: kubeopenapi.AmbiguityFirstMatch})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"x":"a"}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("expected accept in FirstMatch: %v", err)
	}
}
