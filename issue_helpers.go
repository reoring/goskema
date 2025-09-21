package goskema

// IssueAt creates an Issue at the given path with provided code, message and params map.
// This is a convenience helper to improve readability at call sites with many parameters.
func IssueAt(p PathRef, code, msg string, params map[string]any) Issue {
	return Issue{Path: p.Pointer(), Code: code, Message: msg, Params: params}
}
