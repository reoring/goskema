package kubeopenapi_test

import (
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_ServiceMonitorMiniSchema_ValidAndInvalid(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/servicemonitor_mini_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, diag, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if diag.HasWarnings() {
		// informative only for now
		t.Logf("warnings: %v", diag.Warnings())
	}

	// valid payload
	good := []byte(`{
		"labels": {"app": "prom"},
		"targets": ["a:9090","b:9090"],
		"sampleLimit": "1000"
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected valid servicemonitor mini payload, got: %v", err)
	}

	// duplicate set element in targets
	badDup := []byte(`{
		"labels": {"app": "prom"},
		"targets": ["a:9090","a:9090"],
		"sampleLimit": 1000
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badDup)); err == nil {
		t.Fatalf("expected duplicate error for targets set")
	}

	// wrong type in labels value
	badType := []byte(`{
		"labels": {"app": 1},
		"targets": ["a:9090"],
		"sampleLimit": 1000
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(badType)); err == nil {
		t.Fatalf("expected type error for labels value")
	}
}

func TestImport_ServiceMonitorCRDWrapped_ValidAndInvalid(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/servicemonitor_crd_wrapped.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, diag, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	if diag.HasWarnings() {
		t.Logf("warnings: %v", diag.Warnings())
	}

	good := []byte(`{
		"labels": {"app": "prom"},
		"targets": ["a:9090","b:9090"],
		"sampleLimit": 100
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected valid servicemonitor mini payload (crd-wrapped), got: %v", err)
	}

	bad := []byte(`{
		"labels": {"app": "prom"},
		"targets": ["a:9090","a:9090"],
		"sampleLimit": 100
	}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected duplicate error for targets set (crd-wrapped)")
	}
}

func TestImport_ServiceMonitor_PropertyNames_OnLabels(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/servicemonitor_propertynames_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	// accept: keys match ^app-
	good := []byte(`{"labels":{"app-a":"x","app-b":"y"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected accept for propertyNames pattern: %v", err)
	}
	// reject: key not matching pattern
	bad := []byte(`{"labels":{"bad":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected propertyNames pattern violation for 'bad'")
	}
}

// The real ServiceMonitor defines spec.endpoints as an array of Endpoint objects.
// We add a targeted test for list-map-keys / required keys on a representative map-list field.
func TestImport_ServiceMonitor_FromBundle_Endpoints_MinimalAndDuplicate(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "ServiceMonitor", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}

	// minimal object with empty spec should pass
	base := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(base)); err != nil {
		t.Fatalf("base minimal spec should pass: %v", err)
	}

	// endpoints present with two identical entries should trigger duplicate if list-map-keys is configured there.
	// In the real schema endpoints is an array of objects without explicit list-map-keys,
	// so for now this asserts acceptance (no duplicate constraint yet) and serves as a placeholder.
	dup := []byte(`{
        "apiVersion":"monitoring.coreos.com/v1",
        "kind":"ServiceMonitor",
        "spec":{
            "endpoints":[
                {"port":"http"},
                {"port":"http"}
            ]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(dup)); err != nil {
		t.Fatalf("endpoints duplicate by port should currently pass (no list-map-keys): %v", err)
	}
}
