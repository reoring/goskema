package kubeopenapi_test

import (
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestImport_Prometheus_FromBundle_MinimalOK(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Prometheus", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	js := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"Prometheus","spec":{}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js)); err != nil {
		t.Fatalf("minimal Prometheus object should pass: %v", err)
	}
}

func TestImport_Prometheus_ContainersPorts_DuplicateDetected(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("crds/bundle.yaml")
	if err != nil {
		t.Skipf("bundle.yaml not present: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Prometheus", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	// two identical container ports (containerPort+protocol) should violate list-map-keys uniqueness
	dup := []byte(`{
        "apiVersion":"monitoring.coreos.com/v1",
        "kind":"Prometheus",
        "spec":{
            "containers":[{
                "name":"extra",
                "ports":[
                    {"containerPort":8080, "protocol":"TCP"},
                    {"containerPort":8080, "protocol":"TCP"}
                ]
            }]
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(dup)); err == nil {
		t.Fatalf("expected duplicate_item for containers[0].ports")
	} else if iss, ok := goskema.AsIssues(err); ok {
		found := false
		for _, it := range iss {
			if it.Code == "duplicate_item" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected duplicate_item, got: %v", iss)
		}
	}
}
