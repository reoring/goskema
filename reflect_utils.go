package goskema

import (
	"reflect"
	"strings"
)

// ResolveStructKey applies the repository-wide rule to resolve a struct field's
// external key used by the DSL and PresenceMap.
// Priority: goskema:"name=..." > json tag name > field name; "-" disables the field.
func ResolveStructKey(sf reflect.StructField) string {
	if gt := sf.Tag.Get("goskema"); gt != "" {
		parts := strings.Split(gt, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "name=") {
				return strings.TrimPrefix(p, "name=")
			}
		}
	}
	if jt := sf.Tag.Get("json"); jt != "" {
		if jt == "-" {
			return "-"
		}
		// Extract the name before the first comma (options start there)
		name := jt
		if i := strings.IndexByte(jt, ','); i >= 0 {
			name = jt[:i]
		}
		// Per encoding/json, an empty name means use the field's name
		if name == "" {
			return sf.Name
		}
		return name
	}
	return sf.Name
}
