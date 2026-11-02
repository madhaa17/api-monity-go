package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/cache"
	"monity/internal/pkg/validation"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const revokedKeyPrefix = "revoked:"

// passwordChangedKeyPrefix + user ID holds the Unix time of the user's last password change. The
// auth middleware rejects access tokens issued before it without a database lookup.
const passwordChangedKeyPrefix = "pwchanged:"

// rotatedKeyPrefix + refresh token ID holds the Unix time the refresh token was first exchanged.
// Each refresh issues a new refresh token, and the old one stops working after refreshReuseGrace.
const rotatedKeyPrefix = "rotated:"

// refreshReuseGrace is how long an exchanged refresh token keeps working, so requests that refresh
// at the same time (several tabs, parallel retries) all succeed instead of signing the user out.
const refreshReuseGrace = 30 * time.Second

type AuthService struct {
	repo         port.UserRepository
	categoryRepo port.CategoryRepository
	tx           port.Transactor
	cfg          *config.Config
	cache        cache.Cache
}

func NewAuthService(repo port.UserRepository, categoryRepo port.CategoryRepository, tx port.Transactor, cfg *config.Config, c cache.Cache) port.AuthService {
	return &AuthService{
		repo:         repo,
		categoryRepo: categoryRepo,
		tx:           tx,
		cfg:          cfg,
		cache:        c,
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
	if !validation.ValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
	if ok, msg := validation.ValidPassword(req.Password); !ok {
		return nil, errors.New(msg)
	}
	if err := validation.CheckMaxLen(req.Email, validation.MaxEmailLen); err != nil {
		return nil, fmt.Errorf("email %w", err)
	}
	if err := validation.CheckMaxLen(req.Name, validation.MaxNameLen); err != nil {
		return nil, fmt.Errorf("name %w", err)
	}
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

	// The user and their default income/expense categories are created together.
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, newUser); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := s.categoryRepo.CreateDefaults(ctx, newUser.ID); err != nil {
			return fmt.Errorf("create default categories: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
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
	if !validation.ValidEmail(req.Email) {
		return nil, errors.New("invalid email format")
	}
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
	revoked, err := s.isRevoked(ctx, claims)
	if err != nil {
		return nil, fmt.Errorf("check refresh token revocation: %w", err)
	}
	if revoked {
		return nil, errors.New("invalid or expired refresh token")
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
	if issuedBeforePasswordChange(claims, user.PasswordChangedAt) {
		return nil, errors.New("invalid or expired refresh token")
	}
	usable, err := s.markRotated(ctx, claims)
	if err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}
	if !usable {
		return nil, errors.New("invalid or expired refresh token")
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

// Logout revokes the given tokens; either may be empty. Tokens that are invalid or already expired
// are skipped, since they are unusable anyway.
func (s *AuthService) Logout(ctx context.Context, accessToken string, refreshToken string) error {
	if s.cache == nil {
		return nil
	}
	revoke := func(tokenString string, secret []byte) error {
		if tokenString == "" {
			return nil
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			return nil
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil
		}
		jti, _ := claims["jti"].(string)
		if jti == "" {
			return nil
		}
		exp, _ := claims["exp"].(float64)
		ttl := time.Until(time.Unix(int64(exp), 0))
		if ttl <= 0 {
			return nil
		}
		return s.cache.Set(ctx, revokedKeyPrefix+jti, []byte("1"), ttl)
	}
	if err := revoke(accessToken, []byte(s.cfg.Jwt.Secret)); err != nil {
		return fmt.Errorf("revoke access token: %w", err)
	}
	if err := revoke(refreshToken, []byte(s.cfg.Jwt.RefreshSecret)); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// markRotated records the first exchange of a refresh token and reports whether it may still be
// exchanged: on first use, and for refreshReuseGrace after it.
func (s *AuthService) markRotated(ctx context.Context, claims jwt.MapClaims) (bool, error) {
	if s.cache == nil {
		return true, nil
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return true, nil
	}
	key := rotatedKeyPrefix + jti
	raw, err := s.cache.Get(ctx, key)
	if err != nil && !errors.Is(err, cache.ErrMiss) {
		return false, err
	}
	now := time.Now()
	if len(raw) > 0 {
		firstUse, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil {
			return false, fmt.Errorf("parse rotation time: %w", err)
		}
		return now.Sub(time.Unix(firstUse, 0)) <= refreshReuseGrace, nil
	}
	// Kept until the token expires, after which it is rejected anyway.
	exp, _ := claims["exp"].(float64)
	ttl := time.Until(time.Unix(int64(exp), 0))
	if err := s.cache.Set(ctx, key, []byte(strconv.FormatInt(now.Unix(), 10)), ttl); err != nil {
		return false, err
	}
	return true, nil
}

// isRevoked reports whether the token with these claims was revoked by Logout.
func (s *AuthService) isRevoked(ctx context.Context, claims jwt.MapClaims) (bool, error) {
	if s.cache == nil {
		return false, nil
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return false, nil
	}
	revoked, err := s.cache.Get(ctx, revokedKeyPrefix+jti)
	if err != nil && !errors.Is(err, cache.ErrMiss) {
		return false, err
	}
	return len(revoked) > 0, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int64, req port.ChangePasswordRequest) (*port.AuthResponse, error) {
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return nil, port.NewValidationError("current_password and new_password are required")
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, port.ErrNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return nil, port.NewValidationError("current password is incorrect")
	}
	if ok, msg := validation.ValidPassword(req.NewPassword); !ok {
		return nil, port.NewValidationError(msg)
	}
	if req.NewPassword == req.CurrentPassword {
		return nil, port.NewValidationError("new password must be different from the current password")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	// Truncated to whole seconds so it compares exactly with the tokens' "iat" claim.
	changedAt := time.Now().Truncate(time.Second)
	if err := s.repo.UpdatePassword(ctx, user.ID, string(hashedPassword), changedAt); err != nil {
		return nil, fmt.Errorf("update password: %w", err)
	}
	user.Password = string(hashedPassword)
	user.PasswordChangedAt = &changedAt

	if s.cache != nil {
		key := passwordChangedKeyPrefix + strconv.FormatInt(user.ID, 10)
		value := []byte(strconv.FormatInt(changedAt.Unix(), 10))
		// Kept as long as the longest-lived token, after which every older token has expired anyway.
		if err := s.cache.Set(ctx, key, value, s.refreshTokenTTL()); err != nil {
			return nil, fmt.Errorf("cache password change: %w", err)
		}
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

// issuedBeforePasswordChange reports whether a token with these claims predates the user's last
// password change. Tokens without "iat" were issued before it existed and count as older.
func issuedBeforePasswordChange(claims jwt.MapClaims, changedAt *time.Time) bool {
	if changedAt == nil {
		return false
	}
	iat, _ := claims["iat"].(float64)
	return int64(iat) < changedAt.Unix()
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
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(duration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Jwt.Secret))
}

func (s *AuthService) refreshTokenTTL() time.Duration {
	duration, err := time.ParseDuration(s.cfg.Jwt.RefreshExpiration)
	if err != nil {
		return 168 * time.Hour // 7 days default
	}
	return duration
}

func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
	duration := s.refreshTokenTTL()
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"uuid": user.UUID,
		"jti":  jti,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(duration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Jwt.RefreshSecret))
}
