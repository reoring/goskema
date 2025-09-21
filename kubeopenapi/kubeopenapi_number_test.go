package kubeopenapi_test

import (
	"context"
	"encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Minimal_Number_Field_AsJSONNumber(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"price": map[string]any{"type": "number"},
		},
		"required":             []any{"price"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"price": 12.5}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if _, ok := v["price"].(json.Number); !ok {
		t.Fatalf("price should be json.Number, got %T (%#v)", v["price"], v["price"])
	}
}

func TestImport_Minimal_Integer_Field_AsJSONNumber(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer"},
		},
		"required":             []any{"count"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"count": 10}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if got, ok := v["count"].(json.Number); !ok {
		t.Fatalf("count should be json.Number, got %T (%#v)", v["count"], v["count"])
	} else if string(got) != "10" {
		t.Fatalf("count json.Number should be '10', got %q", string(got))
	}
}
