package engine

import (
	"encoding/json"
	"io"
	"strconv"
)

// Kind represents token kinds from a generic source.
type Kind int

const (
	KindBeginObject Kind = iota
	KindEndObject
	KindBeginArray
	KindEndArray
	KindKey
	KindString
	KindNumber
	KindBool
	KindNull
)

// Token represents a streaming token with approximate input offset.
type Token struct {
	Kind   Kind
	String string
	Number string
	Bool   bool
	Offset int64
}

// TokenSource is a minimal interface required by the engine.
type TokenSource interface {
	NextToken() (Token, error)
	Location() int64
}

// DecodeAnyFromSource builds an "any" value from the streaming token source.
func DecodeAnyFromSource(src TokenSource) (any, error) {
	tok, err := src.NextToken()
	if err != nil {
		return nil, err
	}
	return decodeValue(src, tok)
}

func decodeValue(src TokenSource, tok Token) (any, error) {
	switch tok.Kind {
	case KindBeginObject:
		return decodeObject(src)
	case KindBeginArray:
		return decodeArray(src)
	case KindString:
		return tok.String, nil
	case KindNumber:
		return json.Number(tok.Number), nil
	case KindBool:
		return tok.Bool, nil
	case KindNull:
		return nil, nil
	default:
		return nil, io.ErrUnexpectedEOF
	}
}

func decodeObject(src TokenSource) (any, error) {
	m := make(map[string]any)
	for {
		tok, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind == KindEndObject {
			return m, nil
		}
		if tok.Kind != KindKey {
			return nil, io.ErrUnexpectedEOF
		}
		vt, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		v, err := decodeValue(src, vt)
		if err != nil {
			return nil, err
		}
		m[tok.String] = v
	}
}

func decodeArray(src TokenSource) (any, error) {
	var arr []any
	for {
		tok, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind == KindEndArray {
			return arr, nil
		}
		v, err := decodeValue(src, tok)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
}

// ---- number-mode-aware variants (non-breaking additions) ----

type numberConv func(string) (any, error)

// DecodeAnyFromSourceAsFloat64 builds an "any" tree but decodes numbers as float64.
func DecodeAnyFromSourceAsFloat64(src TokenSource) (any, error) {
	return decodeAnyFromSourceWithConv(src, func(s string) (any, error) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	})
}

func decodeAnyFromSourceWithConv(src TokenSource, conv numberConv) (any, error) {
	tok, err := src.NextToken()
	if err != nil {
		return nil, err
	}
	return decodeValueWithConv(src, tok, conv)
}

func decodeValueWithConv(src TokenSource, tok Token, conv numberConv) (any, error) {
	switch tok.Kind {
	case KindBeginObject:
		return decodeObjectWithConv(src, conv)
	case KindBeginArray:
		return decodeArrayWithConv(src, conv)
	case KindString:
		return tok.String, nil
	case KindNumber:
		return conv(tok.Number)
	case KindBool:
		return tok.Bool, nil
	case KindNull:
		return nil, nil
	default:
		return nil, io.ErrUnexpectedEOF
	}
}

func decodeObjectWithConv(src TokenSource, conv numberConv) (any, error) {
	m := make(map[string]any)
	for {
		tok, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind == KindEndObject {
			return m, nil
		}
		if tok.Kind != KindKey {
			return nil, io.ErrUnexpectedEOF
		}
		vt, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		v, err := decodeValueWithConv(src, vt, conv)
		if err != nil {
			return nil, err
		}
		m[tok.String] = v
	}
}

func decodeArrayWithConv(src TokenSource, conv numberConv) (any, error) {
	var arr []any
	for {
		tok, err := src.NextToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind == KindEndArray {
			return arr, nil
		}
		v, err := decodeValueWithConv(src, tok, conv)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
}
