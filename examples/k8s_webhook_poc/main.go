package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/kubeopenapi"
	"gopkg.in/yaml.v3"
)

// Minimal AdmissionReview (v1) structs for PoC
type groupVersionKind struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

type groupVersionResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

type admissionRequest struct {
	UID       string               `json:"uid"`
	Kind      groupVersionKind     `json:"kind"`
	Resource  groupVersionResource `json:"resource"`
	Namespace string               `json:"namespace,omitempty"`
	Operation string               `json:"operation,omitempty"`
	Object    json.RawMessage      `json:"object"`
}

type status struct {
	Code    int32  `json:"code,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type admissionResponse struct {
	UID              string            `json:"uid"`
	Allowed          bool              `json:"allowed"`
	Status           *status           `json:"status,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	AuditAnnotations map[string]string `json:"auditAnnotations,omitempty"`
}

type admissionReview struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Request    *admissionRequest  `json:"request,omitempty"`
	Response   *admissionResponse `json:"response,omitempty"`
}

// Global schema loaded from CRD YAML
var (
	schema           goskema.Schema[map[string]any]
	expectedKind     string
	expectedGroup    string
	expectedVersion  string
	expectedResource string
)

func main() {
	addr := getenv("KWEB_ADDR", ":18080")
	crdPath := getenv("KWEB_CRD_FILE", "examples/k8s_webhook_poc/crd.yaml")
	kind := getenv("KWEB_CRD_KIND", "Widget")

	// Import CRD OpenAPI v3 schema for the target Kind
	b, err := os.ReadFile(crdPath)
	if err != nil {
		log.Fatalf("read CRD file: %v", err)
	}
	s, diag, err := kubeopenapi.ImportYAMLForCRDKind(b, kind, kubeopenapi.Options{})
	if err != nil {
		log.Fatalf("import CRD: %v", err)
	}
	if diag.HasWarnings() {
		log.Printf("[k8s-webhook-poc] import warnings: %v", diag.Warnings())
	}
	schema = s
	expectedKind = kind
	// Derive expected GVR from the CRD YAML for additional request validation
	if g, v, r, ok := parseExpectedGVRFromYAML(b, kind); ok {
		expectedGroup, expectedVersion, expectedResource = g, v, r
		log.Printf("[k8s-webhook-poc] expected GVR: %s/%s, resource=%s", expectedGroup, expectedVersion, expectedResource)
	} else {
		log.Printf("[k8s-webhook-poc] warning: failed to derive expected GVR from CRD; skipping GVR check")
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/validate", handleValidate)

	srv := &http.Server{
		Addr:              addr,
		ReadTimeout:       getenvDuration("KWEB_READ_TIMEOUT", 5*time.Second),
		ReadHeaderTimeout: getenvDuration("KWEB_READ_HEADER_TIMEOUT", 2*time.Second),
		WriteTimeout:      getenvDuration("KWEB_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       getenvDuration("KWEB_IDLE_TIMEOUT", 60*time.Second),
	}
	certFile := getenv("KWEB_TLS_CERT_FILE", "")
	keyFile := getenv("KWEB_TLS_KEY_FILE", "")
	if certFile != "" && keyFile != "" {
		log.Printf("[k8s-webhook-poc] listening TLS on %s (kind=%s, crd=%s)", addr, kind, crdPath)
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Fatal(err)
		}
		return
	}
	log.Printf("[k8s-webhook-poc] listening on %s (kind=%s, crd=%s)", addr, kind, crdPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	// Panic guard to avoid taking down the server on unexpected bugs
	defer func() {
		if rec := recover(); rec != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal error"})
		}
	}()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Basic hardening: size limit and content-type
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MiB
	defer r.Body.Close()
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "unsupported Content-Type"})
		return
	}
	var ar admissionReview
	if err := json.NewDecoder(r.Body).Decode(&ar); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid AdmissionReview: " + err.Error()})
		return
	}
	if ar.Request == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "missing request"})
		return
	}
	// Ensure the request GVK matches the imported CRD kind
	if expectedKind != "" && ar.Request.Kind.Kind != expectedKind {
		resp := &admissionResponse{
			UID:     ar.Request.UID,
			Allowed: false,
			Status:  &status{Code: 422, Reason: "Invalid", Message: "kind mismatch: expected " + expectedKind + ", got " + ar.Request.Kind.Kind},
		}
		out := admissionReview{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview", Response: resp}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	// Ensure the request GVR matches the CRD-derived expectation (when available)
	if expectedGroup != "" && expectedVersion != "" && expectedResource != "" {
		gvr := ar.Request.Resource
		if gvr.Group != expectedGroup || gvr.Version != expectedVersion || gvr.Resource != expectedResource {
			resp := &admissionResponse{
				UID:     ar.Request.UID,
				Allowed: false,
				Status:  &status{Code: 422, Reason: "Invalid", Message: "resource mismatch: expected group=" + expectedGroup + ", version=" + expectedVersion + ", resource=" + expectedResource + "; got group=" + gvr.Group + ", version=" + gvr.Version + ", resource=" + gvr.Resource},
			}
			out := admissionReview{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview", Response: resp}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
			return
		}
	}

	// Validate the object using imported schema.
	opt := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true, Include: []string{"/spec"}},
		Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
		MaxDepth:   256,
		MaxBytes:   10 << 20, // defensive cap at engine level too (10MiB)
		PathRender: goskema.PathRenderOpt{Intern: true, Lazy: true},
	}

	// Per-request timeout for additional safety
	ctx, cancel := context.WithTimeout(r.Context(), getenvDuration("KWEB_HANDLER_TIMEOUT", 2*time.Second))
	defer cancel()
	dm, err := goskema.ParseFromWithMeta(ctx, schema, goskema.JSONReader(bytes.NewReader(ar.Request.Object)), opt)
	resp := &admissionResponse{UID: ar.Request.UID}
	if err != nil {
		if iss, ok := goskema.AsIssues(err); ok {
			// Log structured issues; return as AdmissionResponse denial
			for _, it := range iss {
				log.Printf("[deny] uid=%s code=%s path=%s msg=%s", ar.Request.UID, it.Code, it.Path, it.Message)
			}
			resp.Allowed = false
			resp.Status = &status{Code: 422, Reason: "Invalid", Message: summarizeIssues(iss)}
			// Optional: expose first few issue summaries as warnings/audit annotations
			resp.Warnings = firstN(issueSummaries(iss), 5)
			if resp.AuditAnnotations == nil {
				resp.AuditAnnotations = map[string]string{}
			}
			resp.AuditAnnotations["goskema/issues"] = compactIssues(iss)
			// Always attach presence summary (summarizePresence handles nil)
			resp.AuditAnnotations["goskema/presence"] = summarizePresence(dm.Presence)
		} else {
			resp.Allowed = false
			resp.Status = &status{Code: 422, Reason: "Invalid", Message: err.Error()}
		}
	} else {
		resp.Allowed = true
		if resp.AuditAnnotations == nil {
			resp.AuditAnnotations = map[string]string{}
		}
		// Always attach presence summary for observability
		resp.AuditAnnotations["goskema/presence"] = summarizePresence(dm.Presence)
	}

	out := admissionReview{
		APIVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
		Response:   resp,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func summarizeIssues(iss goskema.Issues) string {
	if len(iss) == 0 {
		return ""
	}
	// Keep it short; details are in auditAnnotations
	if len(iss) == 1 {
		return iss[0].Code + " at " + iss[0].Path + ": " + iss[0].Message
	}
	return iss[0].Code + " at " + iss[0].Path + ", and " + itoa(len(iss)-1) + " more"
}

func compactIssues(iss goskema.Issues) string {
	const max = 8
	var out []map[string]any
	for i, it := range iss {
		if i >= max {
			break
		}
		out = append(out, map[string]any{
			"path": it.Path,
			"code": it.Code,
			"msg":  it.Message,
		})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func issueSummaries(iss goskema.Issues) []string {
	const max = 8
	var out []string
	for i, it := range iss {
		if i >= max {
			break
		}
		out = append(out, it.Code+"@"+it.Path+": "+it.Message)
	}
	return out
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}

func firstN[T any](in []T, n int) []T {
	if n <= 0 || len(in) == 0 {
		return nil
	}
	if len(in) < n {
		return in
	}
	return in[:n]
}

func itoa(n int) string { return strconv.Itoa(n) }

// summarizePresence compacts PresenceMap into a small JSON string for auditAnnotations.
// It reports simple counts for seen, null, and defaultApplied flags across paths.
func summarizePresence(pm goskema.PresenceMap) string {
	if pm == nil {
		return "{}"
	}
	var seen, wasNull, def int
	for _, v := range pm {
		if v&goskema.PresenceSeen != 0 {
			seen++
		}
		if v&goskema.PresenceWasNull != 0 {
			wasNull++
		}
		if v&goskema.PresenceDefaultApplied != 0 {
			def++
		}
	}
	out := map[string]int{
		"seen":           seen,
		"null":           wasNull,
		"defaultApplied": def,
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// parseExpectedGVRFromYAML extracts group, version, and plural resource for the given Kind
// from a CRD YAML bundle. It returns ok=false if not found or malformed.
func parseExpectedGVRFromYAML(data []byte, kind string) (group, version, resource string, ok bool) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var node any
		if err := dec.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return "", "", "", false
		}
		m := yamlToStringMap(node)
		if m == nil || m["kind"] != "CustomResourceDefinition" {
			continue
		}
		spec, _ := m["spec"].(map[string]any)
		names, _ := spec["names"].(map[string]any)
		if k, _ := names["kind"].(string); k != kind {
			continue
		}
		g, _ := spec["group"].(string)
		plural, _ := names["plural"].(string)
		ver, _ := pickStorageOrFirstVersion(spec)
		if g != "" && plural != "" && ver != "" {
			return g, ver, plural, true
		}
	}
	return "", "", "", false
}

// yamlToStringMap normalizes yaml-decoded maps into map[string]any recursively.
func yamlToStringMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[k] = yamlNormalize(vv)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			out[ks] = yamlNormalize(vv)
		}
		return out
	default:
		return nil
	}
}

func yamlNormalize(v any) any {
	switch t := v.(type) {
	case map[string]any, map[any]any:
		return yamlToStringMap(t)
	case []any:
		arr := make([]any, len(t))
		for i := range t {
			arr[i] = yamlNormalize(t[i])
		}
		return arr
	default:
		return v
	}
}

// pickStorageOrFirstVersion finds the storage=true version from spec.versions, falling back to the first name.
func pickStorageOrFirstVersion(spec map[string]any) (string, bool) {
	vers, _ := spec["versions"].([]any)
	var first string
	for i, it := range vers {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		name, _ := m["name"].(string)
		if i == 0 {
			first = name
		}
		if b, _ := m["storage"].(bool); b && name != "" {
			return name, true
		}
	}
	if first != "" {
		return first, true
	}
	return "", false
}
