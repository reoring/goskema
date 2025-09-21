package goskema

// UnknownPolicy controls how unknown keys are handled.
type UnknownPolicy int

const (
	UnknownStrict      UnknownPolicy = iota // Reject unknown keys with an error.
	UnknownStrip                            // Drop unknown keys.
	UnknownPassthrough                      // Preserve unknown keys (optionally storing them elsewhere).
)

// NumberMode dictates how numbers are interpreted.
type NumberMode int

const (
	NumberFloat64    NumberMode = iota // Fast mode (with potential precision loss).
	NumberJSONNumber                   // Preserve json.Number.
	NumberBigFloat                     // Future: *big.Float / *big.Rat family.
	NumberDecimal                      // Future: decimal implementations.
)

// Strictness configures enforcement for duplicate keys and NaN handling.
type Strictness struct {
	OnDuplicateKey Severity // Warn or Error (duplicate JSON keys).
	AllowNaN       bool     // Allow NaN/±Inf values.
}

// Severity expresses the severity level for issues.
type Severity int

const (
	Ignore Severity = iota
	Warn
	Error
)

// PresenceOpt configures presence collection for WithMeta-style parsing.
type PresenceOpt struct {
	Collect bool
	Include []string
	Exclude []string
}

// PathRenderOpt controls how paths are rendered into strings.
type PathRenderOpt struct {
	Lazy   bool
	Intern bool
}

// ParseOpt bundles parsing options.
type ParseOpt struct {
	Strictness Strictness
	MaxDepth   int
	MaxBytes   int64
	Presence   PresenceOpt
	PathRender PathRenderOpt
	FailFast   bool
}
