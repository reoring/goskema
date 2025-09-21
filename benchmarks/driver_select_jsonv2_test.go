//go:build jsonv2

package goskema_test

import (
	goskema "github.com/reoring/goskema"
	drv "github.com/reoring/goskema/source/jsonv2"
)

func init() {
	goskema.SetJSONDriver(drv.Driver())
}
