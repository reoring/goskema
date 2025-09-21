package dsl

import (
	"context"

	goskema "github.com/reoring/goskema"
)

type typedRule[T any] struct {
	name string
	fn   func(goskema.DomainCtx[T], T) []goskema.Issue
	opt  goskema.RefineOpt[T]
}

// typedRuleE is a variant for context rules that can return a fatal error to signal
// dependency outages or other non-422 conditions. It shares the same RefineOpt.
type typedRuleE[T any] struct {
	name string
	fn   func(goskema.DomainCtx[T], T) ([]goskema.Issue, error)
	opt  goskema.RefineOpt[T]
}

func shouldRunRule[T any](v T, pres goskema.PresenceMap, opt goskema.RefineOpt[T]) bool {
	// presence gating
	if len(opt.WhenSeen) > 0 {
		mode := opt.WhenSeenMode
		if mode == 0 {
			mode = goskema.SeenAll
		}
		switch mode {
		case goskema.SeenAll:
			for _, p := range opt.WhenSeen {
				if pres[p]&goskema.PresenceSeen == 0 {
					return false
				}
			}
		case goskema.SeenAny:
			any := false
			for _, p := range opt.WhenSeen {
				if pres[p]&goskema.PresenceSeen != 0 {
					any = true
					break
				}
			}
			if !any {
				return false
			}
		}
	}
	if opt.When != nil && !opt.When(v) {
		return false
	}
	return true
}

func runTypedRules[T any](ctx context.Context, v T, pres goskema.PresenceMap, rules []typedRule[T], phase goskema.Phase) goskema.Issues {
	var iss goskema.Issues
	r := goskema.NewRef(pres)
	dctx := goskema.DomainCtx[T]{Ctx: ctx, Presence: pres, Req: goskema.RequestInfo[T]{}, Ref: r}
	for _, tr := range rules {
		ph := tr.opt.Phase
		if ph == 0 {
			ph = goskema.PhaseDomain
		}
		if ph != phase {
			continue
		}
		if !shouldRunRule(v, pres, tr.opt) {
			continue
		}
		out := tr.fn(dctx, v)
		if len(out) == 0 {
			continue
		}
		for _, it := range out {
			it.Rule = tr.name
			if it.Params == nil {
				it.Params = map[string]any{"rule": tr.name}
			} else {
				if _, ok := it.Params["rule"]; !ok {
					it.Params["rule"] = tr.name
				}
			}
			iss = goskema.AppendIssues(iss, it)
		}
		if goskema.IsFailFast(ctx) {
			return iss
		}
	}
	return iss
}

// runTypedRulesE executes typed rules that may return a fatal error. When an error is returned,
// it is converted into a single Issue with CodeDependencyUnavailable and bubbled to callers.
func runTypedRulesE[T any](ctx context.Context, v T, pres goskema.PresenceMap, rules []typedRuleE[T], phase goskema.Phase) (goskema.Issues, error) {
	var iss goskema.Issues
	r := goskema.NewRef(pres)
	dctx := goskema.DomainCtx[T]{Ctx: ctx, Presence: pres, Req: goskema.RequestInfo[T]{}, Ref: r}
	for _, tr := range rules {
		ph := tr.opt.Phase
		if ph == 0 {
			ph = goskema.PhaseDomain
		}
		if ph != phase {
			continue
		}
		if !shouldRunRule(v, pres, tr.opt) {
			continue
		}
		out, err := tr.fn(dctx, v)
		if err != nil {
			// map to dependency_unavailable without path when not specified by rule; root path
			return iss, goskema.Issues{goskema.Issue{Path: "/", Code: goskema.CodeDependencyUnavailable, Message: err.Error(), Rule: tr.name, Params: map[string]any{"rule": tr.name}}}
		}
		if len(out) == 0 {
			continue
		}
		for _, it := range out {
			it.Rule = tr.name
			if it.Params == nil {
				it.Params = map[string]any{"rule": tr.name}
			} else if _, ok := it.Params["rule"]; !ok {
				it.Params["rule"] = tr.name
			}
			iss = goskema.AppendIssues(iss, it)
		}
		if goskema.IsFailFast(ctx) {
			return iss, nil
		}
	}
	return iss, nil
}
