package dsl_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// 2^53 + 1 cannot be represented exactly as float64 (it will be rounded).
const largeIntBeyondFloat64 = 9007199254740993 // 2^53 + 1

func TestParseFrom_NumberMode_JSONNumber_PreservesPrecision(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("n", g.SchemaOf(g.NumberJSON())).
		Require("n").
		UnknownStrict().
		Build()

	js := []byte(`{"n":` + strconv.FormatInt(largeIntBeyondFloat64, 10) + `}`)
	// Default mode is JSONNumber
	v, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(js))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := v["n"].(json.Number)
	if got.String() != strconv.FormatInt(largeIntBeyondFloat64, 10) {
		t.Fatalf("json.Number should preserve exact string; got=%s", got.String())
	}
}

func TestParseFrom_NumberMode_Float64_RoundsValue(t *testing.T) {
	ctx := context.Background()
	s, _ := g.Object().
		Field("n", g.SchemaOf(g.NumberJSON())).
		Require("n").
		UnknownStrict().
		Build()

	js := []byte(`{"n":` + strconv.FormatInt(largeIntBeyondFloat64, 10) + `}`)
	// Explicitly request Float64 mode
	src := goskema.WithNumberMode(goskema.JSONBytes(js), goskema.NumberFloat64)
	v, err := goskema.ParseFrom(ctx, s, src)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := v["n"].(json.Number).String()
	// Because it goes through float64, verify it no longer matches 9007199254740993 (rounded or exponent form).
	if got == strconv.FormatInt(largeIntBeyondFloat64, 10) {
		t.Fatalf("expected rounded/float64-formatted value, but preserved exact string: %s", got)
	}
}
