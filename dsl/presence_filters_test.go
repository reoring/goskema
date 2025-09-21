package dsl_test

import (
	"context"
	"testing"
	"unsafe"

	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

func TestPresence_IncludeExcludeIntern_Basics(t *testing.T) {
	ctx := context.Background()

	// schema: { id: string, name: string, meta: { note: string } }
	meta, _ := g.Object().
		Field("note", g.StringOf[string]()).
		Build()
	s, _ := g.Object().
		Field("id", g.StringOf[string]()).
		Field("name", g.StringOf[string]()).
		Field("meta", g.SchemaOf(meta)).
		Require("id").
		UnknownStrict().
		Build()

	js := []byte(`{"id":"u1","name":"Reo","meta":{"note":"hello"}}`)

	// Collect only under /meta and intern keys
	opt := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true, Include: []string{"/meta"}},
		PathRender: goskema.PathRenderOpt{Intern: true},
	}
	dm, err := goskema.ParseFromWithMeta(context.Background(), s, goskema.JSONBytes(js), opt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Expect keys only under /meta
	keys := make([]string, 0, len(dm.Presence))
	for k := range dm.Presence {
		keys = append(keys, k)
	}

	// Should include "/meta" and "/meta/note" (ordering not guaranteed)
	has := func(w string) bool {
		for _, k := range keys {
			if k == w {
				return true
			}
		}
		return false
	}
	if !has("/meta") || !has("/meta/note") {
		t.Fatalf("presence include failed, keys=%v", keys)
	}
	if has("/") || has("/id") || has("/name") {
		t.Fatalf("presence include leaked non-meta keys: %v", keys)
	}

	// Exclude /meta/note and verify only /meta remains
	opt2 := goskema.ParseOpt{
		Presence:   goskema.PresenceOpt{Collect: true, Include: []string{"/meta"}, Exclude: []string{"/meta/note"}},
		PathRender: goskema.PathRenderOpt{Intern: true},
	}
	dm2, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(js), opt2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(dm2.Presence) != 1 {
		t.Fatalf("expected only one key after exclude, got: %v", dm2.Presence)
	}
	if _, ok := dm2.Presence["/meta"]; !ok {
		t.Fatalf("expected /meta key present, got: %v", dm2.Presence)
	}

	// Intern behavior: keys should be interned; two maps created with Intern:true should share key pointers
	// Build another map with same options and compare pointers via reflect.StringHeader
	dm3, err := goskema.ParseFromWithMeta(ctx, s, goskema.JSONBytes(js), opt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Pick a common key
	k1 := ""
	for k := range dm.Presence {
		if hasMeta(k) {
			k1 = k
			break
		}
	}
	if k1 == "" {
		t.Fatalf("no meta key found in presence: %v", dm.Presence)
	}
	k2 := ""
	for k := range dm3.Presence {
		if k == k1 {
			k2 = k
			break
		}
	}
	if k2 == "" {
		t.Fatalf("matching key not found in second presence map")
	}

	// Check if the backing string pointers are equal (interned)
	if !sameStringPointer(k1, k2) {
		t.Fatalf("expected interned keys to share backing pointers: %q vs %q", k1, k2)
	}
}

func hasMeta(k string) bool { return k == "/meta" || k == "/meta/note" }

// sameStringPointer compares two Go strings' backing data pointers.
func sameStringPointer(a, b string) bool {
	// Go 1.21+ discourages reflect.StringHeader; use unsafe.StringData to compare
	// the backing pointers instead.
	return unsafe.StringData(a) == unsafe.StringData(b)
}
