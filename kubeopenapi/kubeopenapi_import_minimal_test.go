package kubeopenapi_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Minimal_ObjectRequired_StrictUnknown_OK(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required":             []any{"name"},
		"additionalProperties": false,
	}
	s, diag, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if diag.HasWarnings() {
		t.Logf("warnings: %v", diag.Warnings())
	}

	js := []byte(`{"name":"ok"}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if v["name"] != "ok" {
		t.Fatalf("unexpected value: %#v", v)
	}
}

func TestImport_Minimal_StrictUnknown_RaisesIssue(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required":             []any{"name"},
		"additionalProperties": false,
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"name":"ok","zzz":1}`)
	_, err = goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err == nil {
		t.Fatalf("expected unknown_key error")
	}
	if iss, ok := goskema.AsIssues(err); ok {
		found := false
		for _, it := range iss {
			if it.Code == goskema.CodeUnknownKey && it.Path == "/zzz" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected unknown_key at /zzz, got %v", iss)
		}
	}
}

func TestImport_Minimal_PreserveUnknown_Passthrough(t *testing.T) {
	ctx := context.Background()
	schema := map[string]any{
		"type":                                 "object",
		"x-kubernetes-preserve-unknown-fields": true,
		"properties": map[string]any{
			"name":     map[string]any{"type": "string"},
			"_unknown": map[string]any{"type": "object"},
		},
	}
	s, _, err := kubeopenapi.Import(schema, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	js := []byte(`{"name":"ok","extra":1}`)
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	unk, _ := v["_unknown"].(map[string]any)
	if unk == nil || unk["extra"] == nil {
		t.Fatalf("expected passthrough unknown at _unknown.extra, got %#v", v["_unknown"])
	}
}
