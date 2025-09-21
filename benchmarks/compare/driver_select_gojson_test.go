//go:build gojson

package compare_test

import (
	goskema "github.com/reoring/goskema"
	drv "github.com/reoring/goskema/source/gojson"
)

func init() { goskema.SetJSONDriver(drv.Driver()) }
