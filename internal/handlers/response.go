package handlers

import (
	"Network-control-api/internal/httputil"
	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context, status int, message string) {
	httputil.Error(c, status, message)
}
