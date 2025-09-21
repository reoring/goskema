package main

import (
	"context"
	"fmt"

	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/examples/user"
)

func main() {
	ctx := context.Background()
	src := goskema.JSONBytes([]byte(`{"name":"alice"}`))
	v, err := user.ParseFromUser(ctx, src)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("parsed: %+v\n", v)
}
