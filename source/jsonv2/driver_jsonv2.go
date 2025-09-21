//go:build jsonv2

package jsonv2

import (
	v2json "encoding/json/v2"
	"io"
	"sort"
	"strconv"

	goskema "github.com/reoring/goskema"
)

// Driver returns a goskema.JSONDriver backed by encoding/json/v2.
// Note: Requires building with -tags jsonv2 and GOEXPERIMENT=jsonv2.
func Driver() goskema.JSONDriver { return driverV2{} }

type driverV2 struct{}

func (driverV2) NewReader(r io.Reader) goskema.Source {
	// Read all then decode via v2; experimental path prioritizes simplicity.
	data, _ := io.ReadAll(r)
	return newV2SourceFromBytes(data)
}

func (driverV2) NewBytes(b []byte) goskema.Source { return newV2SourceFromBytes(b) }
func (driverV2) Name() string                     { return "encoding/json/v2" }

// v2Source materializes tokens from a decoded any tree (non-streaming fallback).
type v2Source struct {
	tokens []goskema.Token
	idx    int
}

func newV2SourceFromBytes(b []byte) goskema.Source {
	var v any
	if err := v2json.Unmarshal(b, &v); err != nil {
		// Surface error on first NextToken call by yielding no tokens
		return &v2Source{tokens: nil, idx: 0}
	}
	buf := make([]goskema.Token, 0, 64)
	buf = appendValueTokens(buf, v)
	return &v2Source{tokens: buf, idx: 0}
}

func (s *v2Source) NextToken() (goskema.Token, error) {
	if s.idx >= len(s.tokens) {
		return goskema.Token{}, io.EOF
	}
	t := s.tokens[s.idx]
	s.idx++
	return t, nil
}

func (s *v2Source) NumberMode() goskema.NumberMode { return goskema.NumberJSONNumber }
func (s *v2Source) Location() int64                { return -1 }

func appendValueTokens(out []goskema.Token, v any) []goskema.Token {
	switch x := v.(type) {
	case map[string]any:
		out = append(out, goskema.Token{Kind: goskema.TokenBeginObject, Offset: -1})
		// stable order for determinism
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, goskema.Token{Kind: goskema.TokenKey, String: k, Offset: -1})
			out = appendValueTokens(out, x[k])
		}
		out = append(out, goskema.Token{Kind: goskema.TokenEndObject, Offset: -1})
	case []any:
		out = append(out, goskema.Token{Kind: goskema.TokenBeginArray, Offset: -1})
		for _, e := range x {
			out = appendValueTokens(out, e)
		}
		out = append(out, goskema.Token{Kind: goskema.TokenEndArray, Offset: -1})
	case string:
		out = append(out, goskema.Token{Kind: goskema.TokenString, String: x, Offset: -1})
	case bool:
		out = append(out, goskema.Token{Kind: goskema.TokenBool, Bool: x, Offset: -1})
	case nil:
		out = append(out, goskema.Token{Kind: goskema.TokenNull, Offset: -1})
	case float64:
		out = append(out, goskema.Token{Kind: goskema.TokenNumber, Number: strconv.FormatFloat(x, 'g', -1, 64), Offset: -1})
	case int, int8, int16, int32, int64:
		out = append(out, goskema.Token{Kind: goskema.TokenNumber, Number: strconv.FormatInt(toInt64(x), 10), Offset: -1})
	case uint, uint8, uint16, uint32, uint64:
		out = append(out, goskema.Token{Kind: goskema.TokenNumber, Number: strconv.FormatUint(toUint64(x), 10), Offset: -1})
	default:
		// fallback to string representation
		out = append(out, goskema.Token{Kind: goskema.TokenString, String: toString(x), Offset: -1})
	}
	return out
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	}
	return 0
}

func toUint64(v any) uint64 {
	switch n := v.(type) {
	case uint:
		return uint64(n)
	case uint8:
		return uint64(n)
	case uint16:
		return uint64(n)
	case uint32:
		return uint64(n)
	case uint64:
		return n
	}
	return 0
}

func toString(v any) string {
	// best-effort number rendering when type is unexpected
	switch n := v.(type) {
	case float32:
		return strconv.FormatFloat(float64(n), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(n, 'g', -1, 64)
	}
	// generic
	return ""
}
