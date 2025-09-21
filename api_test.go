package goskema_test

import (
	"context"
	"testing"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// minimalSchema is a stub Schema that echoes input when it's of type string.
type minimalSchema struct{}

func (minimalSchema) Parse(ctx context.Context, v any) (string, error) {
	s, _ := v.(string)
	if s == "" {
		return "", goskema.Issues{goskema.Issue{Code: goskema.CodeInvalidType, Path: "/", Message: "expected string"}}
	}
	return s, nil
}
func (minimalSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[string], error) {
	s, err := (minimalSchema{}).Parse(ctx, v)
	return goskema.Decoded[string]{Value: s, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (minimalSchema) TypeCheck(ctx context.Context, v any) error        { return nil }
func (minimalSchema) RuleCheck(ctx context.Context, v any) error        { return nil }
func (minimalSchema) Validate(ctx context.Context, v any) error         { return nil }
func (minimalSchema) ValidateValue(ctx context.Context, v string) error { return nil }
func (minimalSchema) JSONSchema() (*js.Schema, error)                   { return &js.Schema{}, nil }

func TestParseFrom_DelegatesToSchema(t *testing.T) {
	s := minimalSchema{}
	_, err := goskema.ParseFrom[string](context.Background(), s, goskema.JSONBytes([]byte("dummy")))
	if err == nil {
		t.Fatalf("expected error due to stub Parse expecting string input, got nil")
	}
}

func TestIssues_ErrorSummary(t *testing.T) {
	iss := goskema.Issues{
		{Path: "/a", Code: goskema.CodeInvalidType},
		{Path: "/b", Code: goskema.CodeUnknownKey},
		{Path: "/c", Code: goskema.CodeTooShort},
		{Path: "/d", Code: goskema.CodeTooLong},
	}
	s := iss.Error()
	if s == "" {
		t.Fatalf("expected non-empty error summary")
	}
}
