package json

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	eng "github.com/reoring/goskema/internal/engine"
)

type containerKind int

const (
	kindObject containerKind = iota
	kindArray
)

type dupFrame struct {
	kind containerKind
	// keys map removed; duplicate key detection is handled by enforcement layer
	expectingKey bool
}

type jsonSource struct {
	dec        *json.Decoder
	stack      []dupFrame
	lastOffset int64
}

// NewReader wraps an io.Reader into an engine.TokenSource for JSON.
func NewReader(r io.Reader) eng.TokenSource {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return &jsonSource{dec: dec, stack: nil, lastOffset: -1}
}

// NewBytes wraps a byte slice into an engine.TokenSource for JSON.
func NewBytes(b []byte) eng.TokenSource { return NewReader(bytes.NewReader(b)) }

func (s *jsonSource) NextToken() (eng.Token, error) {
	tok, err := s.dec.Token()
	if err != nil {
		if err == io.EOF {
			return eng.Token{}, io.EOF
		}
		return eng.Token{}, err
	}
	s.lastOffset = s.dec.InputOffset()

	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			s.stack = append(s.stack, dupFrame{kind: kindObject, expectingKey: true})
			return eng.Token{Kind: eng.KindBeginObject, Offset: s.lastOffset}, nil
		case '}':
			if n := len(s.stack); n > 0 {
				s.stack = s.stack[:n-1]
			}
			if n := len(s.stack); n > 0 {
				top := &s.stack[n-1]
				if top.kind == kindObject && !top.expectingKey {
					top.expectingKey = true
				}
			}
			return eng.Token{Kind: eng.KindEndObject, Offset: s.lastOffset}, nil
		case '[':
			s.stack = append(s.stack, dupFrame{kind: kindArray})
			return eng.Token{Kind: eng.KindBeginArray, Offset: s.lastOffset}, nil
		case ']':
			if n := len(s.stack); n > 0 {
				s.stack = s.stack[:n-1]
			}
			if n := len(s.stack); n > 0 {
				top := &s.stack[n-1]
				if top.kind == kindObject && !top.expectingKey {
					top.expectingKey = true
				}
			}
			return eng.Token{Kind: eng.KindEndArray, Offset: s.lastOffset}, nil
		}
	case string:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && top.expectingKey {
				top.expectingKey = false
				return eng.Token{Kind: eng.KindKey, String: v, Offset: s.lastOffset}, nil
			}
		}
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindString, String: v, Offset: s.lastOffset}, nil
	case bool:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindBool, Bool: v, Offset: s.lastOffset}, nil
	case json.Number:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNumber, Number: string(v), Offset: s.lastOffset}, nil
	case float64:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNumber, Number: formatFloat(v), Offset: s.lastOffset}, nil
	case nil:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNull, Offset: s.lastOffset}, nil
	}

	if n := len(s.stack); n > 0 {
		top := &s.stack[n-1]
		if top.kind == kindObject && !top.expectingKey {
			top.expectingKey = true
		}
	}
	return eng.Token{Kind: eng.KindNull, Offset: s.lastOffset}, nil
}

func (s *jsonSource) Location() int64 { return s.lastOffset }

func formatFloat(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }
