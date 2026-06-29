package gateway

import (
	"crypto/subtle"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/mmtaee/ocserv-dashboard/api/pkg/routing/middlewares"
)

func tokenMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			expected := strings.TrimSpace(os.Getenv("GATEWAY_API_TOKEN"))
			if expected == "" {
				return middlewares.UnauthorizedError(c, "gateway API token is not configured")
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return middlewares.UnauthorizedError(c, "missing or invalid Authorization header")
			}

			actual := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
				return middlewares.UnauthorizedError(c, "invalid gateway API token")
			}

			return next(c)
		}
	}
}
