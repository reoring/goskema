package codec_test

import (
	"context"
	"encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

func TestIdentity_String_Parse_Decode_Encode(t *testing.T) {
	ctx := context.Background()

	// schema := z.string()
	schema := g.String() // Schema[string]

	// schema.parse("asdf")
	v, err := schema.Parse(ctx, "asdf")
	if err != nil || v != "asdf" {
		t.Fatalf("parse err=%v v=%q", err, v)
	}

	// id := codec.Identity(schema)
	id := codec.Identity(schema) // Codec[string,string]

	// id.decode / id.encode
	dv, err := id.Decode(ctx, "asdf")
	if err != nil || dv != "asdf" {
		t.Fatalf("decode err=%v v=%q", err, dv)
	}
	ev, err := id.Encode(ctx, dv)
	if err != nil || ev != "asdf" {
		t.Fatalf("encode err=%v v=%q", err, ev)
	}
}

func TestIdentity_Bool_Parse_Decode_Encode(t *testing.T) {
	ctx := context.Background()
	schema := g.Bool()

	v, err := schema.Parse(ctx, true)
	if err != nil || v != true {
		t.Fatalf("parse err=%v v=%v", err, v)
	}

	id := codec.Identity(schema)
	dv, err := id.Decode(ctx, v)
	if err != nil || dv != true {
		t.Fatalf("decode err=%v v=%v", err, dv)
	}
	ev, err := id.Encode(ctx, dv)
	if err != nil || ev != true {
		t.Fatalf("encode err=%v v=%v", err, ev)
	}
}

func TestIdentity_NumberJSON_Parse_Decode_Encode(t *testing.T) {
	ctx := context.Background()
	schema := g.NumberJSON()

	// accept json.Number directly
	n := json.Number("123.45")
	v, err := schema.Parse(ctx, n)
	if err != nil || v != n {
		t.Fatalf("parse err=%v v=%v", err, v)
	}

	// identity codec over number schema
	id := codec.Identity[json.Number](schema)
	dv, err := id.Decode(ctx, v)
	if err != nil || dv != v {
		t.Fatalf("decode err=%v v=%v", err, dv)
	}
	ev, err := id.Encode(ctx, dv)
	if err != nil || ev != v {
		t.Fatalf("encode err=%v v=%v", err, ev)
	}
}

func TestIdentity_WithMeta_Decode_EncodePreserving(t *testing.T) {
	ctx := context.Background()
	schema := g.String()
	id := codec.Identity(schema)

	dm, err := id.DecodeWithMeta(ctx, "x")
	if err != nil || dm.Value != "x" {
		t.Fatalf("decodeWithMeta err=%v v=%v", err, dm.Value)
	}
	if dm.Presence == nil || dm.Presence["/"]&goskema.PresenceSeen == 0 {
		t.Fatalf("expected presence seen at root")
	}

	out, err := id.EncodePreserving(ctx, dm)
	if err != nil || out != "x" {
		t.Fatalf("encodePreserving err=%v v=%v", err, out)
	}
}

func TestIdentity_ObjectWithArray_Parse_Decode_Encode(t *testing.T) {
	ctx := context.Background()

	// schema: { name: string, tags: string[] (min 2) }
	schema := g.Object().
		Field("name", g.StringOf[string]()).
		Field("tags", g.ArrayOfSchema[string](g.Array(g.String()).Min(2))).
		Require("name").
		UnknownStrict().
		MustBuild()

	// parse from wire map
	v, err := schema.Parse(ctx, map[string]any{
		"name": "proj",
		"tags": []any{"a", "b"},
	})
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if v["name"].(string) != "proj" {
		t.Fatalf("unexpected name: %v", v["name"])
	}
	tags, ok := v["tags"].([]string)
	if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("unexpected tags: %#v", v["tags"])
	}

	// identity codec over object schema (map[string]any)
	id := codec.Identity(schema)
	dv, err := id.Decode(ctx, v)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	ev, err := id.Encode(ctx, dv)
	if err != nil {
		t.Fatalf("encode err: %v", err)
	}
	if ev["name"].(string) != "proj" {
		t.Fatalf("unexpected name after encode: %v", ev["name"])
	}
	tags2, ok := ev["tags"].([]string)
	if !ok || len(tags2) != 2 {
		t.Fatalf("unexpected tags after encode: %#v", ev["tags"])
	}
}

func TestIdentity_ObjectWithArray_Typed_Parse_Validate(t *testing.T) {
	ctx := context.Background()

	type Project struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	// typed schema: { name: string, tags: string[] (min 2) }
	schema := g.ObjectTyped[Project]().
		Field("name", g.StringOf[string]()).
		Field("tags", g.ArrayOfSchema[string](g.Array(g.String()).Min(2))).
		Require("name").
		UnknownStrict().
		MustBind()

	// parse from wire map
	v, err := schema.Parse(ctx, map[string]any{
		"name": "proj",
		"tags": []any{"a", "b"},
	})
	if err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if v.Name != "proj" || len(v.Tags) != 2 || v.Tags[0] != "a" || v.Tags[1] != "b" {
		t.Fatalf("unexpected typed value: %+v", v)
	}

	// validate typed value
	if err := schema.ValidateValue(ctx, v); err != nil {
		t.Fatalf("validate typed value err: %v", err)
	}
}

func TestIdentity_ObjectWithArray_Typed_Roundtrip(t *testing.T) {
	ctx := context.Background()

	type Project struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	schema := g.ObjectTyped[Project]().
		Field("name", g.StringOf[string]()).
		Field("tags", g.ArrayOfSchema[string](g.Array(g.String()).Min(2))).
		Require("name").
		UnknownStrict().
		MustBind()

	// start from typed value (domain)
	in := Project{Name: "proj", Tags: []string{"a", "b"}}

	id := codec.Identity(schema) // Codec[Project, Project]

	decoded, err := id.Decode(ctx, in)
	if err != nil || decoded.Name != "proj" || len(decoded.Tags) != 2 {
		t.Fatalf("decode err=%v v=%+v", err, decoded)
	}
	encoded, err := id.Encode(ctx, decoded)
	if err != nil || encoded.Name != "proj" || len(encoded.Tags) != 2 {
		t.Fatalf("encode err=%v v=%+v", err, encoded)
	}
}
