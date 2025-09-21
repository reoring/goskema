package main

import (
	"context"
	"fmt"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

func main() {
	ctx := context.Background()

	// Decode wrapper: apply a Schema in the forward direction (unknown -> T).
	s := g.String()
	sv, err := goskema.Decode(ctx, s, "hello")
	fmt.Println("Decode(Schema):", sv, err)

	// Encode wrapper: apply a Codec in the reverse direction (domain -> wire).
	c := codec.TimeRFC3339()
	cd := g.Codec(c)
	t, _ := cd.Parse(ctx, "2025-01-01T00:00:00Z")
	wire, err := goskema.Encode(ctx, c, t)
	fmt.Println("Encode(Codec):", wire, err)
}
