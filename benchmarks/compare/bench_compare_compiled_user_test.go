package compare_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	d "github.com/reoring/goskema/dsl"
	u "github.com/reoring/goskema/examples/user"
)

// interpreted typed schema for examples/user.User
func interpretedTypedUser(tb testing.TB) goskema.Schema[u.User] {
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

func compiledUserJSON() []byte { return []byte(`{"name":"Alice","active":true}`) }

func Benchmark_Compiled_User_Small(b *testing.B) {
	ctx := context.Background()
	data := compiledUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := u.ParseFromUser(ctx, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Interpreted_User_Small(b *testing.B) {
	ctx := context.Background()
	s := interpretedTypedUser(b)
	data := compiledUserJSON()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := goskema.ParseFrom(ctx, s, goskema.JSONBytes(data)); err != nil {
			b.Fatal(err)
		}
	}
}
