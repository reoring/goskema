package echomw

import (
	"net/http"

	"github.com/labstack/echo/v4"
	goskema "github.com/reoring/goskema"
	"github.com/reoring/goskema/middleware"
)

// ValidateJSON parses request JSON via schema s, stores Decoded[T] in context on success,
// or returns 400 with Issues when validation fails.
func ValidateJSON[T any](s goskema.Schema[T], opt goskema.ParseOpt) echo.MiddlewareFunc {
	if opt.Strictness.OnDuplicateKey == 0 && !opt.Presence.Collect {
		opt = middleware.DefaultParseOpt()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			dm, err := goskema.ParseFromWithMeta(c.Request().Context(), s, goskema.JSONReader(c.Request().Body), opt)
			if err != nil {
				if iss, ok := goskema.AsIssues(err); ok {
					return c.JSON(http.StatusBadRequest, middleware.ErrorPayload(iss))
				}
				return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
			}
			ctx := middleware.ContextWithDecoded(c.Request().Context(), dm)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// GetDecoded fetches Decoded[T] from echo.Context.
func GetDecoded[T any](c echo.Context) (goskema.Decoded[T], bool) {
	return middleware.DecodedFromContext[T](c.Request().Context())
}
