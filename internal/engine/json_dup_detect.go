package engine

import (
	"bytes"
	"encoding/json"
	"io"
)

// DuplicateStrictness controls duplicate key handling in detection helpers.
type DuplicateStrictness int

const (
	DupIgnore DuplicateStrictness = iota
	DupWarn
	DupError
)

// SimpleIssue is a minimal issue representation used by internal helpers.
type SimpleIssue struct {
	Code    string
	Path    string
	Message string
}

// NOTE: containerKind/kindObject/kindArray/dupFrame are defined in enforce.go and reused here.

// DetectJSONDuplicateKeysBytes detects duplicate object keys from a JSON byte slice.
// If onDup is DupIgnore, no issues are produced. maxIssues < 0 means unlimited; 0 means disabled; >0 sets limit.
func DetectJSONDuplicateKeysBytes(data []byte, onDup DuplicateStrictness, maxIssues int) ([]SimpleIssue, error) {
	if onDup == DupIgnore {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return detectJSONDuplicateKeys(dec, onDup, maxIssues)
}

// DetectJSONDuplicateKeysReader detects duplicate object keys from an io.Reader.
// Note: this will consume the reader fully.
func DetectJSONDuplicateKeysReader(r io.Reader, onDup DuplicateStrictness, maxIssues int) ([]SimpleIssue, error) {
	if onDup == DupIgnore {
		return nil, nil
	}
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return detectJSONDuplicateKeys(dec, onDup, maxIssues)
}

func detectJSONDuplicateKeys(dec *json.Decoder, onDup DuplicateStrictness, maxIssues int) ([]SimpleIssue, error) {
	var issues []SimpleIssue
	var stack []dupFrame

	appendIssue := func(i SimpleIssue) {
		if maxIssues == 0 {
			return
		}
		issues = append(issues, i)
		if maxIssues > 0 && len(issues) >= maxIssues {
			issues = append(issues, SimpleIssue{Code: "truncated", Path: "/", Message: "max issues reached"})
		}
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			appendIssue(SimpleIssue{Code: "parse_error", Path: "/", Message: err.Error()})
			break
		}

		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{':
				stack = append(stack, dupFrame{kind: kindObject, keys: make(map[string]struct{}), expectingKey: true})
			case '[':
				stack = append(stack, dupFrame{kind: kindArray})
			case '}':
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
					if len(stack) > 0 {
						top := &stack[len(stack)-1]
						if top.kind == kindObject && !top.expectingKey {
							top.expectingKey = true
						}
					}
				}
			case ']':
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
					if len(stack) > 0 {
						top := &stack[len(stack)-1]
						if top.kind == kindObject && !top.expectingKey {
							top.expectingKey = true
						}
					}
				}
			}
		case string:
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				if top.kind == kindObject && top.expectingKey {
					if _, ok := top.keys[v]; ok {
						appendIssue(SimpleIssue{Code: "duplicate_key", Path: "/", Message: "key '" + v + "' duplicated"})
						if onDup == DupError {
							return issues, nil
						}
					}
					top.keys[v] = struct{}{}
					top.expectingKey = false
					continue
				}
			}
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				if top.kind == kindObject && !top.expectingKey {
					top.expectingKey = true
				}
			}
		default:
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				if top.kind == kindObject && !top.expectingKey {
					top.expectingKey = true
				}
			}
		}
	}

	return issues, nil
}
