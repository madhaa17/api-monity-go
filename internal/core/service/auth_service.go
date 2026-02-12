package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/cache"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const revokedKeyPrefix = "revoked:"

type AuthService struct {
	repo  port.UserRepository
	cfg   *config.Config
	cache cache.Cache
}

func NewAuthService(repo port.UserRepository, cfg *config.Config, c cache.Cache) port.AuthService {
	return &AuthService{
		repo:  repo,
		cfg:   cfg,
		cache: c,
	}
}

func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
	refreshToken, err := s.generateRefreshToken(newUser)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &port.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         newUser,
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
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &port.AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*port.AuthResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token required")
	}
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Jwt.RefreshSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired refresh token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid refresh token claims")
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return nil, errors.New("invalid refresh token payload")
	}
	userID := int64(sub)
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	accessToken, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	return &port.AuthResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, userID int64) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, accessToken string, refreshToken string) error {
	if s.cache == nil {
		return nil
	}
	revoke := func(tokenString string, secret []byte) {
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return
		}
		jti, _ := claims["jti"].(string)
		if jti == "" {
			return
		}
		exp, _ := claims["exp"].(float64)
		ttl := time.Until(time.Unix(int64(exp), 0))
		if ttl <= 0 {
			return
		}
		_ = s.cache.Set(ctx, revokedKeyPrefix+jti, []byte("1"), ttl)
	}
	revoke(accessToken, []byte(s.cfg.Jwt.Secret))
	if refreshToken != "" {
		revoke(refreshToken, []byte(s.cfg.Jwt.RefreshSecret))
	}
	return nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	duration, err := time.ParseDuration(s.cfg.Jwt.ExpirationTime)
	if err != nil {
		duration = time.Hour // Default fallback
	}
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"uuid":  user.UUID,
		"email": user.Email,
		"role":  user.Role,
		"jti":   jti,
		"exp":   time.Now().Add(duration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Jwt.Secret))
}

func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
	duration, err := time.ParseDuration(s.cfg.Jwt.RefreshExpiration)
	if err != nil {
		duration = 168 * time.Hour // 7 days default
	}
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"uuid": user.UUID,
		"jti":  jti,
		"exp":  time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Jwt.RefreshSecret))
}
