package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Refs_LocalDefs_InProperties(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"$defs": map[string]any{
			"Name": map[string]any{"type": "string"},
			"LabelMap": map[string]any{
				"type":                 "object",
				"additionalProperties": map[string]any{"type": "string"},
			},
		},
		"properties": map[string]any{
			"name":   map[string]any{"$ref": "#/$defs/Name"},
			"labels": map[string]any{"$ref": "#/$defs/LabelMap"},
		},
		"required":             []any{"name", "labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"name":"ok","labels":{"a":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("parse err (should resolve $defs): %v", err)
	}
}

func TestImport_Refs_LocalDefs_InItems(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"$defs": map[string]any{
			"Tag": map[string]any{"type": "string"},
		},
		"properties": map[string]any{
			"tags": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/$defs/Tag"},
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
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("parse err (should resolve $defs in items): %v", err)
	}
}

func TestImport_PatternProperties_SingleRegex_StringValues(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				// keys must start with 'app-'
				"patternProperties": map[string]any{
					"^app-": map[string]any{"type": "string"},
				},
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// accept
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"app-a":"x"}}`))); err != nil {
		t.Fatalf("expected accept: %v", err)
	}
	// reject: key not matching regex
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"bad":"x"}}`))); err == nil {
		t.Fatalf("expected key pattern violation for 'bad'")
	}
}

func TestImport_PatternProperties_MultipleRegex_StringValues(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				// allow keys starting with 'app-' or 'sys-'
				"patternProperties": map[string]any{
					"^app-": map[string]any{"type": "string"},
					"^sys-": map[string]any{"type": "string"},
				},
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// accept both prefixes
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"app-a":"x","sys-b":"y"}}`))); err != nil {
		t.Fatalf("expected accept both prefixes: %v", err)
	}
	// reject other prefix
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"bad":"x"}}`))); err == nil {
		t.Fatalf("expected key pattern violation for 'bad'")
	}
}

func TestImport_PatternProperties_WithAdditionalPropertiesTrue_AllowsUnmatchedKeys(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				"patternProperties": map[string]any{
					"^app-": map[string]any{"type": "string"},
				},
				// unmatched keys should be allowed when additionalProperties is true
				"additionalProperties": true,
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// both matching and unmatched keys should be accepted
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"app-a":"x","other":"y"}}`))); err != nil {
		t.Fatalf("expected unmatched key allowed due to additionalProperties=true: %v", err)
	}
}

func TestImport_PatternProperties_WithAdditionalPropertiesSchema_TypeMismatch(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				"patternProperties": map[string]any{
					"^app-": map[string]any{"type": "string"},
				},
				// allow other keys, but enforce that their values are number
				"additionalProperties": map[string]any{"type": "number"},
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// matching pattern key uses string, other key must be number; this should pass
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"app-a":"x","other":1}}`))); err != nil {
		t.Fatalf("expected accept: %v", err)
	}
	// type mismatch on additionalProperties value (should be number)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"other":"y"}}`))); err == nil {
		t.Fatalf("expected type mismatch for additionalProperties schema")
	}
}

func TestImport_PropertyNames_Pattern_Only(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				"propertyNames": map[string]any{
					"pattern": "^app-",
				},
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// accept
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"app-a":"x"}}`))); err != nil {
		t.Fatalf("expected accept: %v", err)
	}
	// reject other key
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"bad":"x"}}`))); err == nil {
		t.Fatalf("expected propertyNames pattern violation for 'bad'")
	}
}

func TestImport_PropertyNames_WithAdditionalPropertiesTrue_RejectsUnmatchedKeys(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"labels": map[string]any{
				"type": "object",
				"propertyNames": map[string]any{
					"pattern": "^app-",
				},
				"additionalProperties": true,
			},
		},
		"required":             []any{"labels"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes([]byte(`{"labels":{"other":"y"}}`))); err == nil {
		t.Fatalf("expected propertyNames pattern violation for 'other'")
	}
}
