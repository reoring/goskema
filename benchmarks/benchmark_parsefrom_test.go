package goskema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// ---- Helpers ----

func smallUserSchemaStrict(tb testing.TB) goskema.Schema[map[string]any] {
	tb.Helper()
	s, err := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id").
		UnknownStrict().
		Build()
	if err != nil {
		tb.Fatalf("schema build failed: %v", err)
	}
	return s
}

func smallUserSchemaStrip(tb testing.TB) goskema.Schema[map[string]any] {
	tb.Helper()
	s, err := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		Build()
	if err != nil {
		tb.Fatalf("schema build failed: %v", err)
	}
	return s
}

func smallUserJSON() []byte {
	return []byte(`{"id":"u_1","name":"alice"}`)
}

// generateHugeJSONArray returns a JSON array of objects of the form:
// [{"id":"obj_0","name":"n0","age":0,"active":true,"meta":{"score":0},"k0":"v0",...}, ...]
func generateHugeJSONArray(numObjects int, extraFields int) []byte {
	var buf bytes.Buffer
	buf.Grow(numObjects * (64 + extraFields*16))
	buf.WriteByte('[')
	for i := 0; i < numObjects; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		// stable core fields
		fmt.Fprintf(&buf, "\"id\":\"obj_%d\",", i)
		fmt.Fprintf(&buf, "\"name\":\"n%d\",", i)
		fmt.Fprintf(&buf, "\"age\":%d,", i)
		if i%2 == 0 {
			buf.WriteString("\"active\":true,")
		} else {
			buf.WriteString("\"active\":false,")
		}
		fmt.Fprintf(&buf, "\"meta\":{\"score\":%d}", i)
		// extras
		for k := 0; k < extraFields; k++ {
			buf.WriteByte(',')
			buf.WriteByte('"')
			buf.WriteString("k")
			buf.WriteString(strconv.Itoa(k))
			buf.WriteString("\":\"v")
			buf.WriteString(strconv.Itoa(i))
			buf.WriteString("_")
			buf.WriteString(strconv.Itoa(k))
			buf.WriteString("\"")
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.Bytes()
}

// schema for huge array: only requires id; strips unknown keys for throughput-oriented parsing
func hugeItemSchema(tb testing.TB) goskema.Schema[map[string]any] {
	tb.Helper()
	s, err := g.Object().
		Field("id", g.StringOf[string]()).
		Require("id").
		UnknownStrip().
		Build()
	if err != nil {
		tb.Fatalf("schema build failed: %v", err)
	}
	return s
}

// ---- Micro benchmarks (small inputs) ----

func Benchmark_ParseFrom_Object_Small_JSONBytes(b *testing.B) {
	ctx := context.Background()
	s := smallUserSchemaStrict(b)
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.JSONBytes(data)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseFrom_Object_Small_JSONReader(b *testing.B) {
	ctx := context.Background()
	s := smallUserSchemaStrict(b)
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		src := goskema.JSONReader(r)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseFromWithMeta_Object_Small(b *testing.B) {
	ctx := context.Background()
	s := smallUserSchemaStrict(b)
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.JSONBytes(data)
		if _, err := goskema.ParseFromWithMeta(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

// Array micro: ["a","b","c"]
func Benchmark_ParseFrom_Array_String_Small(b *testing.B) {
	ctx := context.Background()
	s := g.Array[string](g.String())
	data := []byte("[\"a\",\"b\",\"c\"]")
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.JSONBytes(data)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

// ---- Macro benchmarks (huge JSON) ----

// 10k objects with 8 extra fields each ~ O(10-20MB) depending on numbers
const (
	hugeObjects   = 10000
	hugeExtraKeys = 8
)

func Benchmark_ParseFrom_HugeArray_Objects_JSONBytes(b *testing.B) {
	ctx := context.Background()
	item := hugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.JSONBytes(data)
		if _, err := goskema.ParseFrom(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseFrom_HugeArray_Objects_NumberFloat64(b *testing.B) {
	ctx := context.Background()
	item := hugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
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

func Benchmark_ParseFromWithMeta_HugeArray_Objects(b *testing.B) {
	ctx := context.Background()
	item := hugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := goskema.JSONBytes(data)
		if _, err := goskema.ParseFromWithMeta(ctx, s, src); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_StreamParse_HugeArray_Objects(b *testing.B) {
	ctx := context.Background()
	item := hugeItemSchema(b)
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		if _, err := goskema.StreamParse(ctx, s, r); err != nil {
			b.Fatal(err)
		}
	}
}

// ---- Baseline: encoding/json ----

func Benchmark_encodingJSON_Unmarshal_SmallObject(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_encodingJSON_Unmarshal_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_encodingJSON_Decoder_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(hugeObjects, hugeExtraKeys)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []map[string]any
		dec := json.NewDecoder(bytes.NewReader(data))
		if err := dec.Decode(&v); err != nil && err != io.EOF {
			b.Fatal(err)
		}
	}
}
