package dsl

import (
	"context"
	"sort"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	js "github.com/reoring/goskema/jsonschema"
)

// legacy constructor removed in favor of builder-based Object().
type objectSchema struct {
	fields        map[string]AnyAdapter
	required      map[string]struct{}
	unknownPolicy goskema.UnknownPolicy
	unknownTarget string
	refines       []objRefine
	// typed rules are stored at the builder and copied here
	typedRulesAny any // holds []typedRule[T] at bind time; kept as any at map-level schema
	sortedKeys    []string
}

// Ensure objectSchema implements goskema.Schema[map[string]any]
var _ goskema.Schema[map[string]any] = (*objectSchema)(nil)

// sortedKnownKeys returns object field keys in ascending order for deterministic behavior.
func (o *objectSchema) sortedKnownKeys() []string {
	if o.sortedKeys != nil {
		return o.sortedKeys
	}
	kfs := make([]string, 0, len(o.fields))
	for k := range o.fields {
		kfs = append(kfs, k)
	}
	sort.Strings(kfs)
	o.sortedKeys = kfs
	return o.sortedKeys
}

// issuesFromErr converts an error into Issues, wrapping non-Issues with CodeParseError.
func issuesFromErr(path string, err error) goskema.Issues {
	if err == nil {
		return nil
	}
	if i2, ok := goskema.AsIssues(err); ok {
		return i2
	}
	return goskema.Issues{goskema.Issue{Path: path, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
}

// handleExistingField parses a present field value and records presence flags.
func (o *objectSchema) handleExistingField(ctx context.Context, k string, ad AnyAdapter, val any, pm goskema.PresenceMap) (any, goskema.Issues) {
	pm["/"+k] |= goskema.PresenceSeen
	if val == nil {
		pm["/"+k] |= goskema.PresenceWasNull
	}
	parsed, err := ad.parse(ctx, val)
	if err != nil {
		// If child returned Issues, rebase them under "/field"
		if child, ok := goskema.AsIssues(err); ok {
			var out goskema.Issues
			base := "/" + k
			for _, it := range child {
				p := it.Path
				if p == "" || p == "/" {
					p = base
				} else if len(p) > 0 && p[0] == '/' {
					p = base + p
				} else {
					p = base + "/" + p
				}
				out = goskema.AppendIssues(out, goskema.Issue{Path: p, Code: it.Code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
			}
			return nil, out
		}
		return nil, issuesFromErr("/"+k, err)
	}
	return parsed, nil
}

// handleMissingField applies a default when available and records presence; returns handled=true if default path executed.
func (o *objectSchema) handleMissingField(ctx context.Context, k string, ad AnyAdapter, pm goskema.PresenceMap) (any, goskema.Issues, bool) {
	if ad.applyDefault == nil {
		return nil, nil, false
	}
	dv, err := ad.applyDefault(ctx)
	if err != nil {
		return nil, issuesFromErr("/"+k, err), true
	}
	pm["/"+k] |= goskema.PresenceDefaultApplied
	return dv, nil, true
}

// collectKnownWithPresence parses known fields, applies defaults, and records presence flags.
func (o *objectSchema) collectKnownWithPresence(ctx context.Context, src map[string]any, pm goskema.PresenceMap) (map[string]any, goskema.Issues) {
	out := make(map[string]any, len(src))
	var iss goskema.Issues
	for _, k := range o.sortedKnownKeys() {
		ad := o.fields[k]
		if val, exists := src[k]; exists {
			parsed, i2 := o.handleExistingField(ctx, k, ad, val, pm)
			if len(i2) > 0 {
				iss = goskema.AppendIssues(iss, i2...)
				if goskema.IsFailFast(ctx) {
					return out, iss
				}
				continue
			}
			out[k] = parsed
			continue
		}
		// missing: apply default if provided; otherwise enforce required
		if dv, i2, handled := o.handleMissingField(ctx, k, ad, pm); handled {
			if len(i2) > 0 {
				iss = goskema.AppendIssues(iss, i2...)
				if goskema.IsFailFast(ctx) {
					return out, iss
				}
			} else {
				out[k] = dv
			}
			continue
		}
		if _, req := o.required[k]; req {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeRequired, Message: i18n.T(goskema.CodeRequired, nil), Hint: "required property missing"})
			if goskema.IsFailFast(ctx) {
				return out, iss
			}
		}
	}
	return out, iss
}

// collectUnknown processes unknown keys according to unknownPolicy and may write into out for passthrough.
func (o *objectSchema) collectUnknown(src map[string]any, out map[string]any) goskema.Issues {
	var iss goskema.Issues
	// unknown keys in key-sorted order
	uks := make([]string, 0, len(src))
	for k := range src {
		if _, known := o.fields[k]; !known {
			uks = append(uks, k)
		}
	}
	sort.Strings(uks)
	for _, k := range uks {
		v := src[k]
		switch o.unknownPolicy {
		case goskema.UnknownStrict:
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeUnknownKey, Message: i18n.T(goskema.CodeUnknownKey, nil)})
			// fail-fast shortcut
			// Note: we can't access ctx here; keep fail-fast handling at callers after merge
		case goskema.UnknownStrip:
			// drop
		case goskema.UnknownPassthrough:
			if o.unknownTarget == "" {
				iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeUnknownKey, Message: i18n.T(goskema.CodeUnknownKey, nil)})
				continue
			}
			extra, _ := out[o.unknownTarget].(map[string]any)
			if extra == nil {
				extra = map[string]any{}
			}
			extra[k] = v
			out[o.unknownTarget] = extra
		}
	}
	return iss
}

