package dsl

import (
	"strconv"

	goskema "github.com/reoring/goskema"
)

// markPresenceSubtree records presence bits for a value subtree under the given base JSON Pointer.
// It marks the base as seen, sets WasNull for nulls, and descends into maps and arrays to mark
// nested paths as seen. Keys are appended using straight concatenation (same convention as elsewhere).
func markPresenceSubtree(pm goskema.PresenceMap, base string, v any) {
	if pm == nil {
		return
	}
	pm[base] |= goskema.PresenceSeen
	switch t := v.(type) {
	case nil:
		pm[base] |= goskema.PresenceWasNull
	case map[string]any:
		for k, val := range t {
			markPresenceSubtree(pm, base+"/"+k, val)
		}
	case []any:
		for i, val := range t {
			markPresenceSubtree(pm, base+"/"+strconv.Itoa(i), val)
		}
	default:
		// primitives: nothing further
	}
}
