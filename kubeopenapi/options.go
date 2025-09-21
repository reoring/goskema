package kubeopenapi

import "fmt"

// UnknownBehavior configures how unknown fields are treated when importing CRD schemas.
type UnknownBehavior int

const (
	UnknownPrune UnknownBehavior = iota
	UnknownStrict
	UnknownPreserve
)

// DefaultMode controls how defaults from the schema are applied.
type DefaultMode int

const (
	DefaultIgnore DefaultMode = iota
	DefaultAnnotate
	DefaultApply
)

// Profile selects a compatibility profile.
type Profile string

const (
	ProfileStructuralV1 Profile = "structural-v1"
	ProfileLoose        Profile = "loose"
)

// Options controls import behavior for Kubernetes OpenAPI v3 schemas.
type Options struct {
	Profile              Profile
	Unknown              UnknownBehavior
	EnableCEL            bool
	DefaultMode          DefaultMode
	NumberMode           int // kept unexported dependency-light; wired at call sites
	PathRender           int // kept unexported dependency-light; wired at call sites
	EnableEmbeddedChecks bool
	Ambiguity            AmbiguityStrategy
}

// AmbiguityStrategy configures how to resolve ambiguous composites like oneOf/anyOf/if-then-else
// when multiple branches may match.
type AmbiguityStrategy int

const (
	AmbiguityError AmbiguityStrategy = iota
	AmbiguityFirstMatch
	AmbiguityScoreBest
)

// Diag carries non-fatal warnings produced during import.
type Diag interface {
	HasWarnings() bool
	Warnings() []string
}

type simpleDiag struct{ ws []string }

func (d *simpleDiag) HasWarnings() bool        { return len(d.ws) > 0 }
func (d *simpleDiag) Warnings() []string       { return append([]string(nil), d.ws...) }
func (d *simpleDiag) warnf(f string, a ...any) { d.ws = append(d.ws, fmt.Sprintf(f, a...)) }
