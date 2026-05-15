package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
	"Network-control-api/internal/services"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Create a new user account and return a JWT token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"Register payload"
//	@Success		201		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.auth.Register(c.Request.Context(), services.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrDuplicateEmail):
			Error(c, http.StatusConflict, "email already registered")
		case errors.Is(err, services.ErrInvalidRole):
			Error(c, http.StatusBadRequest, "invalid role")
		default:
			Error(c, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: result.Token,
		User:  toUserResponse(result.User),
	})
}

// Login godoc
//
//	@Summary		Login
//	@Description	Authenticate a user and return a JWT token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"Login payload"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			Error(c, http.StatusUnauthorized, "invalid credentials")
			return
		}
		Error(c, http.StatusInternalServerError, "failed to login")
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: result.Token,
		User:  toUserResponse(result.User),
	})
}

func toUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}
