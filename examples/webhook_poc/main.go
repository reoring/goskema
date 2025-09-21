package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// Webhook input: mixture of metadata and a huge array of items.
// We validate streaming from http.Request.Body without loading all into memory.
type Item struct {
	ID   string `json:"id"`
	Note string `json:"note"`
}

// Request shape for PoC
// {
//   "source": "webhook-x",
//   "items": [ {"id":"..."}, ... ],
//   "meta": {"trace":"..."}
// }

func buildSchema() goskema.Schema[map[string]any] {
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).Required().
		Field("note", g.StringOf[string]()).
		UnknownStrip().
		MustBind()

	// Top-level object with strict unknowns to surface surprises early.
	root := g.Object().
		Field("source", g.StringOf[string]()).Required().
		Field("items", g.ArrayOf(item)).Required().
		Field("meta", g.MapOf(g.String())).Optional().
		UnknownStrict().
		MustBuild()

	return root
}

type httpError struct {
	Error string `json:"error"`
}

type issuesResponse struct {
	Issues []goskema.Issue `json:"issues"`
}

// POST /webhook
// Content-Type: application/json
// Body: large JSON with items array
func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct == "" || !strings.HasPrefix(ct, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_ = json.NewEncoder(w).Encode(httpError{Error: "Content-Type must be application/json"})
		return
	}
	ctx := r.Context()
	s := buildSchema()

	// Presence collection and basic enforcement
	opt := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true},
		Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
		MaxDepth:   128,
		PathRender: goskema.PathRenderOpt{Intern: true},
	}

	// Hard cap request size with both HTTP layer and parser option (default 20MiB; set POC_MAX_BYTES to override)
	maxBytes := getenvBytes("POC_MAX_BYTES", 20<<20)
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		opt.MaxBytes = maxBytes
	}

	// We want presence-aware result, so use ParseFromWithMeta with JSONReader over Body.
	dm, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONReader(r.Body), opt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if iss, ok := goskema.AsIssues(err); ok {
			json.NewEncoder(w).Encode(issuesResponse{Issues: iss})
			return
		}
		_ = json.NewEncoder(w).Encode(httpError{Error: err.Error()})
		return
	}

	// Return: 1) canonical output, 2) preserving output example, 3) presence map stats
	// Canonical output (sorted keys, defaults materialized)
	canonical := dm.Value

	// Preserving output for top-level object (can be disabled via POC_PRESERVE=0)
	preserveEnabled := getenvBool("POC_PRESERVE", true)
	var preserve any
	if preserveEnabled {
		preserve = goskema.EncodePreservingObject(dm)
	}

	resp := map[string]any{
		"ok":         true,
		"canonical":  canonical,
		"preserving": preserve,
		"presence":   dm.Presence, // returned as-is for UI/log
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /items
// Content-Type: application/json
// Body: large JSON array of items
func handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct == "" || !strings.HasPrefix(ct, "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		_ = json.NewEncoder(w).Encode(httpError{Error: "Content-Type must be application/json"})
		return
	}
	ctx := r.Context()
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).Required().
		Field("note", g.StringOf[string]()).
		UnknownStrip().
		MustBind()
	arr := g.Array[Item](item)

	opt := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true},
		Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
		MaxDepth:   128,
		PathRender: goskema.PathRenderOpt{Intern: true},
	}

	// Hard cap request size (default 20MiB; set POC_MAX_BYTES to override)
	maxBytes := getenvBytes("POC_MAX_BYTES", 20<<20)
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		opt.MaxBytes = maxBytes
	}

	dm, err := goskema.ParseFromWithMeta(ctx, arr, goskema.JSONReader(r.Body), opt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if iss, ok := goskema.AsIssues(err); ok {
			_ = json.NewEncoder(w).Encode(issuesResponse{Issues: iss})
			return
		}
		_ = json.NewEncoder(w).Encode(httpError{Error: err.Error()})
		return
	}

	resp := map[string]any{
		"ok":        true,
		"canonical": dm.Value,
		"presence":  dm.Presence,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	addr := getenv("POC_ADDR", ":8080")
	http.HandleFunc("/webhook", handler)
	http.HandleFunc("/items", handleItems)
	srv := &http.Server{
		Addr:              addr,
		ReadTimeout:       getenvDuration("POC_READ_TIMEOUT", 5*time.Second),
		ReadHeaderTimeout: getenvDuration("POC_READ_HEADER_TIMEOUT", 2*time.Second),
		WriteTimeout:      getenvDuration("POC_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       getenvDuration("POC_IDLE_TIMEOUT", 60*time.Second),
	}
	log.Printf("[webhook_poc] listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// getenvBytes parses an environment variable representing size in bytes.
// Examples: "20971520" (bytes), "20MiB", "20MB", "512KiB", "512KB".
func getenvBytes(k string, def int64) int64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	s := strings.TrimSpace(strings.ToLower(v))
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "mib"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "mib")
	case strings.HasSuffix(s, "mb"):
		mult = 1000 * 1000
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "kib"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "kib")
	case strings.HasSuffix(s, "kb"):
		mult = 1000
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		mult = 1
		s = strings.TrimSuffix(s, "b")
	}
	s = strings.TrimSpace(s)
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(n * float64(mult))
	}
	return def
}

// getenvDuration parses a duration from env (e.g., "5s", "200ms").
// If it's a plain integer, it's treated as seconds.
func getenvDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	return def
}

// getenvBool parses boolean-ish strings; default returned when unset or unrecognized.
func getenvBool(k string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "t", "true", "y", "yes", "on":
		return true
	case "0", "f", "false", "n", "no", "off":
		return false
	default:
		return def
	}
}

// How to run:
//   go run ./examples/webhook_poc
// Then POST:
//   curl -sS -X POST http://localhost:8080/webhook \
//     -H 'Content-Type: application/json' \
//     --data '{"source":"poc","items":[{"id":"a"},{"id":1},{"id":"c"}],"meta":{"trace":"t-1"}}' | jq .
// Expect 400 with issues and paths like "/items/1".
// Success example:
//   curl -sS -X POST http://localhost:8080/webhook \
//     -H 'Content-Type: application/json' \
//     --data '{"source":"poc","items":[{"id":"a","note":"n"},{"id":"b"}]}' | jq .
