package irconv

import (
	"testing"

	g "github.com/reoring/goskema"
	d "github.com/reoring/goskema/dsl"
	ir "github.com/reoring/goskema/internal/ir"
)

func TestToIRFromSchemaDynamic_Object(t *testing.T) {
	b := d.Object().
		Field("name", d.StringOf[string]()).Required().
		Field("active", d.BoolOf[bool]()).Default(true).
		UnknownStrict()
	s, err := b.Build()
	if err != nil {
		t.Fatalf("build schema: %v", err)
	}

	node := ToIRFromSchemaDynamic(s)
	if node == nil {
		t.Fatalf("ToIRFromSchemaDynamic returned nil")
	}
	obj, ok := node.(*ir.Object)
	if !ok {
		t.Fatalf("expected *ir.Object, got %T", node)
	}

	// fields must include both keys
	fields := map[string]bool{}
	for _, f := range obj.Fields {
		fields[f.Name] = true
	}
	if !fields["name"] || !fields["active"] {
		t.Fatalf("fields missing; got=%v", fields)
	}

	// required must contain name
	if _, ok := obj.Required["name"]; !ok {
		t.Errorf("required missing 'name'")
	}

	// UnknownPolicy should reflect UnknownStrict
	if obj.UnknownPolicy != int(g.UnknownStrict) {
		t.Errorf("unknown policy got %d, want %d", obj.UnknownPolicy, int(g.UnknownStrict))
	}

	// UnknownTarget should be empty for UnknownStrict
	if obj.UnknownTarget != "" {
		t.Errorf("unknown target got %q, want empty", obj.UnknownTarget)
	}
}

func TestToIRFromSchemaDynamic_DefaultCaptured(t *testing.T) {
	b := d.Object().
		Field("active", d.BoolOf[bool]()).Default(true)
	s, err := b.Build()
	if err != nil {
		t.Fatalf("build schema: %v", err)
	}
	node := ToIRFromSchemaDynamic(s)
	obj, ok := node.(*ir.Object)
	if !ok {
		t.Fatalf("expected *ir.Object, got %T", node)
	}
	var found bool
	for _, f := range obj.Fields {
		if f.Name == "active" {
			found = true
			bv, ok := f.Default.(bool)
			if !ok || !bv {
				t.Fatalf("default not captured correctly; got=%T %v", f.Default, f.Default)
			}
		}
	}
	if !found {
		t.Fatalf("field 'active' not found")
	}
}
