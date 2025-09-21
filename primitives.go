package goskema

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

// Deprecated: prefer dsl.String(). This function may be removed in the future.
// NewStringSchema returns the minimal String schema implementation.
func NewStringSchema() Schema[string] { return stringSchema{} }

// Deprecated: prefer dsl.Bool(). This function may be removed in the future.
// NewBoolSchema returns the minimal Bool schema implementation.
func NewBoolSchema() Schema[bool] { return boolSchema{} }

// Deprecated: prefer dsl.NumberJSON(). This function may be removed in the future.
// NewNumberJSONSchema returns the minimal Number (JSONNumber) schema implementation.
func NewNumberJSONSchema() Schema[json.Number] { return numberJSONSchema{} }

type stringSchema struct{}

func (stringSchema) Parse(ctx context.Context, v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
	// Normalize -> ValidateValue -> Refine
	ns, err := ApplyNormalize[string](ctx, s, stringSchema{})
	if err != nil {
		return "", err
	}
	s = ns
	if err := (stringSchema{}).ValidateValue(ctx, s); err != nil {
		return "", err
	}
	if err := ApplyRefine[string](ctx, s, stringSchema{}); err != nil {
		return "", err
	}
	return s, nil
}

func (stringSchema) ParseWithMeta(ctx context.Context, v any) (Decoded[string], error) {
	s, err := (stringSchema{}).Parse(ctx, v)
	return Decoded[string]{Value: s, Presence: PresenceMap{"/": PresenceSeen}}, err
}

func (stringSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(string); !ok {
		return Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
	return nil
}

func (stringSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (stringSchema) Validate(ctx context.Context, v any) error {
	if err := (stringSchema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (stringSchema{}).RuleCheck(ctx, v)
}

func (stringSchema) ValidateValue(ctx context.Context, v string) error { return nil }

func (stringSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }

type boolSchema struct{}

func (boolSchema) Parse(ctx context.Context, v any) (bool, error) {
	b, ok := v.(bool)
	if !ok {
		return false, Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
	nb, err := ApplyNormalize[bool](ctx, b, boolSchema{})
	if err != nil {
		return false, err
	}
	b = nb
	if err := (boolSchema{}).ValidateValue(ctx, b); err != nil {
		return false, err
	}
	if err := ApplyRefine[bool](ctx, b, boolSchema{}); err != nil {
		return false, err
	}
	return b, nil
}

func (boolSchema) ParseWithMeta(ctx context.Context, v any) (Decoded[bool], error) {
	b, err := (boolSchema{}).Parse(ctx, v)
	return Decoded[bool]{Value: b, Presence: PresenceMap{"/": PresenceSeen}}, err
}

func (boolSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(bool); !ok {
		return Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
	return nil
}

func (boolSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (boolSchema) Validate(ctx context.Context, v any) error {
	if err := (boolSchema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (boolSchema{}).RuleCheck(ctx, v)
}

func (boolSchema) ValidateValue(ctx context.Context, v bool) error { return nil }

func (boolSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }

type numberJSONSchema struct{}

func (numberJSONSchema) Parse(ctx context.Context, v any) (json.Number, error) {
	switch n := v.(type) {
	case json.Number:
		num := n
		nn, err := ApplyNormalize[json.Number](ctx, num, numberJSONSchema{})
		if err != nil {
			return json.Number(""), err
		}
		num = nn
		if err := (numberJSONSchema{}).ValidateValue(ctx, num); err != nil {
			return json.Number(""), err
		}
		if err := ApplyRefine[json.Number](ctx, num, numberJSONSchema{}); err != nil {
			return json.Number(""), err
		}
		return num, nil
	case float64:
		return json.Number(strconv.FormatFloat(n, 'g', -1, 64)), nil
	default:
		return json.Number(""), Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
}

func (numberJSONSchema) ParseWithMeta(ctx context.Context, v any) (Decoded[json.Number], error) {
	n, err := (numberJSONSchema{}).Parse(ctx, v)
	return Decoded[json.Number]{Value: n, Presence: PresenceMap{"/": PresenceSeen}}, err
}

func (numberJSONSchema) TypeCheck(ctx context.Context, v any) error {
	switch v.(type) {
	case json.Number, float64:
		return nil
	default:
		return Issues{{Path: "/", Code: CodeInvalidType, Message: i18n.T(CodeInvalidType, nil)}}
	}
}

func (numberJSONSchema) RuleCheck(ctx context.Context, v any) error { return nil }

func (numberJSONSchema) Validate(ctx context.Context, v any) error {
	if err := (numberJSONSchema{}).TypeCheck(ctx, v); err != nil {
		return err
	}
	return (numberJSONSchema{}).RuleCheck(ctx, v)
}

func (numberJSONSchema) ValidateValue(ctx context.Context, v json.Number) error { return nil }

func (numberJSONSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }

// strconvFormatFloat renders a float64 using the shortest JSON-compatible representation.
func strconvFormatFloat(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }
