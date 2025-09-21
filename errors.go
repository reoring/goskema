package goskema

import (
	"errors"
	"fmt"
	"strings"
)

// Issue codes (exported consts for IDE completion and type safety by convention)
const (
	CodeInvalidType          = "invalid_type"
	CodeRequired             = "required"
	CodeUnknownKey           = "unknown_key"
	CodeDuplicateKey         = "duplicate_key"
	CodeTooSmall             = "too_small"
	CodeTooBig               = "too_big"
	CodeTooShort             = "too_short"
	CodeTooLong              = "too_long"
	CodePattern              = "pattern"
	CodeInvalidEnum          = "invalid_enum"
	CodeInvalidFormat        = "invalid_format"
	CodeDiscriminatorMissing = "discriminator_missing"
	CodeDiscriminatorUnknown = "discriminator_unknown"
	CodeUnionAmbiguous       = "union_ambiguous"
	CodeParseError           = "parse_error"
	CodeOverflow             = "overflow"
	CodeTruncated            = "truncated"
	// Domain/Context passes (business semantics)
	CodeDomainRange        = "domain_range"
	CodeAggregateViolation = "aggregate_violation"
	CodeUniqueness         = "uniqueness"
	CodeBusinessRule       = "business_rule"
	CodeConflict           = "conflict"
	// Dependency temporary/unavailable errors (for mapping to 5xx at API layer)
	CodeDependencyUnavailable = "dependency_unavailable"
)

// Issue represents a single validation entry.
type Issue struct {
	Path    string // JSON Pointer (for example: /items/2/price).
	Code    string // One of the codes listed above.
	Message string
	Hint    string // Optional: remediation hints, format names, etc.
	Cause   error  // Optional: underlying error.
	Offset  int64  // Byte offset in the input source (-1 when unknown).
	// InputFragment is an optional snippet of the offending input. Because it can
	// be expensive to produce, it is best-effort.
	InputFragment string
	// Params carries structured parameters (e.g., {"min":1, "max":10, "got":42})
	// for i18n and observability.
	Params map[string]any
	// Rule optionally records the rule name that produced this issue.
	Rule string
}

// Issues is a collection of validation errors that implements error.
type Issues []Issue

// Error summarizes the first few issues.
func (iss Issues) Error() string {
	if len(iss) == 0 {
		return ""
	}
	const maxShown = 3
	b := &strings.Builder{}
	n := len(iss)
	lim := n
	if lim > maxShown {
		lim = maxShown
	}
	for i := 0; i < lim; i++ {
		if i > 0 {
			b.WriteString("; ")
		}
		it := iss[i]
		// e.g. invalid_type at /path
		fmt.Fprintf(b, "%s at %s", it.Code, it.Path)
	}
	if n > lim {
		fmt.Fprintf(b, "; ... (total %d)", n)
	}
	return b.String()
}

// AppendIssues appends issues to the destination, initializing the slice when
// needed.
func AppendIssues(dst Issues, more ...Issue) Issues {
	if dst == nil {
		dst = Issues{}
	}
	dst = append(dst, more...)
	return dst
}

// AsIssues extracts Issues from an error using errors.As internally.
func AsIssues(err error) (Issues, bool) {
	if err == nil {
		return nil, false
	}
	var iss Issues
	if errors.As(err, &iss) {
		return iss, true
	}
	return nil, false
}
