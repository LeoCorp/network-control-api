package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"Network-control-api/internal/auth"
	"Network-control-api/internal/httputil"
)

const (
	ContextUserIDKey = "userID"
	ContextEmailKey  = "userEmail"
	ContextRoleKey   = "userRole"
)

func JWTAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			httputil.Error(c, http.StatusUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httputil.Error(c, http.StatusUnauthorized, "invalid authorization header format")
			c.Abort()
			return
		}

		claims, err := jwtService.Parse(parts[1])
		if err != nil {
			httputil.Error(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextEmailKey, claims.Email)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

func GetAuthUser(c *gin.Context) (uuid.UUID, string, string, bool) {
	userID, ok := c.Get(ContextUserIDKey)
	if !ok {
		return uuid.Nil, "", "", false
	}

	email, _ := c.Get(ContextEmailKey)
	role, _ := c.Get(ContextRoleKey)

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, "", "", false
	}

	emailStr, _ := email.(string)
	roleStr, _ := role.(string)

	return id, emailStr, roleStr, true
}
