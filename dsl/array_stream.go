package dsl

import (
	"context"
	"strconv"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/i18n"
	eng "github.com/reoring/goskema/internal/engine"
	str "github.com/reoring/goskema/internal/stream"
)

// ---- streaming SPI (temporary fallback) ----

func (a *ArraySchema[E]) ParseFromSource(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) ([]E, error) {
	engSrc := goskema.EngineTokenSource(src)
	var collected []eng.SimpleIssue
	dup := eng.DupIgnore
	switch opt.Strictness.OnDuplicateKey {
	case goskema.Error:
		dup = eng.DupError
	case goskema.Warn:
		dup = eng.DupWarn
	}
	enforced := eng.WrapWithEnforcement(engSrc, eng.EnforceOptions{
		OnDuplicate: dup,
		MaxDepth:    opt.MaxDepth,
		MaxBytes:    opt.MaxBytes,
		IssueSink: func(si eng.SimpleIssue) {
			collected = append(collected, si)
		},
		FailFast: opt.FailFast,
	})

	// expect begin array
	tok, err := enforced.NextToken()
	if err != nil {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if tok.Kind != eng.KindBeginArray {
		return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected array"}}
	}

	var out []E
	var iss goskema.Issues
	idx := 0
	// streaming contains counters
	doContains := a.containsPred != nil && (a.containsMin >= 0 || a.containsMax >= 0)
	matched := 0
	for {
		t, err := enforced.NextToken()
		if err != nil {
			return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
		}
		if t.Kind == eng.KindEndArray {
			break
		}
		if doContains {
			// Peek-only fast path: try to decode a minimal any and run predicate.
			// We use the engine decoder to avoid full element parse cost.
			pre := str.NewPreloadedSource(enforced, t)
			var anyVal any
			if src.NumberMode() == goskema.NumberFloat64 {
				anyVal, _ = eng.DecodeAnyFromSourceAsFloat64(pre)
			} else {
				anyVal, _ = eng.DecodeAnyFromSource(pre)
			}
			if a.containsPred(anyVal) {
				matched++
				if a.containsMax >= 0 && matched > a.containsMax {
					return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooLong, Message: i18n.T(goskema.CodeTooLong, nil), Hint: "contains max exceeded"}}
				}
			}
			// We consumed the subtree via DecodeAnyFromSource; continue to next element without full parse.
			idx++
			continue
		}
		pre := str.NewPreloadedSource(enforced, t)
		ev, perr := goskema.ParseFrom(ctx, a.elem, goskema.SourceFromEngine(pre, src.NumberMode()), opt)
		if perr != nil {
			if i2, ok := goskema.AsIssues(perr); ok {
				base := "/" + strconv.Itoa(idx)
				for _, it := range i2 {
					// rebase child issue paths under current index
					p := it.Path
					code := it.Code
					if p == "" || p == "/" {
						// element-level failure should surface as parse_error at index
						p = base
						code = goskema.CodeParseError
					} else if p[0] == '/' {
						p = base + p
					} else {
						p = base + "/" + p
					}
					iss = goskema.AppendIssues(iss, goskema.Issue{Path: p, Code: code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
				}
			} else {
				ie := goskema.Issue{Path: "/" + strconv.Itoa(idx), Code: goskema.CodeParseError, Message: perr.Error(), Cause: perr}
				if goskema.IsFailFast(ctx) {
					return nil, goskema.Issues{ie}
				}
				iss = goskema.AppendIssues(iss, ie)
			}
		} else {
			out = append(out, ev)
		}
		idx++
	}

	if doContains {
		if a.containsMin >= 0 && matched < a.containsMin {
			return nil, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeTooShort, Message: i18n.T(goskema.CodeTooShort, nil), Hint: "contains min not met"}}
		}
		// matched <= max is already enforced during scan
		// no further element parsing was done in contains-fast path, so out remains nil
		// Return empty typed slice (no projection) as contains-only validation path.
		return []E{}, nil
	}

	if len(iss) > 0 {
		return nil, iss
	}

	if err := a.ValidateValue(ctx, out); err != nil {
		return nil, err
	}

	if len(collected) > 0 && !goskema.IsFailFast(ctx) {
		var eiss goskema.Issues
		for _, si := range collected {
			eiss = goskema.AppendIssues(eiss, goskema.Issue{Path: si.Path, Code: si.Code, Message: si.Message})
		}
		if len(eiss) > 0 {
			return nil, eiss
		}
	}

	nn, err := goskema.ApplyNormalize[[]E](ctx, out, a)
	if err != nil {
		return nil, err
	}
	if err := goskema.ApplyRefine[[]E](ctx, nn, a); err != nil {
		return nil, err
	}
	return nn, nil
}