func (o *objectSchema) Parse(ctx context.Context, v any) (map[string]any, error) {
	src, ok := v.(map[string]any)
	if !ok {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	pm := goskema.PresenceMap{"/": goskema.PresenceSeen}
	out, iss := o.collectKnownWithPresence(ctx, src, pm)
	if goskema.IsFailFast(ctx) && len(iss) > 0 {
		return nil, iss
	}
	issUnknown := o.collectUnknown(src, out)
	if len(issUnknown) > 0 {
		iss = goskema.AppendIssues(iss, issUnknown...)
	}
	if len(iss) > 0 {
		return nil, iss
	}
	nn, err := goskema.ApplyNormalize[map[string]any](ctx, out, o)
	if err != nil {
		return nil, err
	}
	if err := goskema.ApplyRefine[map[string]any](ctx, nn, o); err != nil {
		return nil, err
	}
	return nn, nil
}

func (o *objectSchema) ParseWithMeta(ctx context.Context, v any) (goskema.Decoded[map[string]any], error) {
	pm := goskema.PresenceMap{"/": goskema.PresenceSeen}
	// fast-path: if not an object, delegate to Parse and return root presence only
	src, ok := v.(map[string]any)
	if !ok {
		m, err := o.Parse(ctx, v)
		return goskema.Decoded[map[string]any]{Value: m, Presence: pm}, err
	}

	out, iss := o.collectKnownWithPresence(ctx, src, pm)
	if goskema.IsFailFast(ctx) && len(iss) > 0 {
		return goskema.Decoded[map[string]any]{Value: nil, Presence: pm}, iss
	}
	issUnknown := o.collectUnknown(src, out)
	if len(issUnknown) > 0 {
		iss = goskema.AppendIssues(iss, issUnknown...)
	}
	if len(iss) > 0 {
		return goskema.Decoded[map[string]any]{Value: nil, Presence: pm}, iss
	}

	nn, err := goskema.ApplyNormalize[map[string]any](ctx, out, o)
	if err != nil {
		return goskema.Decoded[map[string]any]{}, err
	}
	if err := goskema.ApplyRefine[map[string]any](ctx, nn, o); err != nil {
		return goskema.Decoded[map[string]any]{}, err
	}
	return goskema.Decoded[map[string]any]{Value: nn, Presence: pm}, nil
}

func (o *objectSchema) TypeCheck(ctx context.Context, v any) error {
	if _, ok := v.(map[string]any); !ok {
		return goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected object"}}
	}
	return nil
}

func (o *objectSchema) RuleCheck(ctx context.Context, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	var iss goskema.Issues
	// required keys in key-sorted order
	rks := make([]string, 0, len(o.required))
	for k := range o.required {
		rks = append(rks, k)
	}
	sort.Strings(rks)
	for _, k := range rks {
		if _, ok := m[k]; !ok {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeRequired, Message: i18n.T(goskema.CodeRequired, nil), Hint: "required property missing"})
			if goskema.IsFailFast(ctx) {
				return iss
			}
		}
	}
	if len(iss) > 0 {
		return iss
	}
	return nil
}

func (o *objectSchema) Validate(ctx context.Context, v any) error {
	if err := o.TypeCheck(ctx, v); err != nil {
		return err
	}
	return o.RuleCheck(ctx, v)
}

func (o *objectSchema) ValidateValue(ctx context.Context, v map[string]any) error {
	// validate known fields in key-sorted order for deterministic error selection
	kfs := make([]string, 0, len(o.fields))
	for k := range o.fields {
		kfs = append(kfs, k)
	}
	sort.Strings(kfs)
	for _, k := range kfs {
		ad := o.fields[k]
		if val, ok := v[k]; ok {
			if err := ad.validateValue(ctx, val); err != nil {
				return err
			}
		} else if _, req := o.required[k]; req {
			return goskema.Issues{goskema.Issue{Path: "/" + k, Code: goskema.CodeRequired, Message: i18n.T(goskema.CodeRequired, nil), Hint: "required property missing"}}
		}
	}
	return nil
}

func (o *objectSchema) JSONSchema() (*js.Schema, error) {
	props := make(map[string]*js.Schema, len(o.fields))
	for k, ad := range o.fields {
		if ad.jsonSchema != nil {
			if ps, err := ad.jsonSchema(); err == nil && ps != nil {
				props[k] = ps
				continue
			}
		}
		props[k] = &js.Schema{}
	}
	// Required list (sorted for deterministic output)
	req := make([]string, 0, len(o.required))
	for k := range o.required {
		req = append(req, k)
	}
	sort.Strings(req)
	// Unknown policy mapping
	var additional any
	switch o.unknownPolicy {
	case goskema.UnknownStrict:
		additional = false
	case goskema.UnknownStrip:
		// Runtime accepts then discards unknown keys, so JSON Schema should mark
		// them as accepted (true).
		additional = true
	case goskema.UnknownPassthrough:
		// UnknownPassthrough implies additionalProperties allowed in JSON Schema terms.
		additional = true
	}
	return &js.Schema{Type: "object", Properties: props, Required: req, AdditionalProperties: additional}, nil
}

// Refine implements goskema.Refiner[map[string]any] using builder-registered hooks.
func (o *objectSchema) Refine(ctx context.Context, v map[string]any) error {
	if len(o.refines) == 0 {
		return nil
	}
	var iss goskema.Issues
	for _, r := range o.refines {
		if r.fn == nil {
			continue
		}
		if err := r.fn(ctx, v); err != nil {
			if i2, ok := goskema.AsIssues(err); ok {
				iss = goskema.AppendIssues(iss, i2...)
			} else {
				iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/", Code: "custom", Message: err.Error(), Cause: err})
			}
			if goskema.IsFailFast(ctx) {
				return iss
			}
		}
	}
	if len(iss) > 0 {
		return iss
	}
	return nil
}

type objRefine struct {
	name string
	fn   func(context.Context, map[string]any) error
}
