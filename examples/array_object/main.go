package main

import (
	"context"
	"fmt"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func main() {
	ctx := context.Background()

	// Element object schema (minimal example)
	item, _ := g.Object().
		Field("id", g.StringOf[string]()).Required().
		Field("name", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	// Assign an array field with Min/Max constraints
	ab := g.Array(item).Min(1).Max(10)
	schema, _ := g.Object().
		Field("items", g.ArrayOfSchema[map[string]any](ab)).
		UnknownStrict().
		Build()

	// Example input
	js := []byte(`{"items":[{"id":"1","name":"A"},{"id":"2","name":"B"}]}`)
	v, err := goskema.ParseFrom(ctx, schema, goskema.JSONBytes(js))
	if err != nil {
		fmt.Println("ERR:", err)
		return
	}
	fmt.Println("OK items_len=", len(v["items"].([]map[string]any)))
}
