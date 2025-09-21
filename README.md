# Goskema

High-performance, type-safe schema/codec foundation for Go. It addresses Zod's real-world pain points (unknown/duplicate/presence handling, huge inputs, machine-readable errors, contract distribution) and optimizes them for practical use in Go.

* Centers on Schema ↔ Codec to unify wire (JSON, etc.) ↔ domain (Go types) with consistent semantics
* Tracks Presence (missing / null / default-applied) across all paths and reproduces it with Preserve encoding
* Detects duplicate keys, supports explicit unknown-key policies (Strict/Strip/Passthrough), and provides DoS guards (MaxBytes/Depth)
* Streaming validation to handle huge arrays/deep nesting efficiently
* Standardizes machine-readable errors with JSON Pointer + code (plug directly into UI/audit/observability)
* Exports JSON Schema and imports OpenAPI (Kubernetes CRD) for contract distribution and compatibility checks

### TL;DR

- If you handle external JSON, large inputs, or strict contracts, use goskema.
- One schema to unify validation, type projection, error reporting, and contract distribution.
- Standardized presence/duplicate/unknown handling, streaming, and Issues (JSON Pointer).

You can get something working with the standard library, but as requirements grow and the lifespan increases, logic scatters and duplicates. With "schema as the single source of truth", goskema integrates validation, errors, codecs, docs, and tests to provide a robust, change-friendly foundation.

---

## Table of contents

