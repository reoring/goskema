// Package dsl provides a type-safe schema DSL for goskema.
//
// Overview
//   - Builder API: declare JSON object semantics (unknown/required/default/refine) with Object()/Field()/Required()/UnknownStrict()/MustBuild().
//   - Typed build: generate a safe projection wire -> T with ObjectOf[T]().Field(...).MustBind().
//   - Primitives/Array/Map: String()/Bool()/NumberJSON(), Array(elem), Map(elem) are provided.
//   - AnyAdapter: adapt existing Schema[T] to AnyAdapter via `SchemaOf[T](s)` to embed into builders.
//   - Presence: obtain missing/wasNull/defaultApplied as a JSON Pointer-based map via ParseFromWithMeta/DecodeWithMeta.
//   - Streaming: stage-wise validation driven by Source for huge arrays and deep nesting (Array/Object have streaming implementations).
//
// Entry points
//   - Object(): create an object builder; chain Field/Required/Unknown* then MustBuild()/Build.
//   - ObjectOf[T](): typed builder; at the end call MustBind()/Bind[T] to construct Schema[T].
//   - Array(elem): build an array schema from an element schema (Min/Max, streaming-ready).
//   - Map(elem)/MapAny(): a map with a value schema, or a sparse passthrough map.
//   - SchemaOf[T](s): adapter from Schema[T] to AnyAdapter (to pass into Field).
//
// File layout (roles)
//   - presence_helpers.go: common helpers for Presence collection (markPresenceSubtree).
//   - array_core.go: normal path for ArraySchema (Parse/Validate/JSONSchema).
//   - array_stream.go: streaming parse for ArraySchema (ParseFromSource*).
//   - map_core.go: implementations for MapAny/Map[V] (normal/streaming).
//   - object_builder.go: objectBuilder/fieldStep and Build/MustBuild, OneOf/Variant APIs.
//   - object_core.go: normal path for objectSchema (Parse/ParseWithMeta/Validate/JSONSchema).
//   - object_stream.go: streaming for objectSchema (handling unknowns, rebasing error paths, helpers).
//   - union.go: simple Union schema based on discriminator.
//   - (aux) adapter.go/of_helpers.go/object_typed_builder.go around AnyAdapter and typed binding.
//
// Design guidelines
//   - Keep public APIs minimal; define small and clear caller-side interfaces.
//   - Align semantics of unknown/required/default/refine between runtime and JSON Schema output.
//   - Prefer minimal buffers in streaming implementations (delegate ambiguity resolution to caller policy).
//   - For large inputs, assume optimizations for Presence collection and path string generation (PathRenderOpt).
//
// Table of contents
//  1. Quickstart (minimal example)
//  2. Builder API (Object/Field/Required/Unknown*/Default/Refine)
//  3. Typed binding (ObjectOf[T]/Bind/MustBind)
//  4. Primitives/Array/Map (String/Bool/NumberJSON/Array/Map)
//  5. Presence and EncodePreserving (reconstruct missing/null/default)
//  6. Streaming (huge arrays, deep nesting)
//  7. Error model (Issues: Path/Code/Message/stable order)
//  8. File layout (where implementations live)
//
// Example (quickstart)
//
//	package main
//
//	import (
//	    "context"
//	    g "github.com/reoring/goskema/dsl"
//	    "github.com/reoring/goskema"
//	)
//
//	type User struct {
//	    ID    string `json:"id"`
//	    Email string `json:"email"`
//	}
//
//	func main() {
//	    ctx := context.Background()
//	    user := g.ObjectOf[User]().
//	        Field("id",    g.StringOf[string]()).
//	        Field("email", g.StringOf[string]()).
//	        Require("id", "email").
//	        UnknownStrict().
//	        MustBind()
//
//	    // ParseFrom: wire(JSON) -> domain(User)
//	    data := []byte(`{"id":"u_1","email":"x@example.com"}`)
//	    _, _ = goskema.ParseFrom(ctx, user, goskema.JSONBytes(data))
//	}
//
// Example (Presence and EncodePreserving)
//
//	obj, _ := g.Object().
//	    Field("name",   g.StringOf[string]()).
//	    Field("active", g.BoolOf[bool]()).Default(true).
//	    Require("name").
//	    UnknownStrict().
//	    Build()
//	// When input misses active -> default is applied
//	dm, _ := goskema.ParseFromWithMeta(ctx, obj, goskema.JSONBytes([]byte(`{"name":"alice"}`)))
//	// EncodePreservingObject drops fields filled only by defaults
//	out := goskema.EncodePreservingObject(dm)
//	_ = out // => map[string]any{"name":"alice"}
//
// Example (UnknownPassthrough)
//
//	obj := g.Object().
//	    Field("known",    g.StringOf[string]()).
//	    Field("_unknown", g.SchemaOf[map[string]any](g.MapAny())).
//	    UnknownPassthrough("_unknown").
//	    MustBuild()
//	val, _ := goskema.ParseFrom(ctx, obj, goskema.JSONBytes([]byte(`{"known":"ok","x":1,"y":"z"}`)))
//	// val["_unknown"] stores {"x":1,"y":"z"}
//	_ = val
//
// Example (switch NumberMode)
//
//	// Default: NumberJSONNumber (preserve precision)
//	_, _ = goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
//
//	// Performance-first: round to float64
//	src := goskema.WithNumberMode(goskema.JSONBytes(js), goskema.NumberFloat64)
//	_, _ = goskema.ParseFrom(ctx, s, src)
//
// Example (Refine: cross-field validation)
//
//	obj := g.Object().
//	    Field("email",   g.StringOf[string]()).
//	    Field("confirm", g.StringOf[string]()).
//	    Require("email", "confirm").
//	    Refine("email==confirm", func(ctx context.Context, m map[string]any) error {
//	        if m["email"] != m["confirm"] {
//	            return fmt.Errorf("confirm must match email")
//	        }
//	        return nil
//	    }).
//	    UnknownStrict().
//	    MustBuild()
//	_, err := goskema.ParseFrom(ctx, obj, goskema.JSONBytes([]byte(`{"email":"a","confirm":"b"}`)))
//	_ = err // returned as Issues (code: "custom")
//
// JSON Schema output hints
//
//	// Obtain JSON Schema from any Schema
//	sch, _ := s.JSONSchema()
//	// Example export (encoding/json)
//	// b, _ := json.MarshalIndent(sch, "", "  ")
//	// fmt.Println(string(b))
//	// Note: UnknownStrict => additionalProperties=false,
//	//       UnknownStrip/UnknownPassthrough => additionalProperties=true
package dsl
