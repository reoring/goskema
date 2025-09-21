package compare_test

import (
	"context"
	"encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
	jschema "github.com/santhosh-tekuri/jsonschema/v5"
)

// Minimal schema that requires id:string; unknowns allowed
const jsonSchemaUser = `{
  "type": "object",
  "properties": {"id": {"type": "string"}},
  "required": ["id"],
  "additionalProperties": true
}`

// ParseAndValidateSchema: use jsonschema/v5 on small payload.
func Benchmark_ParseAndValidateSchema_jsonschema_v5_Small(b *testing.B) {
	comp := jschema.MustCompileString("mem:user", jsonSchemaUser)
	data := []byte(`{"id":"u_1","name":"alice"}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := comp.Validate(bytesToAny(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// Same condition with goskema schema validation side
func Benchmark_ParseAndValidateSchema_goskema_Small_Object(b *testing.B) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		Build()
	data := []byte(`{"id":"u_1","name":"alice"}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// bytesToAny decodes JSON into any using the stdlib for jsonschema v5 input.
func bytesToAny(b []byte) any {
	var v any
	_ = json.Unmarshal(b, &v)
	return v
}
