package kubeopenapi_test

import (
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_PodMonitor_PropertyNames_OnLabels(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/podmonitor_propertynames_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{"labels":{"pm-a":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected accept: %v", err)
	}
	bad := []byte(`{"labels":{"bad":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected propertyNames violation for 'bad'")
	}
}

func TestImport_PodMonitor_FromBundle_MinimalOK(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "PodMonitor", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	base := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"PodMonitor","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(base)); err != nil {
		t.Fatalf("minimal PodMonitor should pass: %v", err)
	}
}

func TestImport_Probe_PropertyNames_OnLabels(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/probe_propertynames_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	s, _, err := kubeopenapi.Import(b, kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import err: %v", err)
	}
	good := []byte(`{"labels":{"probe-a":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected accept: %v", err)
	}
	bad := []byte(`{"labels":{"bad":"x"}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected propertyNames violation for 'bad'")
	}
}

func TestImport_Probe_FromBundle_MinimalOK(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Probe", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	base := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"Probe","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(base)); err != nil {
		t.Fatalf("minimal Probe should pass: %v", err)
	}
}
