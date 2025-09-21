package compare_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"

	sonic "github.com/bytedance/sonic"
	gojson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fastjson"
)

// shared fixtures

func makeUserSchema(tb testing.TB) goskema.Schema[map[string]any] {
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

func smallUserJSON() []byte { return []byte(`{"id":"u_1","name":"alice"}`) }

func generateHugeJSONArray(numObjects int, extraFields int) []byte {
	var buf bytes.Buffer
	buf.Grow(numObjects * (64 + extraFields*16))
	buf.WriteByte('[')
	for i := 0; i < numObjects; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('{')
		buf.WriteString("\"id\":\"obj_")
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString("\",\"name\":\"n")
		buf.WriteString(strconv.Itoa(i))
		buf.WriteString("\",\"age\":")
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte(',')
		if i%2 == 0 {
			buf.WriteString("\"active\":true,")
		} else {
			buf.WriteString("\"active\":false,")
		}
		buf.WriteString("\"meta\":{\"score\":")
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte('}')
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

// ---- Small object ----

// ---- ParseOnly: bytes -> memory structure (no validation) ----

func Benchmark_ParseOnly_stdlib_Small(b *testing.B) {
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

func Benchmark_ParseOnly_gojson_Small(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v map[string]any
		if err := gojson.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_jsoniter_Small(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	var ji = jsoniter.ConfigCompatibleWithStandardLibrary
	for i := 0; i < b.N; i++ {
		var v map[string]any
		if err := ji.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_sonic_Small(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v map[string]any
		if err := sonic.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_fastjson_Small(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p fastjson.Parser
		if _, err := p.ParseBytes(data); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_goskema_Small(b *testing.B) {
	ctx := context.Background()
	// MapOnly: comparable to additionalProperties with minimal type checking.
	s := g.MapAny()
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// ---- ParseAndCheck: minimal check (id:string required) ----

func Benchmark_ParseAndCheck_stdlib_Small(b *testing.B) {
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v map[string]any
		if err := json.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
		id, ok := v["id"].(string)
		if !ok || id == "" {
			b.Fatal("id missing or not string")
		}
	}
}

func Benchmark_ParseAndCheck_goskema_Small(b *testing.B) {
	ctx := context.Background()
	s := makeUserSchema(b)
	data := smallUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

// ---- Huge array ----

const (
	cmpHugeN = 10000
	cmpHugeK = 8
)

func Benchmark_ParseOnly_stdlib_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
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

func Benchmark_ParseOnly_gojson_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []map[string]any
		if err := gojson.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_jsoniter_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	var ji = jsoniter.ConfigCompatibleWithStandardLibrary
	for i := 0; i < b.N; i++ {
		var v []map[string]any
		if err := ji.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_sonic_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var v []map[string]any
		if err := sonic.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_fastjson_HugeArray(b *testing.B) {
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p fastjson.Parser
		if _, err := p.ParseBytes(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Deeply nested JSON generator and cases
func generateDeepNested(depth int) []byte {
	// {"a":{"a":{...{"z":1}...}}}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i := 0; i < depth; i++ {
		buf.WriteString("\"a\":{")
	}
	buf.WriteString("\"z\":1")
	for i := 0; i < depth; i++ {
		buf.WriteByte('}')
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

func Benchmark_ParseOnly_stdlib_DeepNested(b *testing.B) {
	data := generateDeepNested(64)
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

func Benchmark_ParseOnly_goskema_DeepNested(b *testing.B) {
	ctx := context.Background()
	s := g.MapAny()
	data := generateDeepNested(64)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseOnly_goskema_HugeArray(b *testing.B) {
	ctx := context.Background()
	item := g.MapAny()
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseAndCheck_goskema_HugeArray(b *testing.B) {
	ctx := context.Background()
	item := makeUserSchema(b)
	s := g.Array[map[string]any](item)
	data := generateHugeJSONArray(cmpHugeN, cmpHugeK)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}
