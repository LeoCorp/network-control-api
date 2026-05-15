package httputil

import "github.com/gin-gonic/gin"

type errorResponse struct {
	Error string `json:"error"`
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Error: message})
}
