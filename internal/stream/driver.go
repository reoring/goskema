package stream

import (
	eng "github.com/reoring/goskema/internal/engine"
)

// NodeKind represents a plan node kind for streaming parsing.
type NodeKind int

const (
	NodeObject NodeKind = iota
	NodeArray
	NodePrimitive
)

// PlanNode is a minimal marker interface for plan nodes.
type PlanNode interface {
	Kind() NodeKind
}

// ObjectPlan describes an object node at plan level (skeleton).
type ObjectPlan struct{}

func (ObjectPlan) Kind() NodeKind { return NodeObject }

// ArrayPlan describes an array node at plan level (skeleton).
type ArrayPlan struct{}

func (ArrayPlan) Kind() NodeKind { return NodeArray }

// PrimitivePlan describes a primitive node at plan level (skeleton).
type PrimitivePlan struct{}

func (PrimitivePlan) Kind() NodeKind { return NodePrimitive }

// Plan holds the root node for a streaming parse (skeleton).
type Plan struct {
	Root PlanNode
}

// NewPlan constructs a Plan from a root node.
func NewPlan(root PlanNode) Plan { return Plan{Root: root} }

// Driver executes a streaming plan over a token source (skeleton, not wired).
type Driver struct {
	plan Plan
}

// NewDriver creates a new streaming driver for a given plan.
func NewDriver(p Plan) *Driver { return &Driver{plan: p} }

// TypeCheck performs a shape/type-only validation (skeleton).
func (d *Driver) TypeCheck(src eng.TokenSource) error {
	// TODO(stream): implement phased TypeCheck traversal over tokens
	_ = src
	return nil
}

// Parse builds a value by consuming tokens according to the plan (skeleton).
func (d *Driver) Parse(src eng.TokenSource) (any, error) {
	// TODO(stream): implement phased parse (Normalize/Default/Validate/Refine)
	_ = src
	return nil, nil
}
