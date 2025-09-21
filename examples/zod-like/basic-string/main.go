package main

import (
	"context"
	"fmt"

	g "github.com/reoring/goskema/dsl"
)

func main() {
	ctx := context.Background()

	// Zod: const schema = z.string();
	// goskema: string schema
	s := g.String()

	// Zod: schema.parse("asdf") => "asdf"
	v, err := s.Parse(ctx, "asdf")
	fmt.Println("parse:", v, err)

	// Zod: schema.decode("asdf") => "asdf"
	// goskema: Decode goes through a Codec; primitives match Parse so omitted.

	// Zod: schema.encode("asdf") => "asdf"
	// goskema: Encode routes through a Codec; here we only show the identical transformation.
	enc := v
	fmt.Println("encode:", enc)

	// Types: input and output are both string (akin to Zod's Input/Output)
	_ = v // string
}
