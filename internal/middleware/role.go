package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/httputil"
)

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(c *gin.Context) {
		_, _, role, ok := GetAuthUser(c)
		if !ok || !allowed[role] {
			httputil.Error(c, http.StatusForbidden, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}
