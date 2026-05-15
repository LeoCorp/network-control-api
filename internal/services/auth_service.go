package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"Network-control-api/internal/auth"
	"Network-control-api/internal/models"
	"Network-control-api/internal/repositories"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid role")
)

type AuthService struct {
	users repositories.UserRepository
	jwt   *auth.JWTService
}

func NewAuthService(users repositories.UserRepository, jwt *auth.JWTService) *AuthService {
	return &AuthService{
		users: users,
		jwt:   jwt,
	}
}

type RegisterInput struct {
	Email    string
	Password string
	Role     string
}

type AuthResult struct {
	Token string
	User  *models.User
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	role := input.Role
	if role == "" {
		role = models.RoleViewer
	}
	if !models.IsValidRole(role) {
		return nil, ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.jwt.Generate(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwt.Generate(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{
		Token: token,
		User:  user,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
