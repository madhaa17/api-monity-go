package port

import (
	"context"
	"time"

	"monity/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	// UpdatePassword stores a new password hash and the time it changed; ErrNotFound if no such user.
	UpdatePassword(ctx context.Context, id int64, passwordHash string, changedAt time.Time) error
	// Add other methods as needed
}

type AuthService interface {
	Register(ctx context.Context, req RegistryRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error)
	GetMe(ctx context.Context, userID int64) (*models.User, error)
	Logout(ctx context.Context, accessToken string, refreshToken string) error
	// ChangePassword verifies the current password, stores the new one, invalidates every token
	// issued before the change and returns a fresh token pair for the caller's session.
	ChangePassword(ctx context.Context, userID int64, req ChangePasswordRequest) (*AuthResponse, error)
}

// DTOs for Service layer - arguably could be in models or service package but keeping interfaces together
type RegistryRequest struct {
	Email    string
	Password string
	Name     string
}

type LoginRequest struct {
	Email    string
	Password string
}

type ChangePasswordRequest struct {
	CurrentPassword string
	NewPassword     string
}

type AuthResponse struct {
	Token        string
	RefreshToken string
	User         *models.User
}
