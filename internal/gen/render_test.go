package gen

import (
	"testing"

	ir "github.com/reoring/goskema/internal/ir"
)

func TestRenderFile_Minimal(t *testing.T) {
	out, err := RenderFile(File{Package: "foo", Types: []TypeStub{{Name: "User"}}})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("empty output")
	}
}

func TestRenderFileFromIR_ObjectSkeleton(t *testing.T) {
	out, err := RenderFileFromIR("foo", []TypeDef{{Name: "User", Fields: []string{"id", "name"}}})
	if err != nil {
		t.Fatalf("render IR failed: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("empty output")
	}
	// smoke only; detailed assertions when wiring IR pipeline
}

func TestRenderFileFromIRNodes_Object(t *testing.T) {
	obj := &ir.Object{
		Fields:        []ir.Field{{Name: "b"}, {Name: "a"}},
		Required:      map[string]struct{}{"a": {}, "b": {}},
		UnknownPolicy: 1,
		UnknownTarget: "extra",
	}
	code, err := RenderFileFromIRNodes("foo", map[string]ir.Schema{"User": obj})
	if err != nil {
		t.Fatalf("render from IR nodes failed: %v", err)
	}
	if len(code) == 0 {
		t.Fatalf("empty output")
	}
}
