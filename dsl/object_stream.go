package dsl

import (
	"context"
	"sort"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	eng "github.com/reoring/goskema/internal/engine"
	str "github.com/reoring/goskema/internal/stream"
)

// rebaseIssuesUnder prefixes child issue paths with the given base, handling empty or root paths.
func rebaseIssuesUnder(base string, child goskema.Issues) goskema.Issues {
	var out goskema.Issues
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
	return out
}

// parseKnownValue parses a known field value using streaming adapter when available, otherwise
// falls back to decoding into any and parsing. Errors from children are rebased under "/field".
func (o *objectSchema) parseKnownValue(
	ctx context.Context,
	src goskema.Source,
	subtree eng.TokenSource,
	first eng.Token,
	ad AnyAdapter,
	opt goskema.ParseOpt,
	field string,
) (any, goskema.Issues) {
	base := "/" + field
	if ad.parseFromSource != nil {
		pre := str.NewPreloadedSource(subtree, first)
		pv, perr := ad.parseFromSource(ctx, goskema.SourceFromEngine(pre, src.NumberMode()), opt)
		if perr != nil {
			if child, ok := goskema.AsIssues(perr); ok {
				return nil, rebaseIssuesUnder(base, child)
			}
			return nil, goskema.Issues{goskema.Issue{Path: base, Code: goskema.CodeParseError, Message: perr.Error(), Cause: perr}}
		}
		return pv, nil
	}
	// fallback any-building
	pre := str.NewPreloadedSource(subtree, first)
	var anyVal any
	var err error
	if src.NumberMode() == goskema.NumberFloat64 {
		anyVal, err = eng.DecodeAnyFromSourceAsFloat64(pre)
	} else {
		anyVal, err = eng.DecodeAnyFromSource(pre)
	}
	if err != nil {
		return nil, goskema.Issues{goskema.Issue{Path: base, Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	v, perr := ad.parse(ctx, anyVal)
	if perr != nil {
		if child, ok := goskema.AsIssues(perr); ok {
			return nil, rebaseIssuesUnder(base, child)
		}
		return nil, goskema.Issues{goskema.Issue{Path: base, Code: goskema.CodeParseError, Message: perr.Error(), Cause: perr}}
	}
	return v, nil
}

// handleUnknownKey processes an unknown key according to the current unknownPolicy.
// It may append to unknownStrictKeys, mutate out (for passthrough), and return issues.
// When fail-fast applies, it returns shouldReturn=true with issues set.
func (o *objectSchema) handleUnknownKey(
	ctx context.Context,
	src goskema.Source,
	sub eng.TokenSource,
	key string,
	unknownStrictKeys *[]string,
	out map[string]any,
) (goskema.Issues, bool) {
	switch o.unknownPolicy {
	case goskema.UnknownStrict:
		// drain subtree
		for {
			if _, err := sub.NextToken(); err != nil {
				break
			}
		}
		*unknownStrictKeys = append(*unknownStrictKeys, key)
		if goskema.IsFailFast(ctx) {
			return goskema.Issues{goskema.Issue{Path: "/" + key, Code: goskema.CodeUnknownKey, Message: i18n.T(goskema.CodeUnknownKey, nil)}}, true
		}
		return nil, false
	case goskema.UnknownStrip:
		for {
			if _, err := sub.NextToken(); err != nil {
				break
			}
		}
		return nil, false
	case goskema.UnknownPassthrough:
		if o.unknownTarget == "" {
			for {
				if _, err := sub.NextToken(); err != nil {
					break
				}
			}
			*unknownStrictKeys = append(*unknownStrictKeys, key)
			if goskema.IsFailFast(ctx) {
				return goskema.Issues{goskema.Issue{Path: "/" + key, Code: goskema.CodeUnknownKey, Message: i18n.T(goskema.CodeUnknownKey, nil)}}, true
			}
			return nil, false
		}
		var anyVal any
		if src.NumberMode() == goskema.NumberFloat64 {
			anyVal, _ = eng.DecodeAnyFromSourceAsFloat64(sub)
		} else {
			anyVal, _ = eng.DecodeAnyFromSource(sub)
		}
		extra, _ := out[o.unknownTarget].(map[string]any)
		if extra == nil {
			extra = map[string]any{}
		}
		extra[key] = anyVal
		out[o.unknownTarget] = extra
		return nil, false
	default:
		return nil, false
	}
}

// issuesForUnknownStrictKeys converts collected unknown strict keys into Issues (sorted by key).
func (o *objectSchema) issuesForUnknownStrictKeys(keys []string) goskema.Issues {
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	var iss goskema.Issues
	for _, k := range keys {
		iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeUnknownKey, Message: i18n.T(goskema.CodeUnknownKey, nil)})
	}
	return iss
}

// applyDefaultsOrRequired applies defaults for missing known fields or emits required issues.
// Returns issues and whether fail-fast should trigger an immediate return.
func (o *objectSchema) applyDefaultsOrRequired(
	ctx context.Context,
	out map[string]any,
	present map[string]struct{},
) (goskema.Issues, bool) {
	var iss goskema.Issues
	for _, k := range o.sortedKnownKeys() {
		if _, ok := present[k]; ok {
			continue
		}
		ad := o.fields[k]
		if ad.applyDefault != nil {
			dv, derr := ad.applyDefault(ctx)
			if derr != nil {
				iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeParseError, Message: derr.Error(), Cause: derr})
				if goskema.IsFailFast(ctx) {
					return iss, true
				}
				continue
			}
			out[k] = dv
			continue
		}
		if _, rq := o.required[k]; rq {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + k, Code: goskema.CodeRequired, Message: i18n.T(goskema.CodeRequired, nil), Hint: "required property missing"})
			if goskema.IsFailFast(ctx) {
				return iss, true
			}
		}
	}
	return iss, false
}
