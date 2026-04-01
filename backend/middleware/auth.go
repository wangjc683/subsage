package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token format"})
			}

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Set("username", claims["sub"])
			}

			return next(c)
		}
	}
}

// TokenAuthMiddleware reads the current API token from DB on each request.
// This ensures regenerated tokens take effect immediately without restart.
// It also logs each authenticated call to agent_logs for activity tracking.
func TokenAuthMiddleware(db *sql.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read current valid token from DB
			var validToken string
			if err := db.QueryRow("SELECT value FROM settings WHERE key = 'api_token'").Scan(&validToken); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "API token not configured"})
			}

			// Check header first, then query param
			token := c.Request().Header.Get("X-API-Token")
			if token == "" {
				token = c.QueryParam("token")
			}
			if token != validToken {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid API token"})
			}
			c.Set("auth_type", "api_token")

			// Log agent activity
			method := c.Request().Method
			path := c.Request().URL.Path
			ua := c.Request().UserAgent()
			db.Exec("INSERT INTO agent_logs (method, path, user_agent) VALUES (?, ?, ?)", method, path, ua)
			// Auto-prune: keep only latest 200
			db.Exec("DELETE FROM agent_logs WHERE id NOT IN (SELECT id FROM agent_logs ORDER BY id DESC LIMIT 200)")

			return next(c)
		}
	}
}
