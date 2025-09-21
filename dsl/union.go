package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

// unionSchema is a minimal discriminated union schema over map[string]any objects.
type unionSchema struct {
	discriminator string
	mapping       map[string]goskema.Schema[map[string]any]
}

func (u *unionSchema) Parse(ctx context.Context, v any) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	dv := m[u.discriminator]
	tag, _ := dv.(string)
	if tag == "" {
		return nil, goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorMissing, Message: i18n.T(goskema.CodeDiscriminatorMissing, nil), Hint: "discriminator missing"}}
	}
	s, ok := u.mapping[tag]
	if !ok {
		return nil, goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorUnknown, Message: i18n.T(goskema.CodeDiscriminatorUnknown, nil), Hint: "unknown variant: '" + tag + "'"}}
	}
	return s.Parse(ctx, v)
}

func (u *unionSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]any], error) {
	m, ok := v.(map[string]any)
	if !ok {
		var zero goskema.Decoded[map[string]any]
		return zero, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	dv := m[u.discriminator]
	tag, _ := dv.(string)
	if tag == "" {
		var zero goskema.Decoded[map[string]any]
		return zero, goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorMissing, Message: i18n.T(goskema.CodeDiscriminatorMissing, nil), Hint: "discriminator missing"}}
	}
	s, ok := u.mapping[tag]
	if !ok {
		var zero goskema.Decoded[map[string]any]
		return zero, goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorUnknown, Message: i18n.T(goskema.CodeDiscriminatorUnknown, nil), Hint: "unknown variant: '" + tag + "'"}}
	}
	return s.ParseWithMeta(ctx, v)
}

func (u *unionSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(map[string]any); !ok {
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	return nil
}

func (u *unionSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (u *unionSchema) Validate(ctx context.Context, v any) error {
	if err := u.TypeCheck(ctx, v); err != nil {
		return err
	}
	return u.RuleCheck(ctx, v)
}

func (u *unionSchema) ValidateValue(ctx context.Context, v map[string]any) error {
	// delegate to selected variant based on discriminator if present
	dv := v[u.discriminator]
	tag, _ := dv.(string)
	if tag == "" {
		return goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorMissing, Message: i18n.T(goskema.CodeDiscriminatorMissing, nil), Hint: "discriminator missing"}}
	}
	s, ok := u.mapping[tag]
	if !ok {
		return goskema.Issues{goskema.Issue{Path: "/" + u.discriminator, Code: goskema.CodeDiscriminatorUnknown, Message: i18n.T(goskema.CodeDiscriminatorUnknown, nil), Hint: "unknown variant: '" + tag + "'"}}
	}
	return s.ValidateValue(ctx, v)
}

func (u *unionSchema) JSONSchema() (*js.Schema, error) {
	// oneOf with variant schemas; discriminator field documented implicitly
	out := &js.Schema{}
	out.OneOf = make([]*js.Schema, 0, len(u.mapping))
	for _, s := range u.mapping {
		vs, err := s.JSONSchema()
		if err != nil {
			return nil, err
		}
		out.OneOf = append(out.OneOf, vs)
	}
	return out, nil
}
