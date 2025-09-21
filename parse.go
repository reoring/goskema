package goskema

import (
	"context"
	"errors"
	"io"

	eng "github.com/reoring/goskema/internal/engine"
)

// ParseFrom is the primary entry point. It consumes tokens from the Source,
// builds an any value, and delegates validation to the Schema.
func ParseFrom[T any](ctx context.Context, s Schema[T], src Source, opts ...ParseOpt) (T, error) {
	var zero T
	if s == nil {
		return zero, singleIssue(CodeParseError, "nil schema")
	}

	var opt ParseOpt
	if len(opts) > 0 {
		opt = opts[len(opts)-1]
	}
	// propagate fail-fast intent via context for schema implementations
	if opt.FailFast {
		ctx = WithFailFast(ctx, true)
	}
	// streaming driver SPI detection
	if sp, ok := any(s).(sourceParser[T]); ok {
		if v, err := sp.ParseFromSource(ctx, src, opt); err == nil {
			return v, nil
		} else if !errors.Is(err, ErrStreamingUnsupported) {
			return zero, toIssues(err)
		}
	}
	// fallback: legacy any-building path
	v, err := decodeAnyFromSource(src, opt)
	if err != nil {
		return zero, toIssues(err)
	}

	return s.Parse(ctx, v)
}

// ParseFromWithMeta collects presence metadata alongside the parsed value. It
// currently gathers presence after constructing the any value and lets the
// Schema merge its own annotations.
func ParseFromWithMeta[T any](ctx context.Context, s Schema[T], src Source, opts ...ParseOpt) (Decoded[T], error) {
	var zero Decoded[T]
	if s == nil {
		return zero, singleIssue(CodeParseError, "nil schema")
	}
	opt := normalizeWithMetaOpt(opts)
	if opt.FailFast {
		ctx = WithFailFast(ctx, true)
	}
	// Avoid running typed rules twice: mark skip for initial Parse used inside schema implementations.
	ctx = WithSkipTypedRules(ctx, true)
	if sp, ok := any(s).(sourceParser[T]); ok {
		dm, err := sp.ParseFromSourceWithMeta(ctx, src, opt)
		// apply presence options for consistency with non-streaming path (even when err != nil)
		dm = applyPresenceToDecoded(dm, opt)
		if err == nil {
			return dm, nil
		}
		if !errors.Is(err, ErrStreamingUnsupported) {
			return dm, toIssues(err)
		}
	}
	v, err := decodeAnyFromSource(src, opt)
	if err != nil {
		return zero, toIssues(err)
	}
	dm, err := s.ParseWithMeta(ctx, v)
	dm = applyPresenceToDecoded(dm, opt)
	return dm, err
}

// ---- helpers (parse options, decode, presence, error mapping) ----

func normalizeWithMetaOpt(opts []ParseOpt) ParseOpt {
	var opt ParseOpt
	if len(opts) > 0 {
		opt = opts[len(opts)-1]
	}
	if !opt.Presence.Collect && len(opt.Presence.Include) == 0 && len(opt.Presence.Exclude) == 0 {
		opt.Presence.Collect = true
	}
	return opt
}

func decodeAnyFromSource(src Source, opt ParseOpt) (any, error) {
	engSrc := &tokenSourceAdapter{inner: src}
	enforced := eng.WrapWithEnforcement(engSrc, eng.EnforceOptions{
		OnDuplicate: toEngineDup(opt.Strictness.OnDuplicateKey),
		MaxDepth:    opt.MaxDepth,
		MaxBytes:    opt.MaxBytes,
		IssueSink:   nil,
		FailFast:    opt.FailFast,
	})
	// Switch behavior according to the requested NumberMode.
	switch src.NumberMode() {
	case NumberFloat64:
		return eng.DecodeAnyFromSourceAsFloat64(enforced)
	case NumberJSONNumber:
		fallthrough
	default:
		return eng.DecodeAnyFromSource(enforced)
	}
}

