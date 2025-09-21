package goskema_test

import (
	"context"
	"errors"
	"testing"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

func TestEncodeWithMode_Canonical(t *testing.T) {
	ctx := context.Background()
	c := codec.Identity[string](g.String())
	out, err := goskema.EncodeWithMode(ctx, c, "alice", goskema.EncodeCanonical)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "alice" {
		t.Fatalf("want alice, got %q", out)
	}
}

func TestEncodeWithMode_PreserveRequiresPresence(t *testing.T) {
	ctx := context.Background()
	c := codec.Identity[string](g.String())
	_, err := goskema.EncodeWithMode(ctx, c, "alice", goskema.EncodePreserve)
	if !errors.Is(err, goskema.ErrEncodePreserveRequiresPresence) {
		t.Fatalf("expected ErrEncodePreserveRequiresPresence, got: %v", err)
	}
}
