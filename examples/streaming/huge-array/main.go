package main

import (
	"context"
	"fmt"
	"strings"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// This example demonstrates streaming parse for a huge JSON array.
// - Success path: parse from io.Reader using StreamParse
// - Error path: collect per-index Issues (paths like "/1")
// - Optional: enforce size cap via ParseOpt.MaxBytes
type Item struct {
	ID string `json:"id"`
}

func main() {
	ctx := context.Background()

	// Element schema (typed)
	item := g.ObjectOf[Item]().
		Field("id", g.StringOf[string]()).
		Required().
		UnknownStrip().
		MustBind()

	arr := g.Array[Item](item)

	// 1) Success input (all elements valid)
	good := strings.NewReader(`[{"id":"ok1"},{"id":"ok2"}]`)
	vals, err := goskema.StreamParse(ctx, arr, good)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	fmt.Printf("OK: %d items (first=%+v)\n", len(vals), vals[0])

	// 2) Mixed input: element at index 1 is invalid (id is number)
	bad := strings.NewReader(`[{"id":"ok1"},{"id":1},{"id":"ok2"}]`)
	_, err = goskema.StreamParse(ctx, arr, bad)
	if err != nil {
		fmt.Println("Collect mode => err:", err)
		if iss, ok := goskema.AsIssues(err); ok {
			for _, it := range iss {
				fmt.Printf("- %s at %s: %s\n", it.Code, it.Path, it.Message)
			}
		}
	}

	// 3) Size cap (MaxBytes) causes truncated error when exceeded
	capReader := strings.NewReader(`[{"id":"ok"}]`)
	_, err = goskema.StreamParse(ctx, arr, capReader, goskema.ParseOpt{MaxBytes: 8})
	if err != nil {
		if iss, ok := goskema.AsIssues(err); ok && len(iss) > 0 {
			fmt.Println("MaxBytes =>", iss[0].Code)
		}
	}
}
