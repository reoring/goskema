package kubeopenapi

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"gopkg.in/yaml.v3"
)

// DuplicateKeyError reports a duplicate key found in a YAML mapping with both
// the first occurrence position and the duplicate occurrence position.
type DuplicateKeyError struct {
	Key       string
	FirstLine int
	FirstCol  int
	Line      int
	Col       int
}

func (e *DuplicateKeyError) Error() string {
	return fmt.Sprintf("duplicate YAML key %q at %d:%d (first at %d:%d)", e.Key, e.Line, e.Col, e.FirstLine, e.FirstCol)
}

// StrictYAMLReader decodes a multi-document YAML stream using yaml.Node to detect
// duplicate keys (with positions). It returns JSON-like Go values (map[string]any, []any, primitives).
type StrictYAMLReader struct {
	dec *yaml.Decoder
}

// NewStrictYAMLReader constructs a StrictYAMLReader.
func NewStrictYAMLReader(r io.Reader) *StrictYAMLReader {
	return &StrictYAMLReader{dec: yaml.NewDecoder(r)}
}

// Next returns the next YAML document converted into a JSON-compatible Go value.
// It returns (nil, io.EOF) when the stream is exhausted. Duplicate keys cause an error.
func (s *StrictYAMLReader) Next() (any, error) {
	var root yaml.Node
	if err := s.dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, nil
	}
	return nodeToInterfaceStrict(root.Content[0])
}

// ReadAll reads all documents from the YAML stream.
func (s *StrictYAMLReader) ReadAll() ([]any, error) {
	var out []any
	for {
		v, err := s.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			return nil, err
		}
		out = append(out, v)
	}
}

func nodeToInterfaceStrict(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return nodeToInterfaceStrict(n.Content[0])
	case yaml.MappingNode:
		m := make(map[string]any, len(n.Content)/2)
		first := make(map[string][2]int, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			v := n.Content[i+1]
			// Resolve key string (YAML spec keys should be scalars in our expected inputs)
			key := k.Value
			if pos, dup := first[key]; dup {
				return nil, &DuplicateKeyError{Key: key, FirstLine: pos[0], FirstCol: pos[1], Line: k.Line, Col: k.Column}
			}
			first[key] = [2]int{k.Line, k.Column}
			val, err := nodeToInterfaceStrict(v)
			if err != nil {
				return nil, err
			}
			m[key] = val
		}
		return m, nil
	case yaml.SequenceNode:
		arr := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := nodeToInterfaceStrict(c)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str", "!", "":
			return n.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool":
			if n.Value == "true" {
				return true, nil
			}
			if n.Value == "false" {
				return false, nil
			}
			return n.Value, nil
		case "!!int":
			// Use int64 to avoid overflow surprises; callers can coerce later
			if i, err := strconv.ParseInt(n.Value, 0, 64); err == nil {
				return i, nil
			}
			return n.Value, nil
		case "!!float":
			if f, err := strconv.ParseFloat(n.Value, 64); err == nil {
				return f, nil
			}
			return n.Value, nil
		default:
			// Fallback to raw string
			return n.Value, nil
		}
	default:
		return nil, nil
	}
}
