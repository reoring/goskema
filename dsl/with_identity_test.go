package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestWithIdentity_String_Parse_Decode_Encode(t *testing.T) {
	ctx := context.Background()

	// Sugar mirroring schema := z.string(); schema.decode/encode style usage.
	schema := g.WithIdentity(g.String())

	// parse("asdf") => "asdf"
	v, err := schema.Parse(ctx, "asdf")
	if err != nil || v != "asdf" {
		t.Fatalf("parse err=%v v=%q", err, v)
	}

	// decode/encode sugar (equivalent to an identity Codec API)
	dv, err := schema.Decode(ctx, "asdf")
	if err != nil || dv != "asdf" {
		t.Fatalf("decode err=%v v=%q", err, dv)
	}
	ev, err := schema.Encode(ctx, dv)
	if err != nil || ev != "asdf" {
		t.Fatalf("encode err=%v v=%q", err, ev)
	}
}

func TestWithIdentity_WithMeta_Decode_EncodePreserving(t *testing.T) {
	ctx := context.Background()
	schema := g.WithIdentity(g.String())

	dm, err := schema.DecodeWithMeta(ctx, "x")
	if err != nil || dm.Value != "x" {
		t.Fatalf("decodeWithMeta err=%v v=%v", err, dm.Value)
	}
	if dm.Presence == nil || dm.Presence["/"]&goskema.PresenceSeen == 0 {
		t.Fatalf("expected presence seen at root")
	}

	out, err := schema.EncodePreserving(ctx, dm)
	if err != nil || out != "x" {
		t.Fatalf("encodePreserving err=%v v=%v", err, out)
	}
}

func TestWithIdentity_ObjectTyped_Roundtrip(t *testing.T) {
	ctx := context.Background()

	type Project struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	base := g.ObjectTyped[Project]().
		Field("name", g.StringOf[string]()).
		Field("tags", g.ArrayOfSchema(g.Array(g.String()).Min(2))).
		Require("name").
		UnknownStrict().
		MustBind()

	schema := g.WithIdentity(base)

	// start from typed value (domain)
	in := Project{Name: "proj", Tags: []string{"a", "b"}}

	decoded, err := schema.Decode(ctx, in)
	if err != nil || decoded.Name != "proj" || len(decoded.Tags) != 2 {
		t.Fatalf("decode err=%v v=%+v", err, decoded)
	}
	encoded, err := schema.Encode(ctx, decoded)
	if err != nil || encoded.Name != "proj" || len(encoded.Tags) != 2 {
		t.Fatalf("encode err=%v v=%+v", err, encoded)
	}
}
