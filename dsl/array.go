package dsl

import (
	"context"
	"strconv"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

// ArrayBuilder exposes chaining methods for array schemas while implementing Schema[[]E].
type ArrayBuilder[E any] interface {
	goskema.Schema[[]E]
	Min(n int) ArrayBuilder[E]
	Max(n int) ArrayBuilder[E]
	// WithStreamContains enables streaming-time contains min/max counting using a lightweight
	// predicate over raw element values before full element parsing. This allows early failure
	// when max is exceeded and avoids buffering entire arrays.
	WithStreamContains(min, max int, pred func(any) bool) ArrayBuilder[E]
}

// Array returns an array schema with the given element schema.
func Array[E any](elem goskema.Schema[E]) ArrayBuilder[E] {
	return &ArraySchema[E]{elem: elem, minLen: -1, maxLen: -1}
}

type ArraySchema[E any] struct {
	elem   goskema.Schema[E]
	minLen int
	maxLen int
	// streaming contains (optional)
	containsMin  int
	containsMax  int
	containsPred func(any) bool
}

// ArrayOf adapts Array[E] to AnyAdapter for use in typed object builders.
// Example: Field("tags", d.ArrayOf[string](d.String()).Min(1))
func ArrayOf[E any](elem goskema.Schema[E]) AnyAdapter {
	return anyAdapterFromSchema[[]E](Array[E](elem))
}

// Min sets the minimum length.
func (a *ArraySchema[E]) Min(n int) ArrayBuilder[E] { a.minLen = n; return a }

// Max sets the maximum length.
func (a *ArraySchema[E]) Max(n int) ArrayBuilder[E] { a.maxLen = n; return a }

// WithStreamContains configures streaming-time contains checking.
func (a *ArraySchema[E]) WithStreamContains(min, max int, pred func(any) bool) ArrayBuilder[E] {
	a.containsMin = min
	a.containsMax = max
	a.containsPred = pred
	return a
}

// WithStreamContainsAny provides a non-chaining variant for external integrations.
func (a *ArraySchema[E]) WithStreamContainsAny(min, max int, pred func(any) bool) {
	a.containsMin = min
	a.containsMax = max
	a.containsPred = pred
}

func (a *ArraySchema[E]) Parse(ctx context.Context, v any) ([]E, error) {
	switch src := v.(type) {
	case []E:
		if err := a.ValidateValue(ctx, src); err != nil {
			return nil, err
		}
		nn, err := goskema.ApplyNormalize[[]E](ctx, src, a)
		if err != nil {
			return nil, err
		}
		if err := goskema.ApplyRefine[[]E](ctx, nn, a); err != nil {
			return nil, err
		}
		return nn, nil
	case []any:
		res := make([]E, 0, len(src))
		for i := range src {
			ev, err := a.elem.Parse(ctx, src[i])
			if err != nil {
				if iss, ok := goskema.AsIssues(err); ok {
					base := "/" + strconv.Itoa(i)
					var out goskema.Issues
					for _, it := range iss {
						p := it.Path
						code := it.Code
						if p == "" || p == "/" {
							// element-level failure surfaces as parse_error at index
							p = base
							code = goskema.CodeParseError
						} else if p[0] == '/' {
							p = base + p
						} else {
							p = base + "/" + p
						}
						out = goskema.AppendIssues(out, goskema.Issue{Path: p, Code: code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
					}
					return nil, out
				}
				return nil, goskema.Issues{goskema.Issue{Path: "/" + strconv.Itoa(i), Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
			}
			res = append(res, ev)
		}
		if err := a.ValidateValue(ctx, res); err != nil {
			return nil, err
		}
		nn, err := goskema.ApplyNormalize[[]E](ctx, res, a)
		if err != nil {
			return nil, err
		}
		if err := goskema.ApplyRefine[[]E](ctx, nn, a); err != nil {
			return nil, err
		}
		return nn, nil
	default:
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected array"}}
	}
}

func (a *ArraySchema[E]) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[[]E], error) {
	arr, err := a.Parse(ctx, v)
	return goskema.Decoded[[]E]{Value: arr, Presence: goskema.PresenceMap{"/": goskema.PresenceSeen}}, err
}

func (a *ArraySchema[E]) TypeCheck(ctx context.Context, v any) error {
	switch v.(type) {
	case []E, []any:
		return nil
	default:
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected array"}}
	}
}

func (a *ArraySchema[E]) RuleCheck(ctx context.Context, v any) error {
	var n int
	switch t := v.(type) {
	case []E:
		n = len(t)
	case []any:
		n = len(t)
	default:
		return nil
	}
	var iss goskema.Issues
	if a.minLen >= 0 && n < a.minLen {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/", Code: goskema.CodeTooShort, Message: i18n.T(goskema.CodeTooShort, nil), Hint: "array is shorter than min"})
	}
	if a.maxLen >= 0 && n > a.maxLen {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/", Code: goskema.CodeTooLong, Message: i18n.T(goskema.CodeTooLong, nil), Hint: "array is longer than max"})
	}
	if len(iss) > 0 {
		return iss
	}
	return nil
}

func (a *ArraySchema[E]) Validate(ctx context.Context, v any) error {
	if err := a.TypeCheck(ctx, v); err != nil {
		return err
	}
	return a.RuleCheck(ctx, v)
}

func (a *ArraySchema[E]) ValidateValue(ctx context.Context, v []E) error {
	if a.minLen >= 0 && len(v) < a.minLen {
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooShort, Message: i18n.T(goskema.CodeTooShort, nil), Hint: "array is shorter than min"}}
	}
	if a.maxLen >= 0 && len(v) > a.maxLen {
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooLong, Message: i18n.T(goskema.CodeTooLong, nil), Hint: "array is longer than max"}}
	}
	for i := range v {
		if err := a.elem.ValidateValue(ctx, v[i]); err != nil {
			return err
		}
	}
	return nil
}

func (a *ArraySchema[E]) JSONSchema() (*js.Schema, error) {
	// element schema
	es, err := a.elem.JSONSchema()
	if err != nil {
		return nil, err
	}
	s := &js.Schema{Type: "array", Items: es}
	if a.minLen >= 0 {
		n := a.minLen
		s.MinItems = &n
	}
	if a.maxLen >= 0 {
		n := a.maxLen
		s.MaxItems = &n
	}
	return s, nil
}
