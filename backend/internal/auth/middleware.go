package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserID   = "user_id"
	CtxUsername = "username"
)

// Middleware validates the Bearer JWT on every protected route.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		claims, err := ValidateJWT(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string   { return c.GetString(CtxUserID) }
func GetUsername(c *gin.Context) string { return c.GetString(CtxUsername) }
