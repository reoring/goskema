package goskema

import (
	"io"
	"sync"

	eng "github.com/reoring/goskema/internal/engine"
	jsonsrc "github.com/reoring/goskema/source/json"
)

// tokenKind enumerates JSON token kinds.
type tokenKind int

const (
	_tokenBeginObject tokenKind = iota
	_tokenEndObject
	_tokenBeginArray
	_tokenEndArray
	_tokenKey
	_tokenString
	_tokenNumber
	_tokenBool
	_tokenNull
)

// Exported aliases so generated code can reference token kinds without relying
// on unstable APIs. The alias and constants mirror the internal tokenKind.
// NOTE: Generated code may branch on values such as goskema.TokenBeginObject.
type TokenKind = tokenKind

const (
	TokenBeginObject TokenKind = _tokenBeginObject
	TokenEndObject   TokenKind = _tokenEndObject
	TokenBeginArray  TokenKind = _tokenBeginArray
	TokenEndArray    TokenKind = _tokenEndArray
	TokenKey         TokenKind = _tokenKey
	TokenString      TokenKind = _tokenString
	TokenNumber      TokenKind = _tokenNumber
	TokenBool        TokenKind = _tokenBool
	TokenNull        TokenKind = _tokenNull
)

// Token describes a token in the input stream. Offset records the byte position
// when known (-1 otherwise).
type Token struct {
	Kind   tokenKind
	String string // Stored for key/string tokens.
	Number string // Stored as text; NumberMode controls downstream interpretation.
	Bool   bool
	Offset int64 // Approximate decoder.InputOffset().
}

// Source abstracts over polymorphic input sources.
type Source interface {
	NextToken() (Token, error)
	NumberMode() NumberMode
	Location() int64 // byte offset; -1 if unknown
}

// JSONDriver converts JSON input into a Source via a pluggable SPI. The default
// implementation is based on encoding/json and may be swapped with SetJSONDriver.
type JSONDriver interface {
	NewReader(r io.Reader) Source
	NewBytes(b []byte) Source
	Name() string
}

var (
	jsonDriverMu      sync.RWMutex
	currentJSONDriver JSONDriver = defaultJSONDriver{}
)

// SetJSONDriver replaces the global JSON driver; nil values are ignored.
func SetJSONDriver(d JSONDriver) {
	if d == nil {
		return
	}
	jsonDriverMu.Lock()
	currentJSONDriver = d
	jsonDriverMu.Unlock()
}

// UseDefaultJSONDriver restores the default encoding/json-backed driver.
func UseDefaultJSONDriver() {
	jsonDriverMu.Lock()
	currentJSONDriver = defaultJSONDriver{}
	jsonDriverMu.Unlock()
}

func getJSONDriver() JSONDriver {
	jsonDriverMu.RLock()
	d := currentJSONDriver
	jsonDriverMu.RUnlock()
	return d
}

// defaultJSONDriver wraps the encoding/json implementation.
type defaultJSONDriver struct{}

func (defaultJSONDriver) NewReader(r io.Reader) Source {
	return &engineSourceAdapter{inner: jsonsrc.NewReader(r), numMode: NumberJSONNumber}
}
func (defaultJSONDriver) NewBytes(b []byte) Source {
	return &engineSourceAdapter{inner: jsonsrc.NewBytes(b), numMode: NumberJSONNumber}
}
func (defaultJSONDriver) Name() string { return "encoding/json" }

// JSONReader wraps an io.Reader as a JSON Source.
func JSONReader(r io.Reader) Source { return getJSONDriver().NewReader(r) }

// JSONBytes wraps a byte slice as a JSON Source.
func JSONBytes(b []byte) Source { return getJSONDriver().NewBytes(b) }

// SourceFromEngine wraps an engine.TokenSource as a goskema.Source. Callers
// choose the NumberMode to inherit subtree context.
func SourceFromEngine(inner eng.TokenSource, mode NumberMode) Source {
	return &engineSourceAdapter{inner: inner, numMode: mode}
}