func applyPresenceToDecoded[T any](dm Decoded[T], opt ParseOpt) Decoded[T] {
	if opt.Presence.Collect {
		structPM := collectPresenceMapFromValue(any(dm.Value))
		// Merge with precedence: schema-provided presence wins for default-only fields.
		pm := make(PresenceMap, len(dm.Presence)+len(structPM))
		for k, v := range dm.Presence {
			pm[k] = v
		}
		for k, sv := range structPM {
			dv := pm[k]
			defaultOnly := dv&PresenceDefaultApplied != 0 && dv&PresenceSeen == 0 && dv&PresenceWasNull == 0
			if defaultOnly {
				// prevent introducing Seen via struct presence for default-only fields
				sv &^= PresenceSeen
			}
			pm[k] = dv | sv
		}
		dm.Presence = applyPresenceOptions(pm, opt.Presence, opt.PathRender)
		return dm
	}
	dm.Presence = nil
	return dm
}

func toIssues(err error) Issues {
	if err == nil {
		return nil
	}
	if ii, ok := AsIssues(err); ok {
		return ii
	}
	var ie eng.IssueError
	if errors.As(err, &ie) {
		return AppendIssues(nil, Issue{Code: ie.Code, Path: ie.Path, Message: ie.Message})
	}
	return AppendIssues(nil, Issue{Code: CodeParseError, Message: err.Error()})
}

func singleIssue(code, msg string) Issues { return AppendIssues(nil, Issue{Code: code, Message: msg}) }

// StreamParse validates input by streaming tokens from an io.Reader.
// When MaxBytes is set it enforces the size cap up front, otherwise it
// delegates directly to ParseFrom via the Source driver.
func StreamParse[T any](ctx context.Context, s Schema[T], r io.Reader, opts ...ParseOpt) (T, error) {
	if len(opts) > 0 && opts[len(opts)-1].MaxBytes > 0 {
		lr := io.LimitReader(r, opts[len(opts)-1].MaxBytes+1)
		data, err := io.ReadAll(lr)
		if err != nil {
			var zero T
			return zero, singleIssue(CodeParseError, err.Error())
		}
		if int64(len(data)) > opts[len(opts)-1].MaxBytes {
			var zero T
			return zero, singleIssue(CodeTruncated, "max bytes exceeded")
		}
		return ParseFrom[T](ctx, s, JSONBytes(data), opts...)
	}
	return ParseFrom[T](ctx, s, JSONReader(r), opts...)
}

// ---- Source -> engine.TokenSource adapter ----

type tokenSourceAdapter struct{ inner Source }

func (a *tokenSourceAdapter) NextToken() (eng.Token, error) {
	t, err := a.inner.NextToken()
	if err != nil {
		return eng.Token{}, err
	}
	return eng.Token{
		Kind:   toEngineKind(t.Kind),
		String: t.String,
		Number: t.Number,
		Bool:   t.Bool,
		Offset: t.Offset,
	}, nil
}

func (a *tokenSourceAdapter) Location() int64 { return a.inner.Location() }

// EngineTokenSource exposes the engine.TokenSource view of a goskema.Source for internal users.
func EngineTokenSource(s Source) eng.TokenSource {
	// Fast-path: if s is already an engine-backed source, reuse the inner source.
	if ea, ok := s.(*engineSourceAdapter); ok {
		return ea.inner
	}
	return &tokenSourceAdapter{inner: s}
}

func toEngineKind(k tokenKind) eng.Kind {
	switch k {
	case _tokenBeginObject:
		return eng.KindBeginObject
	case _tokenEndObject:
		return eng.KindEndObject
	case _tokenBeginArray:
		return eng.KindBeginArray
	case _tokenEndArray:
		return eng.KindEndArray
	case _tokenKey:
		return eng.KindKey
	case _tokenString:
		return eng.KindString
	case _tokenNumber:
		return eng.KindNumber
	case _tokenBool:
		return eng.KindBool
	case _tokenNull:
		return eng.KindNull
	default:
		return eng.KindNull
	}
}
