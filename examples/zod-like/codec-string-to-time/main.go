package main

import (
	"context"
	"fmt"
	"time"

	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

func main() {
	ctx := context.Background()

	// Equivalent to Zod: z.codec(z.iso.datetime(), z.date(), {...})
	// goskema: wrap a Codec as a Schema (wire: string <-> domain: time.Time)
	c := codec.TimeRFC3339()
	s := g.Codec[string, time.Time](c)

	// decode: wire(string) -> domain(time.Time)
	t, err := s.Parse(ctx, "2024-01-15T10:30:00Z")
	fmt.Println("decode(parse):", t, err)

	// encode: domain(time.Time) -> wire(string)
	// In goskema the Codec itself exposes Encode
	encoded, err := c.Encode(ctx, t)
	fmt.Println("encode:", encoded, err)

	// async/safe variants are delegated to the codec implementation (synchronous example here)
}
