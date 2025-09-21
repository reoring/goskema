package kubeopenapi_test

import (
	"context"
	stdjson "encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Array_Items_SingleSchema_String(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
		"required":             []any{"tags"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"tags":["a","b"]}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	switch tt := v["tags"].(type) {
	case []string:
		if len(tt) != 2 || tt[0] != "a" || tt[1] != "b" {
			t.Fatalf("unexpected tags ([]string): %#v", tt)
		}
	case []any:
		if len(tt) != 2 || tt[0] != "a" || tt[1] != "b" {
			t.Fatalf("unexpected tags ([]any): %#v", tt)
		}
	default:
		t.Fatalf("unexpected tags type: %T (%#v)", v["tags"], v["tags"])
	}
}

func TestImport_IntOrString_Field_CoercesStringNumber(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"replicas": map[string]any{
				"x-kubernetes-int-or-string": true,
			},
		},
		"required":             []any{"replicas"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"replicas":"2"}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if _, ok := v["replicas"].(string); ok {
		t.Fatalf("expected numeric-like value projected, got string: %#v", v["replicas"])
	}
}

func TestImport_IntOrString_ArrayItems_CoercesStringNumber(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"vals": map[string]any{
				"type": "array",
				"items": map[string]any{
					"x-kubernetes-int-or-string": true,
				},
			},
		},
		"required":             []any{"vals"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"vals":["1",2,"3"]}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if arr, ok := v["vals"].([]stdjson.Number); ok {
		if len(arr) != 3 || arr[0] == "" || arr[1] == "" || arr[2] == "" {
			t.Fatalf("unexpected json.Number array: %#v", arr)
		}
		return
	}
	if arr, ok := v["vals"].([]any); ok {
		for i, it := range arr {
			if _, isStr := it.(string); isStr {
				t.Fatalf("item %d remained string, expected numeric-like projection: %#v", i, it)
			}
		}
		return
	}
	t.Fatalf("unexpected vals type: %T (%#v)", v["vals"], v["vals"])
}

func TestImport_ListType_Set_Duplicate_Detected(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{
				"type":                   "array",
				"items":                  map[string]any{"type": "string"},
				"x-kubernetes-list-type": "set",
			},
		},
		"required":             []any{"tags"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"tags":["a","a"]}`)
	_, err = goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected duplicate error for set")
	}
}

func TestImport_ListType_Map_Duplicate_ByKeys(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"selectors": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}, "ns": map[string]any{"type": "string"}},
				},
				"x-kubernetes-list-type":     "map",
				"x-kubernetes-list-map-keys": []any{"name", "ns"},
			},
		},
		"required":             []any{"selectors"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"selectors":[{"name":"a","ns":"n1"},{"name":"a","ns":"n1"}]}`)
	_, err = goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected duplicate error for list-map-keys")
	}
}
