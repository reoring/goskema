package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_EmbeddedResource_MinimalPresence_Object(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"res": map[string]any{
				"type":                           "object",
				"x-kubernetes-embedded-resource": true,
			},
		},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{EnableEmbeddedChecks: true})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{"res":{"apiVersion":"v1","kind":"Pod","metadata":{}}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected minimal embedded resource to pass: %v", err)
	}
	bad := []byte(`{"res":{"kind":"Pod","metadata":{}}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required error for missing apiVersion")
	}
}

func TestImport_EmbeddedResource_MinimalPresence_Array(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                           "object",
					"x-kubernetes-embedded-resource": true,
				},
			},
		},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{EnableEmbeddedChecks: true})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{"items":[{"apiVersion":"v1","kind":"Pod","metadata":{}}]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected minimal embedded resource array to pass: %v", err)
	}
	bad := []byte(`{"items":[{"kind":"Pod","metadata":{}}]}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected required error for missing apiVersion in array element")
	}
}
