package goskema

import "context"

// Phase represents the validation phase after wire-level checks.
// PhaseDomain is pure (no I/O). PhaseContext may perform external I/O.
type Phase uint8

const (
	PhaseDomain Phase = iota
	PhaseContext
)

// SeenMode determines how WhenSeen conditions are evaluated.
// SeenAll requires all listed paths to be present; SeenAny requires at least one.
type SeenMode uint8

const (
	SeenAll SeenMode = iota
	SeenAny
)

// RefineOpt carries rule-time options used by typed refinement rules.
// T is the typed domain model of the schema.
type RefineOpt[T any] struct {
	// WhenSeen lists JSON Pointer paths that must satisfy presence for the rule to run.
	// Use SeenMode to choose All/Any semantics. Empty means no presence gating.
	WhenSeen     []string
	WhenSeenMode SeenMode
	// When is an additional predicate evaluated on the typed value.
	When func(T) bool
	// Phase selects the execution phase. Defaults to PhaseDomain if zero.
	Phase Phase
}

// Operation indicates the high-level request intent for validation.
type Operation uint8

const (
	OpCreate Operation = iota
	OpUpdate
	OpPatch
	OpDelete
)

// RequestInfo carries request-scoped information affecting validation behavior.
// Old is the previous value when applicable (Update/Patch). It can be nil.
type RequestInfo[T any] struct {
	Op  Operation
	Old *T
}

// DomainCtx provides typed refinement rules with execution context, presence, and request info.
type DomainCtx[T any] struct {
	Ctx      context.Context
	Presence PresenceMap
	Req      RequestInfo[T]
	Ref      Ref // Provided by the runner; safe to use for building paths and issues.
}
