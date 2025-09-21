package goskema

import (
	"context"

	js "github.com/reoring/goskema/jsonschema"
)

// Deprecated: prefer dsl.Array() helpers. This compatibility layer will eventually be removed.
// ---------------- Array ----------------

// NewArraySchema returns an array schema backed by the given element schema.
func NewArraySchema[E any](elem Schema[E]) Schema[[]E] {
	return &arraySchema[E]{elem: elem, minLen: -1, maxLen: -1}
}

type arraySchema[E any] struct {
	elem   Schema[E]
	minLen int
	maxLen int
}

// WithMin sets the minimum allowed length.
func (a *arraySchema[E]) WithMin(n int) *arraySchema[E] { a.minLen = n; return a }

// WithMax sets the maximum allowed length.
func (a *arraySchema[E]) WithMax(n int) *arraySchema[E] { a.maxLen = n; return a }

func (a *arraySchema[E]) Parse(ctx context.Context, v any) ([]E, error) {
	switch src := v.(type) {
	case []E:
		if err := a.ValidateValue(ctx, src); err != nil {
			return nil, err
		}
		return src, nil
	case []any:
		res := make([]E, 0, len(src))
		for i := range src {
			ev, err := a.elem.Parse(ctx, src[i])
			if err != nil {
				if iss, ok := AsIssues(err); ok {
					return nil, iss
				}
				return nil, Issues{Issue{Path: "/", Code: CodeParseError, Message: err.Error(), Cause: err}}
			}
			res = append(res, ev)
		}
		if err := a.ValidateValue(ctx, res); err != nil {
			return nil, err
		}
		return res, nil
	default:
		return nil, Issues{Issue{Path: "/", Code: CodeInvalidType, Message: "expected array"}}
	}
}

func (a *arraySchema[E]) ParseWithMeta(ctx context.Context, v any) (Decoded[[]E], error) {
	arr, err := a.Parse(ctx, v)
	return Decoded[[]E]{Value: arr, Presence: PresenceMap{"/": PresenceSeen}}, err
}

func (a *arraySchema[E]) TypeCheck(ctx context.Context, v any) error {
	switch v.(type) {
	case []E, []any:
		return nil
	default:
		return Issues{Issue{Path: "/", Code: CodeInvalidType, Message: "expected array"}}
	}
}

func (a *arraySchema[E]) RuleCheck(ctx context.Context, v any) error {
	var n int
	switch t := v.(type) {
	case []E:
		n = len(t)
	case []any:
		n = len(t)
	default:
		return nil
	}
	var iss Issues
	if a.minLen >= 0 && n < a.minLen {
		iss = AppendIssues(iss, Issue{Path: "/", Code: CodeTooShort, Message: "array is shorter than min"})
	}
	if a.maxLen >= 0 && n > a.maxLen {
		iss = AppendIssues(iss, Issue{Path: "/", Code: CodeTooLong, Message: "array is longer than max"})
	}
	if len(iss) > 0 {
		return iss
	}
	return nil
}

func (a *arraySchema[E]) Validate(ctx context.Context, v any) error {
	if err := a.TypeCheck(ctx, v); err != nil {
		return err
	}
	return a.RuleCheck(ctx, v)
}

func (a *arraySchema[E]) ValidateValue(ctx context.Context, v []E) error {
	if a.minLen >= 0 && len(v) < a.minLen {
		return Issues{Issue{Path: "/", Code: CodeTooShort, Message: "array is shorter than min"}}
	}
	if a.maxLen >= 0 && len(v) > a.maxLen {
		return Issues{Issue{Path: "/", Code: CodeTooLong, Message: "array is longer than max"}}
	}
	for i := range v {
		if err := a.elem.ValidateValue(ctx, v[i]); err != nil {
			return err
		}
	}
	return nil
}

func (a *arraySchema[E]) JSONSchema() (*js.Schema, error) { return &js.Schema{}, nil }
