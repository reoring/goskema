package goskema

import (
	"context"

	"errors"

	js "github.com/reoring/goskema/jsonschema"
)

// Schema surfaces the SRP-aligned pillars of construction, type checking, value
// validation, and typed validation.
type Schema[T any] interface {
	// Parse transforms an unknown input into T (Coerce -> Normalize (Default) ->
	// Validate -> Refine). It returns an error when validation fails.
	Parse(ctx context.Context, v any) (T, error)
	// ParseWithMeta returns the typed value together with presence metadata.
	ParseWithMeta(ctx context.Context, v any) (Decoded[T], error)

	// TypeCheck verifies structure, types, presence/nullable/unknown-policy
	// decisions, and determines which union branch applies.
	TypeCheck(ctx context.Context, v any) error

	// RuleCheck runs min/max/length/pattern/enum/Refine validations assuming
	// TypeCheck already succeeded.
	RuleCheck(ctx context.Context, v any) error

	// Validate composes TypeCheck followed by RuleCheck.
	Validate(ctx context.Context, v any) error

	// ValidateValue verifies a value already typed as T without any conversion.
	ValidateValue(ctx context.Context, v T) error

	// JSONSchema projects the schema into a JSON Schema representation.
	JSONSchema() (*js.Schema, error)
}

// Codec performs bidirectional transformation and validation between the wire
// representation A and the domain representation B.
type Codec[A, B any] interface {
	In() Schema[A]                              // Wire schema (input side).
	Out() Schema[B]                             // Domain schema (output side).
	Decode(ctx context.Context, a A) (B, error) // A (In) -> B (convert) -> Out.ValidateValue.
	Encode(ctx context.Context, b B) (A, error) // Out.ValidateValue -> A -> In.Parse for revalidation.
	// DecodeWithMeta returns the value and presence metadata (enabling
	// preserving encode).
	DecodeWithMeta(ctx context.Context, a A) (Decoded[B], error)
	// EncodePreserving emits output using preserving semantics guided by
	// presence metadata.
	EncodePreserving(ctx context.Context, db Decoded[B]) (A, error)
}

// EncodeMode exposes canonical vs preserving output intent at call sites.
// For non-WithMeta values, Preserving is not applicable and callers must supply presence via Decoded.
type EncodeMode int

const (
	EncodeCanonical EncodeMode = iota
	EncodePreserve
)

// ErrEncodePreserveRequiresPresence indicates EncodePreserve was requested without presence metadata.
// Callers should use EncodeWithDecoded (or codec-specific preserving helpers) when presence is required.
var ErrEncodePreserveRequiresPresence = errors.New("goskema: encode preserve requires presence; supply Decoded via EncodeWithDecoded")

// EncodeWithMode encodes a domain value using the given mode.
// If mode is EncodePreserve, this function returns ErrEncodePreserveRequiresPresence because
// preserving semantics require presence metadata. Prefer EncodeWithDecoded with a Decoded value.
func EncodeWithMode[A, B any](ctx context.Context, c Codec[A, B], b B, mode EncodeMode) (A, error) {
	if mode == EncodePreserve {
		var zero A
		return zero, ErrEncodePreserveRequiresPresence
	}
	return c.Encode(ctx, b)
}

// EncodeWithDecoded encodes a domain value using the given mode, consuming presence
// information when provided. When mode is EncodePreserve, this will call c.EncodePreserving
// to honor presence (missing vs null vs defaultApplied). When mode is EncodeCanonical it
// falls back to Encode.
func EncodeWithDecoded[A, B any](ctx context.Context, c Codec[A, B], db Decoded[B], mode EncodeMode) (A, error) {
	switch mode {
	case EncodePreserve:
		return c.EncodePreserving(ctx, db)
	default:
		return c.Encode(ctx, db.Value)
	}
}

// ---- Convenience wrappers (Zod-like entry points) ----

// Decode is a thin wrapper around Schema.Parse for forward (input->output) direction.
// It exists to mirror the Zod mental model where decode is the forward direction.
// For typed domain decoding via Codec, prefer c.Decode.
func Decode[T any](ctx context.Context, s Schema[T], v any) (T, error) {
	return s.Parse(ctx, v)
}

// Encode is a convenience wrapper over Codec.Encode (output->input) direction.
// Callers must provide a Codec because generic Schema does not define encode semantics.
// This intentionally keeps bidirectional transforms explicit and type-safe.
func Encode[A, B any](ctx context.Context, c Codec[A, B], b B) (A, error) {
	return c.Encode(ctx, b)
}

// Normalizer provides an optional hook to normalize typed values during the
// Normalize phase of parsing. If it is not implemented, the phase is skipped.
type Normalizer[T any] interface {
	Normalize(ctx context.Context, v T) (T, error)
}

// Refiner provides an optional hook at the end of parsing to perform
// cross-field validation or external I/O. If it is not implemented, the phase
// is skipped.
type Refiner[T any] interface {
	Refine(ctx context.Context, v T) error
}

// Presence/Decoded moved to presence.go to improve modularity.

// SafeParse parses v into T, returning (zero, false) on validation error.
func SafeParse[T any](ctx context.Context, s Schema[T], v any) (T, bool) {
	val, err := s.Parse(ctx, v)
	if err != nil {
		var zero T
		return zero, false
	}
	return val, true
}

// Is returns true if v conforms to the schema s (TypeCheck+RuleCheck).
func Is[T any](ctx context.Context, s Schema[T], v any) bool {
	return s.Validate(ctx, v) == nil
}

// ---- Parse-time context options (internal wiring, exported for subpackages) ----

type contextKey int

const (
	_ctxKeyFailFast contextKey = iota
	_ctxKeySkipTypedRules
)

// WithFailFast returns a child context that marks fail-fast parsing behavior.
// This is set by ParseFrom/ParseFromWithMeta based on ParseOpt and consumed by schema implementations.
func WithFailFast(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, _ctxKeyFailFast, enabled)
}

// IsFailFast reports whether the current parse should stop on the first issue.
func IsFailFast(ctx context.Context) bool {
	v := ctx.Value(_ctxKeyFailFast)
	b, _ := v.(bool)
	return b
}

// WithSkipTypedRules marks the context to skip executing typed domain/context rules during Parse.
// This is used by ParseWithMeta to avoid running typed rules twice; rules will be executed once
// with Presence available in the WithMeta path.
func WithSkipTypedRules(ctx context.Context, skip bool) context.Context {
	return context.WithValue(ctx, _ctxKeySkipTypedRules, skip)
}

// IsSkipTypedRules reports whether typed rules should be skipped for this parse.
func IsSkipTypedRules(ctx context.Context) bool {
	v := ctx.Value(_ctxKeySkipTypedRules)
	b, _ := v.(bool)
	return b
}

// ---- Streaming SPI (internal) ----

// ErrStreamingUnsupported indicates a Schema does not yet support streaming driver path.
// Callers should fallback to the legacy any-building path.
var ErrStreamingUnsupported = errors.New("streaming: unsupported")

// sourceParser is an internal interface for schemas that support source-driven parsing.
// Implementations can return ErrStreamingUnsupported to signal fallback to legacy path.
// NOTE: This is an internal SPI and may evolve without backward compatibility guarantees.
type sourceParser[T any] interface {
	ParseFromSource(ctx context.Context, src Source, opt ParseOpt) (T, error)
	ParseFromSourceWithMeta(ctx context.Context, src Source, opt ParseOpt) (Decoded[T], error)
}
