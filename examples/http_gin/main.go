package main

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	goskema "github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
	ginmw "github.com/reoring/goskema/middleware/gin"
)

// Demo model
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname,omitempty"`
}

func buildSchema() goskema.Schema[User] {
	return g.ObjectOf[User]().
		Field("id", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required().
		// presence demo: default applied when missing
		Field("nickname", g.StringOf[string]()).Default("anon").
		UnknownStrict().
		MustBind()
}

func main() {
	r := gin.Default()
	s := buildSchema()

	// Validate via middleware and echo typed value + presence
	r.POST("/validate", ginmw.ValidateJSON(s, goskema.ParseOpt{}), func(c *gin.Context) {
		dm, _ := ginmw.GetDecoded[User](c)
		canonical := preserveTopObjectUser(dm)
		c.JSON(http.StatusOK, gin.H{
			"ok":        true,
			"value":     dm.Value,
			"presence":  dm.Presence,
			"canonical": canonical,
		})
	})

	// Expose JSON Schema
	r.GET("/schema", func(c *gin.Context) {
		sch, err := s.JSONSchema()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "application/json")
		_ = json.NewEncoder(c.Writer).Encode(sch)
	})

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	_ = r.Run(":8081")
}

// preserveTopObjectUser applies the same semantics as EncodePreservingObject for top-level struct fields.
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
