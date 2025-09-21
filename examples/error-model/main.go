package main

import (
	"context"
	"errors"
	"fmt"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// This sample demonstrates goskema's error model.
// - Collect mode (default) aggregates multiple issues.
// - Fail-Fast mode stops at the first issue.
// - Use errors.As and AsIssues helpers to inspect details.
func main() {
	ctx := context.Background()

	// Schema: id/email are required strings, unknown keys are errors
	user, _ := g.Object().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		UnknownStrict().
		Build()

	// Input: missing id, numeric email, includes unknown key zzz
	js := []byte(`{"email": 1, "zzz": true}`)

	// 1) Collect (default): aggregate multiple issues
	_, err := goskema.ParseFrom(ctx, user, goskema.JSONBytes(js))
	if err != nil {
		fmt.Println("Collect mode => err:", err)
		// Extract details using errors.As
		var iss goskema.Issues
		if errors.As(err, &iss) {
			fmt.Println("Collect mode issues:")
			for _, it := range iss {
				fmt.Printf("- %s at %s: %s\n", it.Code, it.Path, it.Message)
			}
		}
	}

	// 2) Fail-Fast: stop at the first failure
	_, err = goskema.ParseFrom(ctx, user, goskema.JSONBytes(js), goskema.ParseOpt{FailFast: true})
	if iss, ok := goskema.AsIssues(err); ok {
		fmt.Println("Fail-Fast mode issues (first only):")
		for _, it := range iss {
			fmt.Printf("- %s at %s: %s\n", it.Code, it.Path, it.Message)
		}
	}
}
