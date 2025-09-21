package codec

import (
	"context"
	"time"

	goskema "github.com/reoring/goskema"
	js "github.com/reoring/goskema/jsonschema"
)

// TimeRFC3339 returns a Codec that converts between RFC3339 strings and time.Time.
func TimeRFC3339() goskema.Codec[string, time.Time] {
	return &rfc3339Codec{
		in:  stringSchema{},
		out: timeSchema{},
	}
}

type rfc3339Codec struct {
	in  goskema.Schema[string]
	out goskema.Schema[time.Time]
}

func (c *rfc3339Codec) In() goskema.Schema[string]     { return c.in }
func (c *rfc3339Codec) Out() goskema.Schema[time.Time] { return c.out }

func (c *rfc3339Codec) Decode(ctx context.Context, a string) (time.Time, error) {
	// wire(string) -> domain(time.Time) -> Out.ValidateValue
	t, err := parseRFC3339(a)
	if err != nil {
		return time.Time{}, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidFormat, Message: "invalid RFC3339 time", Cause: err}}
	}
	if err := c.out.ValidateValue(ctx, t); err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (c *rfc3339Codec) Encode(ctx context.Context, b time.Time) (string, error) {
	// Validate using Out, convert to wire(string), then re-validate via In.Parse
	if err := c.out.ValidateValue(ctx, b); err != nil {
		return "", err
	}
	s := formatRFC3339Canonical(b)
	if _, err := c.in.Parse(ctx, s); err != nil {
		return "", err
	}
	return s, nil
}

func (c *rfc3339Codec) DecodeWithMeta(ctx context.Context, a string) (goskema.Decoded[time.Time], error) {
	t, err := c.Decode(ctx, a)
	return goskema.Decoded[time.Time]{Value: t, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (c *rfc3339Codec) EncodePreserving(ctx context.Context, db goskema.Decoded[time.Time]) (string, error) {
	// Respect presence metadata. Top-level scalars cannot represent null/missing,
	// so treat those cases as errors.
	if db.Presence != nil {
		if p, ok := db.Presence["/"]; ok {
			if p&goskema.PresenceWasNull != 0 {
				return "", goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "cannot encode null as RFC3339 string"}}
			}
			if p&goskema.PresenceSeen == 0 {
				return "", goskema.Issues{{Path: "/", Code: goskema.CodeRequired, Message: "missing value (preserving)"}}
			}
		}
	}
	return c.Encode(ctx, db.Value)
}

// ---- helpers ----

type stringSchema struct{}

func (stringSchema) Parse(ctx context.Context, v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected string"}}
}

func (stringSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[string], error) {
	s, err := (stringSchema{}).Parse(ctx, v)
	return goskema.Decoded[string]{Value: s, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (stringSchema) TypeCheck(ctx context.Context, v any) error        { return nil }
func (stringSchema) RuleCheck(ctx context.Context, v any) error        { return nil }
func (stringSchema) Validate(ctx context.Context, v any) error         { return nil }
func (stringSchema) ValidateValue(ctx context.Context, v string) error { return nil }
func (stringSchema) JSONSchema() (*js.Schema, error)                   { return &js.Schema{}, nil }

type timeSchema struct{}

func (timeSchema) Parse(ctx context.Context, v any) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		return t, nil
	}
	return time.Time{}, goskema.Issues{{Path: "/", Code: goskema.CodeInvalidType, Message: "expected time.Time"}}
}

func (timeSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[time.Time], error) {
	t, err := (timeSchema{}).Parse(ctx, v)
	return goskema.Decoded[time.Time]{Value: t, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}
func (timeSchema) TypeCheck(ctx context.Context, v any) error { return nil }
func (timeSchema) RuleCheck(ctx context.Context, v any) error { return nil }
func (timeSchema) Validate(ctx context.Context, v any) error  { return nil }
func (timeSchema) ValidateValue(ctx context.Context, v time.Time) error {
	return nil
}
func (timeSchema) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }

func parseRFC3339(s string) (time.Time, error) {
	// Accept RFC3339Nano (trailing zeros optional)
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		if t2, err2 := time.Parse(time.RFC3339, s); err2 == nil {
			return t2, nil
		}
		return time.Time{}, err
	}
	return t, nil
}

func formatRFC3339Canonical(t time.Time) string {
	// Normalize to UTC and format using RFC3339Nano (Go trims trailing zeros)
	return t.UTC().Format(time.RFC3339Nano)
}