func (a *ArraySchema[E]) ParseFromSourceWithMeta(ctx context.Context, src goskema.Source, opt goskema.ParseOpt) (goskema.Decoded[[]E], error) {
	engSrc := goskema.EngineTokenSource(src)
	var collected []eng.SimpleIssue
	dup := eng.DupIgnore
	switch opt.Strictness.OnDuplicateKey {
	case goskema.Error:
		dup = eng.DupError
	case goskema.Warn:
		dup = eng.DupWarn
	}
	enforced := eng.WrapWithEnforcement(engSrc, eng.EnforceOptions{
		OnDuplicate: dup,
		MaxDepth:    opt.MaxDepth,
		MaxBytes:    opt.MaxBytes,
		IssueSink: func(si eng.SimpleIssue) {
			collected = append(collected, si)
		},
		FailFast: opt.FailFast,
	})

	tok, err := enforced.NextToken()
	if err != nil {
		return goskema.Decoded[[]E]{}, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
	}
	if tok.Kind != eng.KindBeginArray {
		return goskema.Decoded[[]E]{}, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeInvalidType, Message: i18n.T(goskema.CodeInvalidType, nil), Hint: "expected array"}}
	}

	var out []E
	pm := goskema.PresenceMap{"/": goskema.PresenceSeen}
	var iss goskema.Issues
	idx := 0
	for {
		t, err := enforced.NextToken()
		if err != nil {
			return goskema.Decoded[[]E]{}, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeParseError, Message: err.Error(), Cause: err}}
		}
		if t.Kind == eng.KindEndArray {
			break
		}
		path := "/" + strconv.Itoa(idx)
		pm[path] |= goskema.PresenceSeen
		if t.Kind == eng.KindNull {
			pm[path] |= goskema.PresenceWasNull
		}
		pre := str.NewPreloadedSource(enforced, t)
		dv, perr := goskema.ParseFromWithMeta(ctx, a.elem, goskema.SourceFromEngine(pre, src.NumberMode()), opt)
		if perr != nil {
			if i2, ok := goskema.AsIssues(perr); ok {
				base := path
				for _, it := range i2 {
					p := it.Path
					code := it.Code
					if p == "" || p == "/" {
						p = base
						code = goskema.CodeParseError
					} else if p[0] == '/' {
						p = base + p
					} else {
						p = base + "/" + p
					}
					iss = goskema.AppendIssues(iss, goskema.Issue{Path: p, Code: code, Message: it.Message, Hint: it.Hint, Cause: it.Cause})
				}
			} else {
				ie := goskema.Issue{Path: path, Code: goskema.CodeParseError, Message: perr.Error(), Cause: perr}
				if goskema.IsFailFast(ctx) {
					return goskema.Decoded[[]E]{Value: nil, Presence: pm}, goskema.Issues{ie}
				}
				iss = goskema.AppendIssues(iss, ie)
			}
		} else {
			out = append(out, dv.Value)
		}
		idx++
	}

	if len(iss) > 0 {
		return goskema.Decoded[[]E]{Value: nil, Presence: pm}, iss
	}

	if err := a.ValidateValue(ctx, out); err != nil {
		return goskema.Decoded[[]E]{Value: nil, Presence: pm}, err
	}

	if len(collected) > 0 && !goskema.IsFailFast(ctx) {
		var eiss goskema.Issues
		for _, si := range collected {
			eiss = goskema.AppendIssues(eiss, goskema.Issue{Path: si.Path, Code: si.Code, Message: si.Message})
		}
		if len(eiss) > 0 {
			return goskema.Decoded[[]E]{Value: nil, Presence: pm}, eiss
		}
	}

	nn, err := goskema.ApplyNormalize[[]E](ctx, out, a)
	if err != nil {
		return goskema.Decoded[[]E]{}, err
	}
	if err := goskema.ApplyRefine[[]E](ctx, nn, a); err != nil {
		return goskema.Decoded[[]E]{}, err
	}
	return goskema.Decoded[[]E]{Value: nn, Presence: pm}, nil
}
