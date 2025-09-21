package kubeopenapi

import "strings"

// extractDefs returns local $defs map from the given document node.
func extractDefs(doc map[string]any) map[string]any {
	if m, ok := doc["$defs"].(map[string]any); ok {
		return m
	}
	return nil
}

// resolveRefsInPlace expands local $refs (limited to properties/items for MVP) using the provided $defs.
func resolveRefsInPlace(node map[string]any, defs map[string]any, d *simpleDiag, visited map[string]bool) {
	if node == nil || defs == nil {
		return
	}
	// properties
	if pm, ok := node["properties"].(map[string]any); ok {
		for k, raw := range pm {
			if sch, ok := raw.(map[string]any); ok {
				pm[k] = resolveOne(sch, defs, d, visited)
			}
		}
	}
	// items (single schema only for MVP)
	if it, ok := node["items"].(map[string]any); ok {
		node["items"] = resolveOne(it, defs, d, visited)
	}
}

// resolveOne expands a single schema map with local $ref, performing a shallow merge.
func resolveOne(s map[string]any, defs map[string]any, d *simpleDiag, visited map[string]bool) map[string]any {
	if s == nil {
		return nil
	}
	if ref, ok := s["$ref"].(string); ok {
		if !strings.HasPrefix(ref, "#/$defs/") {
			d.warnf("$ref %q not supported (local $defs only)", ref)
			return s
		}
		key := strings.TrimPrefix(ref, "#/$defs/")
		base, ok := defs[key].(map[string]any)
		if !ok {
			d.warnf("$ref to unknown $defs/%s", key)
			return s
		}
		if visited[key] {
			// cycle detected
			d.warnf("cyclic $ref detected at $defs/%s (skipping expansion)", key)
			return s
		}
		visited[key] = true
		copy := deepCopyMap(base)
		resolveRefsInPlace(copy, defs, d, visited)
		delete(visited, key)
		delete(s, "$ref")
		// merge resolved into s (shallow), preferring explicit fields in s
		for k, v := range copy {
			if _, exists := s[k]; !exists {
				s[k] = v
			}
		}
		return s
	}
	// descend
	resolveRefsInPlace(s, defs, d, visited)
	return s
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if mv, ok := v.(map[string]any); ok {
			out[k] = deepCopyMap(mv)
			continue
		}
		out[k] = v
	}
	return out
}
