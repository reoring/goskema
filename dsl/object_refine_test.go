package dsl_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestObjectBuilder_Refine_PasswordConfirm(t *testing.T) {
	ctx := context.Background()

	s, err := g.Object().
		Field("email", g.StringOf[string]()).
		Field("password", g.StringOf[string]()).
		Field("confirm", g.StringOf[string]()).
		Require("email", "password", "confirm").
		UnknownStrict().
		Refine("password==confirm", func(ctx context.Context, v map[string]any) error {
			pw, _ := v["password"].(string)
			cf, _ := v["confirm"].(string)
			if pw != cf {
				return goskema.Issues{{Path: "/confirm", Code: "custom", Message: "password mismatch"}}
			}
			return nil
		}).
		Build()
	if err != nil {
		t.Fatalf("unexpected build err: %v", err)
	}

	// ng: mismatch
	if _, err := s.Parse(ctx, map[string]any{"email": "a@b", "password": "x", "confirm": "y"}); err == nil {
		t.Fatalf("expected refine custom error for mismatch")
	}

	// ok: match
	if _, err := s.Parse(ctx, map[string]any{"email": "a@b", "password": "x", "confirm": "x"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}
