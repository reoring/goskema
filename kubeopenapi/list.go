package kubeopenapi

import (
	"context"
	"fmt"
	"strconv"

	goskema "github.com/reoring/goskema"
)

// listUniquenessChecker represents uniqueness checks for x-kubernetes-list-type.
type listUniquenessChecker interface {
	Check(fieldName string, val any) goskema.Issues
}

// setChecker performs uniqueness checks for set-type lists.
type setChecker struct{}

func (setChecker) Check(fieldName string, val any) goskema.Issues {
	arr, ok := val.([]string)
	if !ok {
		return nil // MVP: only []string fully supported
	}
	seen := make(map[string]int, len(arr))
	var iss goskema.Issues
	for i, sv := range arr {
		if j, dup := seen[sv]; dup {
			iss = goskema.AppendIssues(iss, goskema.Issue{
				Path:    "/" + fieldName + "/" + strconv.Itoa(i),
				Code:    "duplicate_item",
				Message: "duplicate element in set",
				Hint:    "first at /" + fieldName + "/" + strconv.Itoa(j),
			})
			continue
		}
		seen[sv] = i
	}
	return iss
}

// mapChecker performs uniqueness checks for map-type lists.
type mapChecker struct{ keys []string }

func (c mapChecker) Check(fieldName string, val any) goskema.Issues {
	// Preferred: []map[string]any
	if arr, ok := val.([]map[string]any); ok {
		return c.checkSliceOfMap(fieldName, arr)
	}
	// Compatible: best-effort to treat []any as map entries
	if gen, ok := val.([]any); ok {
		seen := make(map[string]int, len(gen))
		var iss goskema.Issues
		for i, it := range gen {
			m2, _ := it.(map[string]any)
			if m2 == nil {
				continue
			}
			comp, missing := c.compositeKey(m2)
			for _, k := range missing {
				iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + strconv.Itoa(i) + "/" + k, Code: goskema.CodeRequired, Message: "required for list-map-keys"})
			}
			if comp == "" {
				continue
			}
			if j, dup := seen[comp]; dup {
				iss = goskema.AppendIssues(iss, goskema.Issue{
					Path:    "/" + fieldName + "/" + strconv.Itoa(i),
					Code:    "duplicate_item",
					Message: "duplicate element in list-map by keys",
					Hint:    "first at /" + fieldName + "/" + strconv.Itoa(j),
				})
				continue
			}
			seen[comp] = i
		}
		return iss
	}
	return nil
}

func (c mapChecker) checkSliceOfMap(fieldName string, arr []map[string]any) goskema.Issues {
	seen := make(map[string]int, len(arr))
	var iss goskema.Issues
	for i, el := range arr {
		comp, missing := c.compositeKey(el)
		for _, k := range missing {
			iss = goskema.AppendIssues(iss, goskema.Issue{Path: "/" + fieldName + "/" + strconv.Itoa(i) + "/" + k, Code: goskema.CodeRequired, Message: "required for list-map-keys"})
		}
		if comp == "" {
			continue
		}
		if j, dup := seen[comp]; dup {
			iss = goskema.AppendIssues(iss, goskema.Issue{
				Path:    "/" + fieldName + "/" + strconv.Itoa(i),
				Code:    "duplicate_item",
				Message: "duplicate element in list-map by keys",
				Hint:    "first at /" + fieldName + "/" + strconv.Itoa(j),
			})
			continue
		}
		seen[comp] = i
	}
	return iss
}

func (c mapChecker) compositeKey(m map[string]any) (string, []string) {
	var comp string
	var missing []string
	for idx, k := range c.keys {
		v, exists := m[k]
		if !exists {
			missing = append(missing, k)
			continue
		}
		if idx > 0 {
			comp += "|"
		}
		comp += k + "=" + fmt.Sprint(v)
	}
	return comp, missing
}

var listCheckerFactories = map[string]func([]string) listUniquenessChecker{
	"set": func(_ []string) listUniquenessChecker { return setChecker{} },
	"map": func(keys []string) listUniquenessChecker { return mapChecker{keys: append([]string(nil), keys...)} },
}

func newListUniquenessChecker(lt string, keys []string) listUniquenessChecker {
	if f, ok := listCheckerFactories[lt]; ok {
		return f(keys)
	}
	return nil
}

// buildListUniquenessRefiner builds a uniqueness checker according to x-kubernetes-list-type.
func buildListUniquenessRefiner(name string, lt string, keys []string) func(ctx context.Context, m map[string]any) error {
	chk := newListUniquenessChecker(lt, keys)
	if chk == nil {
		return nil
	}
	return func(ctx context.Context, m map[string]any) error {
		val, ok := m[name]
		if !ok || val == nil {
			return nil
		}
		iss := chk.Check(name, val)
		if len(iss) > 0 {
			return iss
		}
		return nil
	}
}
