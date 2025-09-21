package engine

import (
	"strconv"
	"strings"
)

// Enforcement wrapper for TokenSource to apply duplicate key handling,
// max depth checks, and max bytes truncation in a streaming fashion.

// EnforceOptions controls runtime enforcement behavior.
type EnforceOptions struct {
	OnDuplicate DuplicateStrictness
	MaxDepth    int
	MaxBytes    int64
	// IssueSink is an optional callback to receive lightweight issues when in collect mode.
	// If nil, issues are not reported unless they are fatal.
	IssueSink func(SimpleIssue)
	// FailFast stops at the first issue encountered (duplicate/depth/bytes), returning an error immediately.
	FailFast bool
}

type containerKind int

const (
	kindObject containerKind = iota
	kindArray
)

type dupFrame struct {
	kind         containerKind
	keys         map[string]struct{}
	expectingKey bool
	path         string
	nextIndex    int
	pendingKey   string
}

// IssueError is a lightweight error carrying a SimpleIssue.
type IssueError struct{ SimpleIssue }

func (e IssueError) Error() string { return e.SimpleIssue.Message }

// WrapWithEnforcement returns a TokenSource that enforces duplicate key policy,
// maximum nesting depth, and maximum consumed bytes.
func WrapWithEnforcement(inner TokenSource, opt EnforceOptions) TokenSource {
	return &enforcingTokenSource{inner: inner, opt: opt}
}

type enforcingTokenSource struct {
	inner TokenSource
	opt   EnforceOptions
	stack []dupFrame
	depth int
	path  string
}

func (e *enforcingTokenSource) NextToken() (Token, error) {
	tok, err := e.inner.NextToken()
	if err != nil {
		return Token{}, err
	}

	path := e.currentPathForToken(tok)
	npath := normalizeIssuePath(path)

	switch tok.Kind {
	case KindBeginObject:
		e.stack = append(e.stack, dupFrame{kind: kindObject, keys: make(map[string]struct{}), expectingKey: true, path: path})
		e.depth++
		if e.opt.MaxDepth > 0 && e.depth > e.opt.MaxDepth {
			si := SimpleIssue{Code: "parse_error", Path: npath, Message: "max depth exceeded"}
			if e.opt.IssueSink != nil {
				e.opt.IssueSink(si)
			}
			return Token{}, IssueError{si}
		}
	case KindEndObject:
		if n := len(e.stack); n > 0 {
			e.stack = e.stack[:n-1]
		}
		if e.depth > 0 {
			e.depth--
		}
		if n := len(e.stack); n > 0 {
			top := &e.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
				top.pendingKey = ""
			}
		}
	case KindBeginArray:
		e.stack = append(e.stack, dupFrame{kind: kindArray, path: path})
		e.depth++
		if e.opt.MaxDepth > 0 && e.depth > e.opt.MaxDepth {
			si := SimpleIssue{Code: "parse_error", Path: npath, Message: "max depth exceeded"}
			if e.opt.IssueSink != nil {
				e.opt.IssueSink(si)
			}
			return Token{}, IssueError{si}
		}
	case KindEndArray:
		if n := len(e.stack); n > 0 {
			e.stack = e.stack[:n-1]
		}
		if e.depth > 0 {
			e.depth--
		}
		if n := len(e.stack); n > 0 {
			top := &e.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
				top.pendingKey = ""
			}
		}
	case KindKey:
		if n := len(e.stack); n > 0 {
			top := &e.stack[n-1]
			if top.kind == kindObject && top.expectingKey {
				if e.opt.OnDuplicate != DupIgnore {
					if _, ok := top.keys[tok.String]; ok {
						msg := "key '" + tok.String + "' duplicated"
						si := SimpleIssue{Code: "duplicate_key", Path: npath, Message: msg}
						if e.opt.IssueSink != nil {
							e.opt.IssueSink(si)
						}
						if e.opt.OnDuplicate == DupError || e.opt.FailFast {
							return Token{}, IssueError{si}
						}
					}
				}
				top.keys[tok.String] = struct{}{}
				top.expectingKey = false
				top.pendingKey = tok.String
			}
		}
	case KindString, KindNumber, KindBool, KindNull:
		if n := len(e.stack); n > 0 {
			top := &e.stack[n-1]
			if top.kind == kindObject && !top.expectingKey {
				top.expectingKey = true
				top.pendingKey = ""
			}
		}
	}

	if e.opt.MaxBytes > 0 {
		if off := e.Location(); off >= 0 && off > e.opt.MaxBytes {
			si := SimpleIssue{Code: "truncated", Path: npath, Message: "max bytes exceeded"}
			if e.opt.IssueSink != nil {
				e.opt.IssueSink(si)
			}
			return Token{}, IssueError{si}
		}
	}

	return tok, nil
}

func (e *enforcingTokenSource) currentPathForToken(tok Token) string {
	var path string
	if len(e.stack) == 0 {
		switch tok.Kind {
		case KindKey:
			path = joinJSONPointer("", tok.String)
		case KindEndObject, KindEndArray:
			path = ""
		default:
			path = ""
		}
		e.path = path
		return path
	}

	top := &e.stack[len(e.stack)-1]
	switch tok.Kind {
	case KindKey:
		path = joinJSONPointer(top.path, tok.String)
		top.pendingKey = tok.String
	case KindBeginObject, KindBeginArray, KindString, KindNumber, KindBool, KindNull:
		if top.kind == kindArray {
			path = joinJSONPointer(top.path, strconv.Itoa(top.nextIndex))
			top.nextIndex++
		} else if top.kind == kindObject {
			if top.pendingKey != "" || !top.expectingKey {
				path = joinJSONPointer(top.path, top.pendingKey)
			} else {
				path = top.path
			}
		} else {
			path = top.path
		}
	case KindEndObject, KindEndArray:
		path = top.path
	default:
		path = top.path
	}

	e.path = path
	return path
}

func normalizeIssuePath(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

var jsonPointerEscaper = strings.NewReplacer("~", "~0", "/", "~1")

func escapeJSONPointerToken(s string) string {
	return jsonPointerEscaper.Replace(s)
}

func joinJSONPointer(base, token string) string {
	if base == "" {
		return "/" + escapeJSONPointerToken(token)
	}
	return base + "/" + escapeJSONPointerToken(token)
}

func (e *enforcingTokenSource) Location() int64 { return e.inner.Location() }
