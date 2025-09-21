package goskema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// Micro: small object with numeric fields
func numberModeSmallSchema(tb testing.TB) goskema.Schema[map[string]any] {
	tb.Helper()
	s, err := g.Object().
		Field("a", g.SchemaOf[json.Number](g.NumberJSON())).
		Field("b", g.SchemaOf[json.Number](g.NumberJSON())).
		Field("c", g.SchemaOf[json.Number](g.NumberJSON())).
		UnknownStrip().
		Build()
	if err != nil {
		tb.Fatalf("schema build failed: %v", err)
	}
	return s
}

func Benchmark_NumberMode_Small_JSONNumber(b *testing.B) {
	ctx := context.Background()
	s := numberModeSmallSchema(b)
	data := []byte(`{"a":1,"b":2.5,"c":-3.75}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_NumberMode_Small_Float64(b *testing.B) {
	ctx := context.Background()
	s := numberModeSmallSchema(b)
	data := []byte(`{"a":1,"b":2.5,"c":-3.75}`)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.WithNumberMode(goskema.JSONBytes(data), goskema.NumberFloat64)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

// Macro: huge array of small numeric objects
func numberModeHugeItemSchema(tb testing.TB) goskema.Schema[map[string]any] {
	tb.Helper()
	s, err := g.Object().
		Field("x", g.SchemaOf[json.Number](g.NumberJSON())).
		Field("y", g.SchemaOf[json.Number](g.NumberJSON())).
		Field("z", g.SchemaOf[json.Number](g.NumberJSON())).
		UnknownStrip().
		Build()
	if err != nil {
		tb.Fatalf("schema build failed: %v", err)
	}
	return s
}

func generateNumericJSONArray(num int) []byte {
	var buf bytes.Buffer
	buf.Grow(num * 48)
	buf.WriteByte('[')
	for i := 0; i < num; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		// oscillate values to avoid trivial constant folding
		buf.WriteString(`{"x":`)
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString(`,"y":`)
		if i%2 == 0 {
			buf.WriteString("1.5")
		} else {
			buf.WriteString("2.5")
		}
		buf.WriteString(`,"z":-3.75}`)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

const numberModeHugeN = 50000

func Benchmark_NumberMode_HugeArray_JSONNumber(b *testing.B) {
	ctx := context.Background()
	item := numberModeHugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateNumericJSONArray(numberModeHugeN)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_NumberMode_HugeArray_Float64(b *testing.B) {
	ctx := context.Background()
	item := numberModeHugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateNumericJSONArray(numberModeHugeN)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.WithNumberMode(goskema.JSONBytes(data), goskema.NumberFloat64)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}
