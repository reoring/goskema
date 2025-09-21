package goskema

import (
	"strings"
	"sync"
)

// Presence is the bit flag collected by WithMeta APIs.
type Presence uint8

const (
    PresenceSeen           Presence = 1 << iota // Field appeared in the input.
    PresenceWasNull                             // Field value was null.
    PresenceDefaultApplied                      // Default value was applied.
)

// PresenceMap maps JSON Pointers to Presence flags.
type PresenceMap map[string]Presence

// Decoded carries the parsed value along with presence metadata.
type Decoded[T any] struct {
    Value    T
    Presence PresenceMap
}

// simple string interner for PresenceMap keys
var (
	_internMu   sync.RWMutex
	_internPool = map[string]string{}
)

func internString(s string) string {
	_internMu.RLock()
	if v, ok := _internPool[s]; ok {
		_internMu.RUnlock()
		return v
	}
	_internMu.RUnlock()

	_internMu.Lock()
	if v, ok := _internPool[s]; ok { // double-check
		_internMu.Unlock()
		return v
	}
	_internPool[s] = s
	_internMu.Unlock()
	return s
}

func applyPresenceOptions(pm PresenceMap, popt PresenceOpt, ropt PathRenderOpt) PresenceMap {
	if pm == nil {
		return nil
	}
	if !popt.Collect {
		return nil
	}
	// Build includes and excludes as prefixes
	var includes []string
	if len(popt.Include) > 0 {
		includes = popt.Include
	}
	var excludes []string
	if len(popt.Exclude) > 0 {
		excludes = popt.Exclude
	}

	filtered := make(PresenceMap, len(pm))

	shouldInclude := func(path string) bool {
		if len(includes) > 0 {
			ok := false
			for _, p := range includes {
				if strings.HasPrefix(path, p) {
					ok = true
					break
				}
			}
			if !ok {
				return false
			}
		}
		for _, p := range excludes {
			if strings.HasPrefix(path, p) {
				return false
			}
		}
		return true
	}

	for k, v := range pm {
		if !shouldInclude(k) {
			continue
		}
		key := k
		if ropt.Intern {
			key = internString(k)
		}
		filtered[key] = v
	}
	return filtered
}

// mergePresenceMaps returns a new PresenceMap that is the bitwise-OR merge of a and b.
func mergePresenceMaps(a, b PresenceMap) PresenceMap {
	if a == nil && b == nil {
		return nil
	}
	// pre-size roughly
	out := make(PresenceMap, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] |= v
	}
	return out
}

// collectPresenceMapFromValue walks a decoded value and collects JSON Pointer paths
// for objects (map[string]any) and arrays ([]any). Root path "/" is always marked seen.
func collectPresenceMapFromValue(v any) PresenceMap {
	pm := make(PresenceMap)
	pm["/"] = PresenceSeen
	collectPresenceRecurse(v, "", pm)
	return pm
}

func collectPresenceRecurse(v any, cur string, pm PresenceMap) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			p := cur + "/" + k
			pm[p] |= PresenceSeen
			collectPresenceRecurse(val, p, pm)
		}
	case []any:
		for i, val := range t {
			p := cur + "/" + itoa(i)
			pm[p] |= PresenceSeen
			collectPresenceRecurse(val, p, pm)
		}
	default:
		// primitives: nothing to descend
	}
}

// small local itoa to avoid extra imports here
func itoa(i int) string {
	// fast enough for test scenarios
	s := "0123456789"
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	bp := len(buf)
	for i > 0 {
		bp--
		buf[bp] = s[i%10]
		i /= 10
	}
	return string(buf[bp:])
}
