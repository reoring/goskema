package goskema

// EncodePreservingObject returns a shallow-copied map that respects presence
// semantics for preserving output:
//   - Fields materialized only by defaults (PresenceDefaultApplied set while not seen)
//     are removed to keep them missing in the output.
//   - Fields explicitly present as null (PresenceWasNull) are kept as-is.
//   - Other fields are copied verbatim.
//
// This function operates only on the top-level object keys.
func EncodePreservingObject(db Decoded[map[string]any]) map[string]any {
	in := db.Value
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	for k := range in {
		p := db.Presence["/"+k]
		isDefaultOnly := p&PresenceDefaultApplied != 0 && p&PresenceSeen == 0 && p&PresenceWasNull == 0
		if isDefaultOnly {
			delete(out, k)
		}
	}
	return out
}

// EncodePreservingArray returns the slice as-is. Array elements currently do not
// carry default materialization semantics at element granularity in MVP.
func EncodePreservingArray(db Decoded[[]any]) []any { return db.Value }