* [When you don't need it / when you do](#when-you-dont-need-it--when-you-do)
* [Quickest demo](#quickest-demo-presence-missingnulldefault-and-preserve-output)
* [Features](#features)
* [Why goskema?](#why-goskema)
* [Using with encoding/json / validator](#using-with-encodingjson--validator)
* [JSON Schema export / OpenAPI import](#json-schema-export--openapi-import)
* [Webhook examples (HTTP / Kubernetes)](#webhook-examples-http--kubernetes)
* [Quick start](#quick-start)
* [Streaming (safe for huge arrays)](#streaming-safe-for-huge-arrays)
* [Enforcement (duplicate keys, depth, size limits)](#enforcement-duplicate-keys-depth-size-limits)
* [WithMeta / Presence (distinguishing missing/null/default)](#withmeta--presence-distinguishing-missingnulldefault)
* [NumberMode tips (precision vs speed)](#numbermode-tips-precision-vs-speed)
* [Switching JSON drivers](#switching-json-drivers)
* [DSL overview](#dsl-overview)
* [Value proposition / ROI](#value-proposition--roi)
* [When not to use it (kill criteria)](#when-not-to-use-it-kill-criteria)
* [Phased rollout](#phased-rollout)
* [FAQ](#faq)
* [Status / Roadmap](#status--roadmap)
* [Benchmarks (excerpt)](#benchmarks-excerpt)
* [Stability policy / Supported Go versions](#stability-policy--supported-go-versions)
* [License / Contributing](#license--contributing)
* [Extension points (Codec / Refine / Format)](#extension-points-codec--refine--format)
* [Error model](#error-model)

---

## When you don't need it / when you do

* You don't need it (encoding/json or a validator is enough) if:
  * Input is small and trusted (e.g., internally generated); you don't strictly manage unknown or duplicate keys
  * You don't need to distinguish missing / null / default, nor track presence for PATCH
  * You don't need streaming validation for huge arrays/deep nesting
  * String error messages are sufficient (no need to pipe machine-readable errors to UI/audit)

* You do need it (where goskema shines) if you want:
  * Strict handling of Presence (missing / null / default-applied) and Preserve output
  * JSON-level duplicate-key detection and explicit unknown policies (Strict/Strip/Passthrough)
  * Streaming validation for external/large input to improve DoS resilience
  * Two-way conversion (Codec) and JSON Schema export / OpenAPI import for contract distribution
  * Machine-readable errors with JSON Pointer + code to plug into UI/audit

---

## Quickest demo: Presence (missing/null/default) and Preserve output

```go
// Schema (for map): nickname has a default
obj, _ := g.Object().
  Field("name",     g.StringOf[string]()).Required().
  Field("nickname", g.StringOf[string]()).Default("anon").
  UnknownStrict().
  Build()

// Input: nickname is null (not missing)
dm, _ := goskema.ParseFromWithMeta(ctx, obj, goskema.JSONBytes([]byte(`{"name":"alice","nickname":null}`)))

// Presence lets you tell missing/null/default-applied apart
_ = (dm.Presence["/nickname"] & goskema.PresenceWasNull) != 0 // => true

// Preserve output: keep missing as missing, keep null as null, drop values materialized only by default
out := goskema.EncodePreservingObject(dm)
_ = out // => map[string]any{"name":"alice","nickname":nil}
```

### Quickest demo: duplicate JSON keys (with Path/Code)

```go
obj := g.Object().MustBuild()
opt := goskema.ParseOpt{Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error}}
_, err := goskema.ParseFrom(ctx, obj, goskema.JSONBytes([]byte(`{"x":1,"x":2}`)), opt)
if iss, ok := goskema.AsIssues(err); ok {
  // e.g., code="duplicate_key", path="/x"
  fmt.Printf("%s at %s\n", iss[0].Code, iss[0].Path)
}
```

---

## Features

* Schema/Codec integration (wire ↔ domain)
* Explicit semantics (unknown / duplicate / presence)
* Strong for streaming and huge inputs
* Type-safe and extensible (format/codec additions)

---

## Why goskema?

We tackle real-world pain points that the standard stack can't fully address:

1. Duplicate-key detection  
   encoding/json can't detect duplicates because "last one wins" semantics lose the history. goskema detects them in streaming and returns `path="/x" code="duplicate_key"`. Great for audit/security/UI.

2. Declared and consistent unknown-key policy  
   DisallowUnknownFields is too coarse and can't declare project-wide semantics (Strict/Strip/Passthrough). With goskema, declare once and get JSON Pointer-based errors.

3. Mechanical presence tracking  
   Distinguishing missing vs null vs default safely in pure Go is cumbersome. goskema tracks presence bits end-to-end and ties them to PATCH/diff/Preserve output.

4. Streaming validation for DoS resilience  
   Simply composing Decoder + custom checks makes it hard to build a safe pipeline for validation/type projection/error collection. goskema validates large arrays/deep nesting in stages and fixes MaxBytes/Depth/FailFast by declaration.

5. Machine-readable errors  
   Reserved codes + JSON Pointer + optional hints. Standardized for UI highlighting/audit/observability; useful for SLO analysis.

6. Codec (wire↔domain) + Schema  
   Marshaler/Unmarshaler split contracts and runtime behavior. goskema bridges Codec/Refine with JSON Schema export and OpenAPI import to unify contract and implementation.

> For small, trusted input, the standard stack is enough. For external/large/mission-critical inputs, goskema pays off.

---

## Using with encoding/json / validator (at a glance)

| Aspect                         | goskema                                   | encoding/json + validator                  |
| :----------------------------- | :---------------------------------------- | :----------------------------------------- |
| unknown (explicit)             | UnknownStrict/Strip/Passthrough           | DisallowUnknownFields etc. ad hoc          |
| duplicate key                  | Detected (streaming)                      | Not supported (last value wins)            |
| presence (missing/null/default)| Tracked + Preserve output                 | Not supported (manual, error-prone)        |
| streaming                      | Supported (staged validation, enforcement)| Not supported (assumes full in-memory)     |
| error model                    | JSON Pointer + code (Issues)              | ValidationErrors / strings                  |

See `docs/compare.md` for details.

---

## JSON Schema export / OpenAPI import

```go
// JSON Schema export
sch, _ := userSchema.JSONSchema()
b, _ := json.MarshalIndent(sch, "", "  ")
fmt.Println(string(b))

// OpenAPI (Kubernetes CRD) import
s, _, _ := kubeopenapi.ImportYAMLForCRDKind(crdYAML, "Widget", kubeopenapi.Options{})
_ = s
```

Use cases:

* Pre-validation for clients/LSP/CLI
* Foundation for type generation, form generation, API documentation
* Contract distribution and compatibility checks across services

Notes:

* Presence (missing/null/default) and duplicate-key detection can't be fully represented in plain JSON Schema (enforced at runtime)
* Unknown policy mapping: UnknownStrict → `additionalProperties:false`; UnknownStrip/Passthrough → `true`
* `default` is an annotation; applying values is client-dependent
* Some aspects of Codec/Refine may not fully round-trip to JSON Schema (use `x-goskema-*` annotations if needed)

Minimal save example:

```go
sch, _ := userSchema.JSONSchema()
b, _ := json.MarshalIndent(sch, "", "  ")
_ = os.WriteFile("schema.json", b, 0o644)
```

---

## Webhook examples (HTTP / Kubernetes)

Key points:

* Streaming-first: wrap `http.Request.Body` with `goskema.JSONReader(r.Body)`
* Pipe Issues to UI/audit: JSON Pointer + code
* Clear control over unknown/duplicate handling
* Presence is easy: `PresenceOpt{Collect:true}`

HTTP example:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    s := buildSchema() // g.Object()...UnknownStrict()...MustBuild()
    opt := goskema.ParseOpt{
        Presence:   goskema.PresenceOpt{Collect: true},
        Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
    }
    dm, err := goskema.ParseFromWithMeta(r.Context(), s, goskema.JSONReader(r.Body), opt)
    if err != nil {
        if iss, ok := goskema.AsIssues(err); ok {
            _ = json.NewEncoder(w).Encode(map[string]any{"issues": iss})
            return
        }
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    _ = json.NewEncoder(w).Encode(map[string]any{
        "ok": true, "canonical": dm.Value, "presence": dm.Presence,
    })
}
```

Kubernetes ValidatingWebhook (AdmissionReview v1):

```go
crd := mustRead("crd.yaml")
s, _, _ := kubeopenapi.ImportYAMLForCRDKind(crd, "Widget", kubeopenapi.Options{})
opt := goskema.ParseOpt{
    Presence:   goskema.PresenceOpt{Collect: true},
    Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
}
_, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(ar.Request.Object), opt)
resp := &admissionResponse{UID: ar.Request.UID, Allowed: err == nil}
if err != nil {
    if iss, ok := goskema.AsIssues(err); ok {
        resp.Allowed = false
        resp.Status = &status{Code: 400, Reason: "Invalid", Message: iss[0].Code + " at " + iss[0].Path}
    }
}
```

Kubernetes integration details (unknown/list-type/int-or-string, etc.) are in `docs/k8s.md`.

---

## Quick start

```go
package main

import (
    "context"
    g "github.com/reoring/goskema/dsl"
    "github.com/reoring/goskema"
)

type User struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}

func main() {
    ctx := context.Background()
    userSchema := g.ObjectOf[User]().
        Field("id",    g.StringOf[string]()).Required().
        Field("email", g.StringOf[string]()).Required().
        UnknownStrict().
        MustBind()

    user, err := goskema.ParseFrom(ctx, userSchema, goskema.JSONBytes([]byte(`{"id":"u_1","email":"x@example.com"}`)))
    _ = user
    _ = err
}
```

More examples: `docs/user-guide.md`.

* Sample: `sample-projects/user-api`
* Representative tests: `dsl/zod_basics_test.go`, `dsl/codec_zod_usecases_test.go`

---

## Streaming (safe for huge arrays)

Process huge arrays incrementally without materializing everything. `ParseFrom` / `StreamParse` uses a Source to validate in stages.

```go
// Element schema (typed)
type Item struct{ ID string `json:"id"` }
itemS := g.ObjectOf[Item]().
  Field("id", g.StringOf[string]()).Required().
  UnknownStrip().
  MustBind()

// Read as io.Reader and wrap as Source
items, err := goskema.StreamParse(context.Background(), g.Array[Item](itemS), r)
_ = items; _ = err
```

* Issue.Path for failed elements uses indices like `/0`, `/2`
* Current behavior: if any element fails, the value is not returned; only Issues are returned  
  → A future iterative API (handler-style) may collect failures while returning successful elements
* Example: `benchmarks/benchmark_parsefrom_test.go` `Benchmark_StreamParse_HugeArray_Objects`
* Test: `dsl/array_stream_integration_test.go`

Unsupported schema handling:

* DSL Object/Array/Primitive are optimized. Union/custom Codec/`MapAny` etc. fall back to the legacy path automatically (same validation result; higher memory for huge input)
* Use `MaxBytes / FailFast / MaxDepth` as safety valves where needed
* See the roadmap section for future directions

---

## Enforcement (duplicate keys, depth, size limits)

`ParseFrom`/`StreamParse` performs streaming checks for duplicate keys, nesting depth, and max bytes. Issues carry precise JSON Pointer paths (e.g., `/items/0/foo`).

Minimal example:

```go
opt := goskema.ParseOpt{
    Strictness: goskema.Strictness{OnDuplicateKey: goskema.Error},
    MaxDepth:   16,
    MaxBytes:   1 << 20,
    FailFast:   true,
}
_, err := goskema.StreamParse(ctx, someArraySchema, r, opt)
_ = err
```

---

## WithMeta / Presence (distinguishing missing/null/default)

Presence is a set of flags per path indicating: "was the field present in input?", "was it null?", "was a default applied?".

```go
dm, err := goskema.ParseFromWithMeta(ctx, user, goskema.JSONBytes(data))
if dm.Presence["/nickname"]&goskema.PresenceSeen == 0 {
    // missing
}
if dm.Presence["/nickname"]&goskema.PresenceWasNull != 0 {
    // null
}
if dm.Presence["/nickname"]&goskema.PresenceDefaultApplied != 0 {
    // materialized by default
}
```

* Control the collection range with `PresenceOpt` (Include/Exclude/Collect)
* Optimize path handling with `PathRenderOpt` (Intern/Lazy)
* `EncodePreservingObject/Array` reproduces missing/null/default

Note: `EncodePreserve` requires presence metadata. Calling `EncodeWithMode(..., EncodePreserve)` without presence returns `ErrEncodePreserveRequiresPresence`.

---

## NumberMode tips (precision vs speed)

```go
// Default: NumberJSONNumber (precision-first)
v1, _ := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))

// Speed-first: float64 rounding
src := goskema.WithNumberMode(goskema.JSONBytes(js), goskema.NumberFloat64)
v2, _ := goskema.ParseFrom(ctx, s, src)
```

* Precision-first (huge integers/currency): `NumberJSONNumber` (default)
* Speed/low-overhead: `NumberFloat64`  
  See `dsl/numbermode_integration_test.go` and `docs/user-guide.md`.

---

## Switching JSON drivers (encoding/json ↔ go-json ↔ json/v2)

```go
// Use go-json explicitly
import (
    goskema "github.com/reoring/goskema"
    drv "github.com/reoring/goskema/source/gojson"
)

func init() {
    goskema.SetJSONDriver(drv.Driver())
}
```

* Build tag `-tags gojson`: use `goccy/go-json`
* Default: fall back to `encoding/json`
* Use a side-effect import to set default driver: `import _ "github.com/reoring/goskema/source"`

Using `encoding/json/v2` (experimental):

```go
// Requires GOEXPERIMENT=jsonv2 and -tags jsonv2
import (
    goskema "github.com/reoring/goskema"
    drv "github.com/reoring/goskema/source/jsonv2"
)

func init() {
    goskema.SetJSONDriver(drv.Driver())
}
```

Return to default with `goskema.UseDefaultJSONDriver()`.

JSON Schema alignment (UnknownStrip):

* UnknownStrip at runtime means "accept unknown keys, drop them during projection"
* Generated schema uses `additionalProperties:true` (accept)
  * This avoids discrepancies between IDE pre-validation and runtime behavior
  * If you want end-to-end reject, use UnknownStrict (schema `false`)

---

## DSL overview

```go
user := g.ObjectOf[User]().
  Field("id",    g.StringOf[string]()).Required().
  Field("email", g.StringOf[string]()).Required().
  UnknownStrict().
  MustBind()
```

See `docs/dsl.md` for details.

---

## Value proposition / ROI

* Incident reduction: early rejection + precise paths to prevent misconfig/vulnerabilities from duplicate/unknown keys
* Dev efficiency: remove boilerplate for presence tracking (PATCH, diffs become reliable)
* Contract operations: automate distribution and compatibility checks with JSON Schema export ↔ OpenAPI import
* SRE/Security: declare enforcement (MaxBytes/MaxDepth, etc.) to close DoS vectors

Simple estimation (example):

* (Rate of invalid external JSON) × (time to analyze/rollback)  
* (Years of operation) × (change frequency) × (cost of format drift)
* (Lines of boilerplate removed) × (bug rate) × (years of maintenance)
  → Even for just 10 boundary endpoints, avoiding one incident per month can pay back.

---

## When not to use it (kill criteria)

* Input is internal-only and trusted, and small in size
* You don't do PATCH or diff-based management
* Errors as plain log strings are enough (no UI/audit integration)

In these cases, `encoding/json` + a light validator is sufficient; goskema would be overengineering.

---

## Phased rollout

1. Start with just one place: external input Validating Webhook (Admission in Kubernetes)
2. Make duplicate/unknown/presence detection visible in logs (path, code)
3. Use Preserve output to fix at least one PATCH-related issue and build internal leverage
4. Distribute JSON Schema and run pre-validation in IDE/LSP
5. Expand to huge arrays / configuration JSON paths once benefits are clear

Internal blurb template:

> The standard stack is fine for small, trusted input. Our external JSON requires:
>
> * Streaming detection of duplicate/unknown
> * Mechanical distinction of missing/null/default (PATCH/Preserve)
> * JSON Pointer + code wired to UI/audit
>
> goskema standardizes these. Keep simple paths on the standard stack; replace boundaries with goskema for a thin, incremental rollout.

---

## FAQ

**Q. Isn't the standard stack enough?**  
A. Yes for small, trusted input. No when you need duplicate/unknown/presence handling, huge inputs, or machine-readable errors. goskema lets you declare these in one place and unifies contract and runtime.

**Q. How is this different from validator libraries?**  
A. Many focus on struct-level checks and string errors. goskema provides JSON Pointer + code for UI/audit, Presence/Preserve, streaming validation/enforcement, and Codec/Schema integration to build an end-to-end runtime.

**Q. Isn't OpenAPI / JSON Schema enough?**  
A. Great for contract distribution, but runtime semantics like Presence/duplicate/Strip/Preserve aren't fully expressible. goskema bridges contract (Schema) and runtime.

**Q. How about CUE?**  
A. CUE is a powerful declarative language. For Go-focused runtime—type projection/webhooks/Codec/Refine integrated—goskema is a pragmatic fit. They can be combined.

**Q. What about performance?**  
A. Validation adds cost over plain Unmarshal, but streaming optimizations keep behavior stable even for huge inputs. See `docs/benchmarks.md`.

---

## Status / Roadmap

* Currently prioritizing stabilization of the interpreted engine
* Compiled (code generation) pipeline is planned (details will be integrated into a user-facing document)

---

## Benchmarks (excerpt)

|               Suite | Case                              | Notes                                                                              |
| ------------------: | :-------------------------------- | :--------------------------------------------------------------------------------- |
| compare / HugeArray | go-json/sonic/encoding/json vs    | goskema's ParseAndCheck is stricter, so it can be slower than plain Unmarshal.    |
|                     |                                   | However, it remains stable even on huge arrays.                                   |

Run examples:

```bash
make bench BENCH_FILTER=.
cd benchmarks/compare && go test -bench . -benchmem
```

---

## Stability policy / supported Go versions

* v0 allows necessary breaking changes while we refine semantics. When changes occur, document rationale and migration in `CHANGELOG.md` and `docs/adr/`.
* Minimum supported Go version follows `go.mod` (currently Go 1.25.1). CI runs tests with `-race` continuously.

---

## License / Contributing

* License: see `LICENSE`
* Issues/PRs welcome. See project policy at `docs/policy.md`.

---

## Extension points (Codec / Refine / Format)

```go
// Example: RFC3339 string <-> time.Time
c := codec.TimeRFC3339()
s := g.Codec[string, time.Time](c)
// Decode
t, _ := s.Parse(ctx, "2024-01-15T10:30:00Z")
// Encode
wire, _ := c.Encode(ctx, t)
```

See `docs/extensibility.md` for details.

---

## Error model

goskema aggregates validation failures into `Issues ([]Issue)` that implements `error`. Each `Issue` holds:

* Path: JSON Pointer (e.g., `/items/2/price`)
* Code: reserved code (e.g., `invalid_type`, `required`, `unknown_key`, `duplicate_key`, `too_small`, `too_big`, `too_short`, `too_long`, `pattern`, `invalid_enum`, `invalid_format`, `discriminator_missing`, `discriminator_unknown`, `union_ambiguous`, `parse_error`, `overflow`, `truncated`)
* Message: localizable
* Hint / Cause / Offset / InputFragment: optional

```go
u, err := goskema.ParseFrom(ctx, userSchema, goskema.JSONBytes(input))
if iss, ok := goskema.AsIssues(err); ok {
    for _, it := range iss {
        log.Printf("%s at %s: %s", it.Code, it.Path, it.Message)
    }
    return
}
```

Minimal JSON example:

```json
{"issues":[{"code":"duplicate_key","path":"/x","message":"..."}]}
```

* Fail-fast with `ParseOpt{FailFast:true}`; default is collect (aggregate multiple)
* Error order is stable (object keys in ascending order; arrays by index)
* See `docs/error-model.md`, sample `examples/error-model/main.go`, and test `api_error_model_test.go`

---

> Keep lightweight paths on the standard stack; secure boundaries with goskema. Bridge contract and runtime, observability and operations, to stabilize real-world systems over the long haul.


