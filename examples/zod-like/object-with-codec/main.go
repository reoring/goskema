package main

import (
	"context"
	"fmt"
	"time"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/codec"
	g "github.com/reoring/goskema/dsl"
)

func main() {
	ctx := context.Background()

	// Use a Codec within an object field.
	dt := g.Codec[string, time.Time](codec.TimeRFC3339())

	payload, _ := g.Object().
		Field("startDate", g.SchemaOf[time.Time](dt)).
		Field("title", g.StringOf[string]()).
		Require("startDate", "title").
		UnknownStrict().
		Build()

	in := []byte(`{"startDate":"2024-01-15T10:30:00Z","title":"kickoff"}`)
	v, err := goskema.ParseFrom(ctx, payload, goskema.JSONBytes(in))
	fmt.Println("decoded:", v["startDate"].(time.Time).Format(time.RFC3339), v["title"], err)
}
