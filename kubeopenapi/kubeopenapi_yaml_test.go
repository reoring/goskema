package kubeopenapi_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImportYAMLForCRDKind_ServiceMonitor(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, diag, err := kubeopenapi.ImportYAMLForCRDKind(b, "ServiceMonitor", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	if diag.HasWarnings() {
		t.Logf("warnings: %v", diag.Warnings())
	}

	// minimal valid Kubernetes object for this CRD (root requires spec)
	js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("parse err: %v", err)
	}
}

func TestImportYAMLForCRDKind_Widget_NullableNote(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("../examples/k8s_webhook_poc/crd.yaml")
	if err != nil {
		t.Fatalf("read crd: %v", err)
	}
	s, diag, err := kubeopenapi.ImportYAMLForCRDKind(b, "Widget", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import CRD: %v", err)
	}
	if diag.HasWarnings() {
		t.Logf("warnings: %v", diag.Warnings())
	}
	// spec.note is nullable: true in the CRD.
	js := []byte(`{"apiVersion":"demo.example.com/v1","kind":"Widget","spec":{"name":"n","note":null}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("parse err (nullable note should be allowed): %v", err)
	}
}

func TestImportYAMLForCRDKind_Widget_NullableNote_JSONReader_WithMeta(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("../examples/k8s_webhook_poc/crd.yaml")
	if err != nil {
		t.Fatalf("read crd: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Widget", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import CRD: %v", err)
	}
	js := []byte(`{"apiVersion":"demo.example.com/v1","kind":"Widget","spec":{"name":"n","note":null}}`)
	src := goskema.JSONReader(bytes.NewReader(js))
	if _, err := goskema.ParseFromWithMeta(ctx, s, src, goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}); err != nil {
		t.Fatalf("parse err (nullable note should be allowed): %v", err)
	}
}

func TestImportYAMLForCRDName_ServiceMonitors(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDName(b, "servicemonitors.monitoring.coreos.com", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml by name err: %v", err)
	}
	js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("parse err: %v", err)
	}
}
