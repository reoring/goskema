package user

import (
	"context"
	"encoding/json"
	"testing"

	goskema "github.com/reoring/goskema"
	d "github.com/reoring/goskema/dsl"
)

// interpreted schema used for comparison
func interpretedSchema() goskema.Schema[User] {
	b := d.Object().
		Field("name", d.StringOf[string]()).Required().
		Field("active", d.BoolOf[bool]()).Default(true).
		UnknownStrict()
	s, _ := d.Bind[User](b)
	return s
}

func TestCompiled_Equals_Interpreted_Normal(t *testing.T) {
	ctx := context.Background()
	s := interpretedSchema()

	cases := []struct {
		name string
		json string
		want User
	}{
		{"full", `{"name":"Alice","active":false}`, User{Name: "Alice", Active: false}},
		{"default_active", `{"name":"Bob"}`, User{Name: "Bob", Active: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var anyVal any
			json.Unmarshal([]byte(tc.json), &anyVal)
			iv, ierr := s.Parse(ctx, anyVal)
			if ierr != nil {
				t.Fatalf("interpreted err: %v", ierr)
			}
			cv, cerr := ParseFromUser(ctx, goskema.JSONBytes([]byte(tc.json)))
			if cerr != nil {
				t.Fatalf("compiled err: %v", cerr)
			}
			if iv != tc.want {
				t.Fatalf("interpreted got=%+v want=%+v", iv, tc.want)
			}
			if cv != tc.want {
				t.Fatalf("compiled got=%+v want=%+v", cv, tc.want)
			}
		})
	}
}

func TestCompiled_Equals_Interpreted_Errors(t *testing.T) {
	ctx := context.Background()
	s := interpretedSchema()

	cases := []struct {
		name          string
		json          string
		wantFirstCode string
	}{
		{"missing_name", `{"active":true}`, goskema.CodeRequired},
		{"unknown_key", `{"name":"A","extra":1}`, goskema.CodeUnknownKey},
		{"invalid_type_name", `{"name":123}`, goskema.CodeInvalidType},
		{"invalid_type_active", `{"name":"A","active":"x"}`, goskema.CodeInvalidType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// interpreted
			_, ierr := s.Parse(ctx, func() any { var v any; _ = json.Unmarshal([]byte(tc.json), &v); return v }())
			if ierr == nil {
				t.Fatalf("interpreted expected error")
			}
			// compiled
			_, cerr := ParseFromUser(ctx, goskema.JSONBytes([]byte(tc.json)))
			if cerr == nil {
				t.Fatalf("compiled expected error")
			}
			// compare first code if possible
			ic, _ := goskema.AsIssues(ierr)
			cc, _ := goskema.AsIssues(cerr)
			if len(ic) == 0 || len(cc) == 0 {
				t.Fatalf("missing issues: interpreted=%v compiled=%v", ic, cc)
			}
			if ic[0].Code != cc[0].Code {
				t.Fatalf("first issue code mismatch: interp=%s compiled=%s", ic[0].Code, cc[0].Code)
			}
			// and match expected code
			if ic[0].Code != tc.wantFirstCode {
				t.Fatalf("expected first code=%s got=%s", tc.wantFirstCode, ic[0].Code)
			}
		})
	}
}
