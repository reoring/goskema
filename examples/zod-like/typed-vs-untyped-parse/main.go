package main

import (
	"context"
	"fmt"
	"time"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

// This example mirrors the Zod-style parse/decode/encode flow and illustrates
// how to express it with goskema.
func main() {
	ctx := context.Background()

	// 1) Untyped: Parse accepts unknown input and returns the schema's output type.
	s := g.String()
	v, err := s.Parse(ctx, "hello")
	fmt.Println("untyped parse:", v, err)

	// 2) Typed: use a Codec for safe bidirectional conversions between wire and domain.
	tc := codec.TimeRFC3339()
	cd := g.Codec(tc)
	timeVal, err := cd.Parse(ctx, "2025-01-01T00:00:00Z") // decode
	fmt.Println("typed decode:", timeVal, err)

	// encode (domain -> wire)
	wire, err := tc.Encode(ctx, timeVal)
	fmt.Println("typed encode:", wire, err)

	// 3) Embed the codec inside an object schema for an end-to-end workflow.
	payload, _ := g.Object().
		Field("startDate", g.SchemaOf[time.Time](cd)).
		Field("name", g.StringOf[string]()).
		Require("startDate", "name").
		UnknownStrict().
		Build()
	in := []byte(`{"startDate":"2025-01-01T00:00:00Z","name":"kickoff"}`)
	m, err := goskema.ParseFrom(ctx, payload, goskema.JSONBytes(in))
	fmt.Println("object decoded:", m["startDate"].(time.Time).Format(time.RFC3339), m["name"], err)
}
