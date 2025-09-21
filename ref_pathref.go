package goskema

import (
	"fmt"
	"strconv"
	"strings"
)

// Ref exposes helpers for refinement rules: presence access and path building.
type Ref interface {
	Presence() PresenceMap
	Root() PathRef
	At(path string) PathRef
}

// PathRef builds JSON Pointer paths in a chain-safe way and creates Issues.
type PathRef interface {
	Field(name string) PathRef
	Index(i int) PathRef
	Pointer() string
	Issue(code, msg string, kv ...any) Issue
}

type refImpl struct {
	presence PresenceMap
}

func NewRef(pm PresenceMap) Ref { return &refImpl{presence: pm} }

func (r *refImpl) Presence() PresenceMap { return r.presence }
func (r *refImpl) Root() PathRef         { return &pathRef{parts: nil} }
func (r *refImpl) At(path string) PathRef {
	if path == "" || path == "/" {
		return r.Root()
	}
	// naive split on '/', ignoring first empty due to leading '/'
	parts := []string{}
	for _, p := range strings.Split(path, "/") {
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}
	return &pathRef{parts: parts}
}

type pathRef struct {
	parts []string
}

func (p *pathRef) Field(name string) PathRef {
	if name == "" {
		return p
	}
	// escape '~' -> '~0', '/' -> '~1' per RFC6901
	esc := strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
	return &pathRef{parts: append(append([]string{}, p.parts...), esc)}
}

func (p *pathRef) Index(i int) PathRef {
	return &pathRef{parts: append(append([]string{}, p.parts...), strconv.Itoa(i))}
}

func (p *pathRef) Pointer() string {
	if len(p.parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(p.parts, "/")
}

func (p *pathRef) Issue(code, msg string, kv ...any) Issue {
	m := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[fmt.Sprint(kv[i])] = kv[i+1]
	}
	return Issue{Path: p.Pointer(), Code: code, Message: msg, Params: m}
}
