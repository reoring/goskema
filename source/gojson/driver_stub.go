//go:build !gojson

package gojson

import (
	"io"

	goskema "github.com/reoring/goskema"
	jsonsrc "github.com/reoring/goskema/source/json"
)

// Driver returns a stub driver description when gojson tag is not enabled.
// It delegates to the encoding/json-based source directly to avoid recursion.
func Driver() goskema.JSONDriver { return stub{} }

type stub struct{}

func (stub) NewReader(r io.Reader) goskema.Source {
	return goskema.SourceFromEngine(jsonsrc.NewReader(r), goskema.NumberJSONNumber)
}
func (stub) NewBytes(b []byte) goskema.Source {
	return goskema.SourceFromEngine(jsonsrc.NewBytes(b), goskema.NumberJSONNumber)
}
func (stub) Name() string { return "encoding/json (gojson stub)" }
