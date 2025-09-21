//go:build !jsonv2

package jsonv2

import (
	"io"

	goskema "github.com/reoring/goskema"
	jsonsrc "github.com/reoring/goskema/source/json"
)

// Driver returns a fallback driver when jsonv2 build tag is not enabled.
// It delegates to the default encoding/json-based source.
func Driver() goskema.JSONDriver { return driverStub{} }

type driverStub struct{}

func (driverStub) NewReader(r io.Reader) goskema.Source {
	return goskema.SourceFromEngine(jsonsrc.NewReader(r), goskema.NumberJSONNumber)
}

func (driverStub) NewBytes(b []byte) goskema.Source {
	return goskema.SourceFromEngine(jsonsrc.NewBytes(b), goskema.NumberJSONNumber)
}

func (driverStub) Name() string { return "encoding/json (jsonv2 stub)" }
