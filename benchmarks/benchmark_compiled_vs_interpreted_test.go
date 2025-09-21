package goskema_test

import (
	"bytes"
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	d "github.com/reoring/goskema/dsl"
	u "github.com/reoring/goskema/examples/user"
)

// --- Fixtures for compiled vs interpreted (examples/user) ---

func smallUserJSONCompiled() []byte { return []byte(`{"name":"Alice","active":true}`) }

func interpretedUserSchema(tb testing.TB) goskema.Schema[u.User] {
	tb.Helper()
	s, err := d.ObjectOf[u.User]().
		Field("name", d.StringOf[string]()).
		Field("active", d.BoolOf[bool]()).
		Require("name").
		UnknownStrict().
		Bind()
	if err != nil {
		tb.Fatalf("bind schema: %v", err)
	}
	return s
}

// --- Compiled ---

func Benchmark_Compiled_User_Small_JSONBytes(b *testing.B) {
	ctx := context.Background()
	data := smallUserJSONCompiled()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := u.ParseFromUser(ctx, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Compiled_User_Small_JSONReader(b *testing.B) {
	ctx := context.Background()
	data := smallUserJSONCompiled()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		if _, err := u.ParseFromUser(ctx, goskema.JSONReader(r)); err != nil {
			b.Fatal(err)
		}
	}
}

// --- Interpreted ---

func Benchmark_Interpreted_User_Small_JSONBytes(b *testing.B) {
	ctx := context.Background()
	s := interpretedUserSchema(b)
	data := smallUserJSONCompiled()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Interpreted_User_Small_JSONReader(b *testing.B) {
	ctx := context.Background()
	s := interpretedUserSchema(b)
	data := smallUserJSONCompiled()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONReader(r)); err != nil {
			b.Fatal(err)
		}
	}
}
