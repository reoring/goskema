package kubeopenapi_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Bundle_SplitAndScan_MinimalParse(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	parts := bytes.Split(b, []byte("\n---"))
	if len(parts) < 2 {
		t.Skip("bundle has single document; skipping split scan")
	}
	var imported int
	for _, p := range parts {
		// try import this doc as-is
		s, _, err := kubeopenapi.ImportYAMLForCRDKind(p, "Prometheus", kubeopenapi.Options{})
		if err == nil {
			// minimal spec parse for Prometheus
			js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"Prometheus","spec":{}}`)
			if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
				t.Fatalf("minimal Prometheus should pass: %v", err)
			}
			imported++
			continue
		}
		s, _, err = kubeopenapi.ImportYAMLForCRDKind(p, "ServiceMonitor", kubeopenapi.Options{})
		if err == nil {
			js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"ServiceMonitor","spec":{}}`)
			if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
				t.Fatalf("minimal ServiceMonitor should pass: %v", err)
			}
			imported++
			continue
		}
		s, _, err = kubeopenapi.ImportYAMLForCRDKind(p, "Alertmanager", kubeopenapi.Options{})
		if err == nil {
			js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"Alertmanager","spec":{}}`)
			if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
				t.Fatalf("minimal Alertmanager should pass: %v", err)
			}
			imported++
			continue
		}
		s, _, err = kubeopenapi.ImportYAMLForCRDKind(p, "AlertmanagerConfig", kubeopenapi.Options{})
		if err == nil {
			js := []byte(`{"apiVersion":"monitoring.coreos.com/v1alpha1","kind":"AlertmanagerConfig","spec":{}}`)
			if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
				t.Fatalf("minimal AlertmanagerConfig should pass: %v", err)
			}
			imported++
			continue
		}
		// silently skip non-target docs
	}
	if imported == 0 {
		t.Fatalf("no target CRDs imported from split bundle")
	}
}
