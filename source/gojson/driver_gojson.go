//go:build gojson

package gojson

import (
	"bytes"
	"io"
	"strconv"

	j "github.com/goccy/go-json"

	goskema "github.com/reoring/goskema"
	eng "github.com/reoring/goskema/internal/engine"
)

// Driver returns a goskema.JSONDriver backed by goccy/go-json.
func Driver() goskema.JSONDriver { return driverGoJSON{} }

type driverGoJSON struct{}

func (driverGoJSON) NewReader(r io.Reader) goskema.Source {
	return goskema.SourceFromEngine(NewReader(r), goskema.NumberJSONNumber)
}
func (driverGoJSON) NewBytes(b []byte) goskema.Source {
	return goskema.SourceFromEngine(NewBytes(b), goskema.NumberJSONNumber)
}
func (driverGoJSON) Name() string { return "go-json" }

// ---- engine.TokenSource implementation using go-json Decoder ----

type containerKind int

const (
	kindObject containerKind = iota
	kindArray
)

type frame struct {
	kind         containerKind
	expectingKey bool
}

type source struct {
	dec   *j.Decoder
	stack []frame
}

// NewReader wraps an io.Reader into an engine.TokenSource for JSON using go-json.
func NewReader(r io.Reader) eng.TokenSource {
	dec := j.NewDecoder(r)
	dec.UseNumber()
	return &source{dec: dec}
}

// NewBytes wraps a byte slice into an engine.TokenSource for JSON using go-json.
func NewBytes(b []byte) eng.TokenSource { return NewReader(bytes.NewReader(b)) }

func (s *source) NextToken() (eng.Token, error) {
	tok, err := s.dec.Token()
	if err != nil {
		if err == io.EOF {
			return eng.Token{}, io.EOF
		}
		return eng.Token{}, err
	}
	switch v := tok.(type) {
	case j.Delim:
		switch v {
		case '{':
			s.stack = append(s.stack, frame{kind: kindObject, expectingKey: true})
			return eng.Token{Kind: eng.KindBeginObject, Offset: -1}, nil
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
			return eng.Token{Kind: eng.KindEndObject, Offset: -1}, nil
		case '[':
			s.stack = append(s.stack, frame{kind: kindArray})
			return eng.Token{Kind: eng.KindBeginArray, Offset: -1}, nil
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
			return eng.Token{Kind: eng.KindEndArray, Offset: -1}, nil
		}
	case string:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && top.expectingKey {
				top.expectingKey = false
				return eng.Token{Kind: eng.KindKey, String: v, Offset: -1}, nil
			}
		}
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindString, String: v, Offset: -1}, nil
	case bool:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindBool, Bool: v, Offset: -1}, nil
	case j.Number:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNumber, Number: string(v), Offset: -1}, nil
	case float64:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNumber, Number: strconv.FormatFloat(v, 'g', -1, 64), Offset: -1}, nil
	case nil:
		if n := len(s.stack); n > 0 {
			top := &s.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
			}
		}
		return eng.Token{Kind: eng.KindNull, Offset: -1}, nil
	}
	if n := len(s.stack); n > 0 {
		top := &s.stack[n-1]
		if top.kind == kindObject && !top.expectingKey {
			top.expectingKey = true
		}
	}
	return eng.Token{Kind: eng.KindNull, Offset: -1}, nil
}

func (s *source) Location() int64 { return -1 }
