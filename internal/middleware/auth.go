// Package middleware holds Gin middlewares.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/your-username/agnos-middleware/internal/auth"
)

const claimsKey = "claims"

// RequireAuth rejects requests without a valid "Authorization: Bearer <jwt>" header and stores
// the token claims in the request context.
func RequireAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		scheme, tokenString, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(tokenString) == "" {
			abort(c, "missing or malformed Authorization header")
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(tokenString))
		if err != nil {
			abort(c, err.Error())
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

// ClaimsFrom returns the claims set by RequireAuth.
func ClaimsFrom(c *gin.Context) (*auth.Claims, bool) {
	v, ok := c.Get(claimsKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	return claims, ok
}

func abort(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}
