package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/reoring/goskema/kubeopenapi"
)

// helper to build AdmissionReview JSON with given object JSON
func wrapAR(obj string) string {
	return `{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"00000000-0000-0000-0000-000000000001","kind":{"group":"demo.example.com","version":"v1","kind":"Widget"},"resource":{"group":"demo.example.com","version":"v1","resource":"widgets"},"object":` + obj + `}}`
}

func setupSchema(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile("crd.yaml")
	if err != nil {
		t.Fatalf("read crd: %v", err)
	}
	s, _, err := kubeopenapi.ImportYAMLForCRDKind(b, "Widget", kubeopenapi.Options{})
	if err != nil {
		t.Fatalf("import CRD: %v", err)
	}
	schema = s
	expectedKind = "Widget"
}

func TestHandleValidate_UnknownStrict_RebasedPath(t *testing.T) {
	setupSchema(t)
	obj := `{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","oops":"x"}}`
	body := wrapAR(obj)
	r := httptest.NewRequest("POST", "/validate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleValidate(w, r)

	var out admissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Response == nil || out.Response.Allowed {
		t.Fatalf("expected Allowed=false, got: %+v", out.Response)
	}
	if out.Response.Status == nil || !strings.Contains(out.Response.Status.Message, "/spec/oops") {
		t.Fatalf("expected path rebased to /spec/oops, got: %+v", out.Response.Status)
	}
}

func TestHandleValidate_NullableNote_Allowed(t *testing.T) {
	setupSchema(t)
	obj := `{"apiVersion":"demo.example.com/v1","kind":"Widget","metadata":{"name":"w1"},"spec":{"name":"n","note":null}}`
	body := wrapAR(obj)
	r := httptest.NewRequest("POST", "/validate", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleValidate(w, r)

	var out admissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Response == nil || !out.Response.Allowed {
		t.Fatalf("expected Allowed=true, got: %+v", out.Response)
	}
	// presence annotation should exist
	if out.Response.AuditAnnotations == nil || out.Response.AuditAnnotations["goskema/presence"] == "" {
		t.Fatalf("expected presence annotation, got: %+v", out.Response.AuditAnnotations)
	}
}
