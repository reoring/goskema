package goskema

import (
	"io"

	eng "github.com/reoring/goskema/internal/engine"
)

// DetectJSONDuplicateKeysBytes is a thin wrapper that detects duplicate keys in
// JSON byte slices. The implementation delegates to internal/engine.
func DetectJSONDuplicateKeysBytes(data []byte, strict Strictness, maxIssues int) (Issues, error) {
	mode := toEngineDup(strict.OnDuplicateKey)
	si, err := eng.DetectJSONDuplicateKeysBytes(data, mode, maxIssues)
	if err != nil {
		return nil, err
	}
	return fromEngineIssues(si), nil
}

// DetectJSONDuplicateKeysReader is a thin wrapper that detects duplicate keys
// from an io.Reader.
func DetectJSONDuplicateKeysReader(r io.Reader, strict Strictness, maxIssues int) (Issues, error) {
	mode := toEngineDup(strict.OnDuplicateKey)
	si, err := eng.DetectJSONDuplicateKeysReader(r, mode, maxIssues)
	if err != nil {
		return nil, err
	}
	return fromEngineIssues(si), nil
}

func toEngineDup(s Severity) eng.DuplicateStrictness {
	switch s {
	case Error:
		return eng.DupError
	case Warn:
		return eng.DupWarn
	default:
		return eng.DupIgnore
	}
}

func fromEngineIssues(si []eng.SimpleIssue) Issues {
	var iss Issues
	for _, s := range si {
		iss = AppendIssues(iss, Issue{Code: s.Code, Path: s.Path, Message: s.Message})
	}
	return iss
}
