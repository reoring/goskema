package source

import (
	goskema "github.com/reoring/goskema"
	drvgojson "github.com/reoring/goskema/source/gojson"
)

// init in a separate package to avoid import cycle in root. This sets go-json as default driver.
func init() { goskema.SetJSONDriver(drvgojson.Driver()) }
