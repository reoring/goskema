package stream

import (
	"io"

	eng "github.com/reoring/goskema/internal/engine"
)

// SubtreeSource wraps an engine.TokenSource and exposes only a single
// object/array subtree starting from the first token provided at construction.
// It stops after the matching EndObject/EndArray is observed.
// Skeleton implementation; not wired into ParseFrom yet.
type SubtreeSource struct {
	inner eng.TokenSource
	// depth counts nested containers; 0 means the first begin is not seen yet.
	depth int
	done  bool
	// started indicates we've consumed the initial token that defines the subtree.
	started bool
}

// NewSubtreeSource constructs a subtree view over the next container in inner.
func NewSubtreeSource(inner eng.TokenSource) *SubtreeSource { return &SubtreeSource{inner: inner} }

func (s *SubtreeSource) NextToken() (eng.Token, error) {
	if s.done {
		return eng.Token{}, io.EOF
	}
	tok, err := s.inner.NextToken()
	if err != nil {
		return eng.Token{}, err
	}
	switch tok.Kind {
	case eng.KindBeginObject, eng.KindBeginArray:
		if !s.started {
			s.started = true
		}
		s.depth++
	case eng.KindEndObject, eng.KindEndArray:
		if s.depth > 0 {
			s.depth--
			if s.depth == 0 {
				// Emit this end token, then mark done for subsequent calls.
				if !s.done {
					s.done = true
					return tok, nil
				}
			}
		}
	}
	// If not started and we received a primitive, this defines a primitive subtree.
	if !s.started {
		s.started = true
		// primitives don't increase depth; single token subtree → mark done after return
		switch tok.Kind {
		case eng.KindString, eng.KindNumber, eng.KindBool, eng.KindNull:
			// Mark done for the subsequent call
			s.done = true
		}
	} else if s.started && s.depth == 0 {
		// If we've already closed the container, mark done and stop.
		s.done = true
	}
	return tok, nil
}

func (s *SubtreeSource) Location() int64 { return s.inner.Location() }

// PreloadedSource is a subtree source that first returns a preloaded token
// (typically the first token of an element) and then continues to stream the
// remaining tokens for the same subtree from the underlying source. It stops
// after the subtree end is reached, returning io.EOF afterwards.
type PreloadedSource struct {
	inner       eng.TokenSource
	preloaded   *eng.Token
	depth       int
	done        bool
	firstServed bool
}

// NewPreloadedSource constructs a subtree source that will return the provided
// first token before consuming further tokens from inner. The subtree boundary
// is determined by matching container begin/end pairs starting from the first
// token.
func NewPreloadedSource(inner eng.TokenSource, first eng.Token) *PreloadedSource {
	return &PreloadedSource{inner: inner, preloaded: &first, depth: 0, done: false, firstServed: false}
}

func (p *PreloadedSource) NextToken() (eng.Token, error) {
	if p.done {
		return eng.Token{}, io.EOF
	}
	if p.preloaded != nil && !p.firstServed {
		t := *p.preloaded
		p.firstServed = true
		// update depth/start state according to the first token
		switch t.Kind {
		case eng.KindBeginObject, eng.KindBeginArray:
			p.depth++
		case eng.KindEndObject, eng.KindEndArray:
			if p.depth > 0 {
				p.depth--
				if p.depth == 0 {
					p.done = true
				}
			}
		default:
			// primitives: single-token subtree
			p.done = true
		}
		return t, nil
	}

	tok, err := p.inner.NextToken()
	if err != nil {
		return eng.Token{}, err
	}
	switch tok.Kind {
	case eng.KindBeginObject, eng.KindBeginArray:
		p.depth++
	case eng.KindEndObject, eng.KindEndArray:
		if p.depth > 0 {
			p.depth--
			if p.depth == 0 {
				// emit this end token and mark done afterwards
				if !p.done {
					p.done = true
					return tok, nil
				}
			}
		}
	}
	// If depth reached zero (primitives handled above), mark done for subsequent calls
	if p.depth == 0 && p.firstServed {
		p.done = true
	}
	return tok, nil
}

func (p *PreloadedSource) Location() int64 { return p.inner.Location() }
