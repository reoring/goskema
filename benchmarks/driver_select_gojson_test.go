//go:build gojson

package goskema_test

import (
	goskema "github.com/reoring/goskema"
	drv "github.com/reoring/goskema/source/gojson"
)

func init() {
	goskema.SetJSONDriver(drv.Driver())
}
