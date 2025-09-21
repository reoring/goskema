package ginmw

import (
	"net/http"

	"github.com/gin-gonic/gin"
	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/middleware"
)

// ValidateJSON parses the incoming JSON using schema s with opt (or DefaultParseOpt when zero value),
// stores Decoded[T] in the context, and on validation failure returns 400 with Issues payload.
func ValidateJSON[T any](s goskema.Schema[T], opt goskema.ParseOpt) gin.HandlerFunc {
	// merge defaults if caller passed zero
	if opt.Strictness.OnDuplicateKey == 0 && !opt.Presence.Collect {
		opt = middleware.DefaultParseOpt()
	}
	return func(c *gin.Context) {
		dm, err := goskema.ParseFromWithMeta(c.Request.Context(), s, goskema.JSONReader(c.Request.Body), opt)
		if err != nil {
			if iss, ok := goskema.AsIssues(err); ok {
				c.JSON(http.StatusBadRequest, middleware.ErrorPayload(iss))
				c.Abort()
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		// store decoded in request context
		c.Request = c.Request.WithContext(middleware.ContextWithDecoded(c.Request.Context(), dm))
		c.Next()
	}
}

// GetDecoded fetches Decoded[T] from gin.Context.
func GetDecoded[T any](c *gin.Context) (goskema.Decoded[T], bool) {
	return middleware.DecodedFromContext[T](c.Request.Context())
}
