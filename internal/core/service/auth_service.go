package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo port.UserRepository
	cfg  *config.Config
}

func NewAuthService(repo port.UserRepository, cfg *config.Config) port.AuthService {
	return &AuthService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, req port.RegistryRequest) (*port.AuthResponse, error) {
	// Check if user exists
	existingUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newUser := &models.User{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     &req.Name,
		Role:     models.UserRoleUser,
	}

	if err := s.repo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateToken(newUser)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &port.AuthResponse{
		Token: token,
		User:  newUser,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req port.LoginRequest) (*port.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &port.AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	duration, err := time.ParseDuration(s.cfg.Jwt.ExpirationTime)
	if err != nil {
		duration = time.Hour // Default fallback
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"uuid":  user.UUID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(duration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Jwt.Secret))
}
