package kubeopenapi_test

import (
	"context"
	"os"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
)

func TestE2E_Alertmanager_Routes_Nested_DuplicateReceiver_Detected_FromMiniBundle(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_routes_mini.yaml")
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Alertmanager", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	good := []byte(`{
        "apiVersion":"monitoring.coreos.com/v1",
        "kind":"Alertmanager",
        "spec":{
            "route":{
                "receiver":"root",
                "routes":[{"receiver":"a","routes":[{"receiver":"a-child"}]}]
            }
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(good)); err != nil {
		t.Fatalf("expected nested routes unique by receiver: %v", err)
	}
	bad := []byte(`{
        "apiVersion":"monitoring.coreos.com/v1",
        "kind":"Alertmanager",
        "spec":{
            "route":{
                "routes":[{"receiver":"x"},{"receiver":"x"}]
            }
        }
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(bad)); err == nil {
		t.Fatalf("expected duplicate_item for top-level routes receiver")
	}
}

func TestE2E_Alertmanager_Matchers_String_FromMiniBundle(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_matchers_mini.yaml")
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Alertmanager", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	strOK := []byte(`{
        "apiVersion":"monitoring.coreos.com/v1",
        "kind":"Alertmanager",
        "spec":{"route":{"matchers":["env=prod","app=web"]}}
    }`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(strOK)); err != nil {
		t.Fatalf("expected string matchers accepted: %v", err)
	}
}

func TestE2E_Alertmanager_Matchers_Object_FromMiniBundle(t *testing.T) {
	ctx := context.Background()
	b, err := os.ReadFile("testdata/alertmanager_matchers_object_mini.yaml")
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Alertmanager", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import yaml err: %v", err)
	}
	objOK := []byte(`{"apiVersion":"monitoring.coreos.com/v1","kind":"Alertmanager","spec":{"route":{"matchers":[{"name":"env","value":"prod","regex":false}]}}}`)
	if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(objOK)); err != nil {
		t.Fatalf("expected object matchers accepted: %v", err)
	}
}
