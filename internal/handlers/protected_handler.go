package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/middleware"
)

type ProtectedHandler struct{}

func NewProtectedHandler() *ProtectedHandler {
	return &ProtectedHandler{}
}

type protectedTestResponse struct {
	Message string       `json:"message"`
	User    UserResponse `json:"user"`
}

func (h *ProtectedHandler) Test(c *gin.Context) {
	userID, email, role, ok := middleware.GetAuthUser(c)
	if !ok {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	c.JSON(http.StatusOK, protectedTestResponse{
		Message: "protected route accessible",
		User: UserResponse{
			ID:    userID.String(),
			Email: email,
			Role:  role,
		},
	})
}
