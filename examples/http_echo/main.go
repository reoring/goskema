package main

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
	echomw "github.com/reoring/goskema/middleware/echo"
)

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname,omitempty"`
}

func buildSchema() goskema.Schema[User] {
	return g.ObjectOf[User]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		Field("nickname", g.StringOf[string]()).Default("anon").
		UnknownStrict().
		MustBind()
}

func main() {
	e := echo.New()
	s := buildSchema()

	e.POST("/validate", func(c echo.Context) error {
		dm, _ := echomw.GetDecoded[User](c)
		canonical := preserveTopObjectUser(dm)
		return c.JSON(http.StatusOK, map[string]any{
			"ok":        true,
			"value":     dm.Value,
			"presence":  dm.Presence,
			"canonical": canonical,
		})
	}, echomw.ValidateJSON(s, goskema.ParseOpt{}))

	e.GET("/schema", func(c echo.Context) error {
		sch, err := s.JSONSchema()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
		c.Response().Header().Set("Content-Type", "application/json")
		return json.NewEncoder(c.Response().Writer).Encode(sch)
	})

	e.GET("/healthz", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })

	_ = e.Start(":8082")
}

func preserveTopObjectUser(db goskema.Decoded[User]) map[string]any {
	in := map[string]any{
		"id":       db.Value.ID,
		"email":    db.Value.Email,
		"nickname": db.Value.Nickname,
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	for k := range in {
		p := db.Presence["/"+k]
		isDefaultOnly := p&goskema.PresenceDefaultApplied != 0 && p&goskema.PresenceSeen == 0 && p&goskema.PresenceWasNull == 0
		if isDefaultOnly {
			delete(out, k)
		}
	}
	return out
}