// EnforceSource wraps a Source with runtime enforcement (duplicate keys, depth, bytes)
// using public options projected to internal engine options. This enables generated code
// to apply enforcement without importing internal packages.
func EnforceSource(s Source, opt ParseOpt) Source {
	// Fast-path: if s already wraps an engine.TokenSource, unwrap to avoid
	// public<->engine adapter round-trips.
	if ea, ok := s.(*engineSourceAdapter); ok {
		enforced := eng.WrapWithEnforcement(ea.inner, eng.EnforceOptions{
			OnDuplicate: toEngineDup(opt.Strictness.OnDuplicateKey),
			MaxDepth:    opt.MaxDepth,
			MaxBytes:    opt.MaxBytes,
			IssueSink:   nil,
			FailFast:    opt.FailFast,
		})
		return &engineSourceAdapter{inner: enforced, numMode: s.NumberMode()}
	}
	engSrc := EngineTokenSource(s)
	enforced := eng.WrapWithEnforcement(engSrc, eng.EnforceOptions{
		OnDuplicate: toEngineDup(opt.Strictness.OnDuplicateKey),
		MaxDepth:    opt.MaxDepth,
		MaxBytes:    opt.MaxBytes,
		IssueSink:   nil,
		FailFast:    opt.FailFast,
	})
	return SourceFromEngine(enforced, s.NumberMode())
}

// EnforceSourceIfNeeded returns the original Source if the options are
// effectively disabled (ignore duplicate keys, zero depth, zero size),
// preventing unnecessary overhead for small inputs.
func EnforceSourceIfNeeded(s Source, opt ParseOpt) Source {
	if opt.Strictness.OnDuplicateKey == Ignore && opt.MaxDepth == 0 && opt.MaxBytes == 0 {
		return s
	}
	return EnforceSource(s, opt)
}

// EnforceSourceWith wraps a Source with runtime enforcement and forwards lightweight
// issues to the provided sink. The sink receives goskema.Issue values converted
// from internal engine issues. This enables generated code to collect duplicate
// key warnings or truncation notices in collect mode without importing internal packages.
func EnforceSourceWith(s Source, opt ParseOpt, sink func(Issue)) Source {
	// Fast-path unwrap when Source already backed by engine.TokenSource
	var forward func(eng.SimpleIssue)
	if sink != nil {
		forward = func(si eng.SimpleIssue) {
			// Convert to public Issue. Offset is best-effort from current source location.
			sink(Issue{Path: si.Path, Code: si.Code, Message: si.Message, Offset: s.Location()})
		}
	}
	if ea, ok := s.(*engineSourceAdapter); ok {
		enforced := eng.WrapWithEnforcement(ea.inner, eng.EnforceOptions{
			OnDuplicate: toEngineDup(opt.Strictness.OnDuplicateKey),
			MaxDepth:    opt.MaxDepth,
			MaxBytes:    opt.MaxBytes,
			IssueSink:   forward,
			FailFast:    opt.FailFast,
		})
		return &engineSourceAdapter{inner: enforced, numMode: s.NumberMode()}
	}
	engSrc := EngineTokenSource(s)
	enforced := eng.WrapWithEnforcement(engSrc, eng.EnforceOptions{
		OnDuplicate: toEngineDup(opt.Strictness.OnDuplicateKey),
		MaxDepth:    opt.MaxDepth,
		MaxBytes:    opt.MaxBytes,
		IssueSink:   forward,
		FailFast:    opt.FailFast,
	})
	return SourceFromEngine(enforced, s.NumberMode())
}

// WithNumberMode wraps a Source and overrides its NumberMode.
func WithNumberMode(s Source, m NumberMode) Source { return &overrideNumberMode{inner: s, mode: m} }

type overrideNumberMode struct {
	inner Source
	mode  NumberMode
}

func (o *overrideNumberMode) NextToken() (Token, error) { return o.inner.NextToken() }
func (o *overrideNumberMode) NumberMode() NumberMode    { return o.mode }
func (o *overrideNumberMode) Location() int64           { return o.inner.Location() }

type engineSourceAdapter struct {
	inner   eng.TokenSource
	numMode NumberMode
}

func (s *engineSourceAdapter) NextToken() (Token, error) {
	t, err := s.inner.NextToken()
	if err != nil {
		return Token{}, err
	}
	return Token{Kind: fromEngineKind(t.Kind), String: t.String, Number: t.Number, Bool: t.Bool, Offset: t.Offset}, nil
}
func (s *engineSourceAdapter) NumberMode() NumberMode { return s.numMode }
func (s *engineSourceAdapter) Location() int64        { return s.inner.Location() }

func fromEngineKind(k eng.Kind) tokenKind {
	switch k {
	case eng.KindBeginObject:
		return _tokenBeginObject
	case eng.KindEndObject:
		return _tokenEndObject
	case eng.KindBeginArray:
		return _tokenBeginArray
	case eng.KindEndArray:
		return _tokenEndArray
	case eng.KindKey:
		return _tokenKey
	case eng.KindString:
		return _tokenString
	case eng.KindNumber:
		return _tokenNumber
	case eng.KindBool:
		return _tokenBool
	case eng.KindNull:
		return _tokenNull
	default:
		return _tokenNull
	}
}
