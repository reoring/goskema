package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

// Ensure that nested objects beyond one level are imported recursively
// and unknown keys are rejected when additionalProperties=false at each level.
func TestImport_NestedObjects_Recursive_ThreeLevels_Accept(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"level1": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"level2": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name": map[string]any{"type": "string"},
								},
								"additionalProperties": false,
							},
						},
						"additionalProperties": false,
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []any{"spec"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// exact match should pass
	js := []byte(`{"spec":{"level1":{"level2":{"name":"ok"}}}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("expected accept nested object: %v", err)
	}
}

func TestImport_NestedObjects_Recursive_ThreeLevels_UnknownRejected(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"spec": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"level1": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"level2": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name": map[string]any{"type": "string"},
								},
								"additionalProperties": false,
							},
						},
						"additionalProperties": false,
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []any{"spec"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// unknown at level2 should be rejected (additionalProperties=false)
	bad := []byte(`{"spec":{"level1":{"level2":{"name":"ok","extra":1}}}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected unknown key rejection at level2")
	}
}
