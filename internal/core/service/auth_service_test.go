package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"monity/internal/config"
	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/cache"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	users map[int64]*models.User
}

func (r *fakeUserRepo) Create(ctx context.Context, user *models.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			copied := *u
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	copied := *u
	return &copied, nil
}

func (r *fakeUserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string, changedAt time.Time) error {
	u, ok := r.users[id]
	if !ok {
		return port.ErrNotFound
	}
	u.Password = passwordHash
	u.PasswordChangedAt = &changedAt
	return nil
}

var testJwtConfig = config.JwtConfig{
	Secret:            "access-secret",
	ExpirationTime:    "1h",
	RefreshSecret:     "refresh-secret",
	RefreshExpiration: "168h",
}

func newTestAuthService(t *testing.T, password string) (port.AuthService, *fakeUserRepo, cache.Cache) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeUserRepo{users: map[int64]*models.User{
		testUser: {ID: testUser, UUID: "user-uuid", Email: "a@example.com", Password: string(hash), Role: models.UserRoleUser},
	}}
	c := cache.NewMemoryCache()
	return NewAuthService(repo, nil, nil, &config.Config{Jwt: testJwtConfig}, c), repo, c
}

// signedRefreshToken returns a refresh token for testUser issued at issuedAt.
func signedRefreshToken(t *testing.T, issuedAt time.Time) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  testUser,
		"uuid": "user-uuid",
		"jti":  "old",
		"iat":  issuedAt.Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testJwtConfig.RefreshSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("validation", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		change := func(current, next string) error {
			_, err := svc.ChangePassword(ctx, testUser, port.ChangePasswordRequest{CurrentPassword: current, NewPassword: next})
			return err
		}
		wantValidation(t, change("", "newpass123"), "current_password and new_password are required")
		wantValidation(t, change("wrongpass1", "newpass123"), "current password is incorrect")
		wantValidation(t, change("oldpass123", "short1"), "password must be at least 8 characters")
		wantValidation(t, change("oldpass123", "oldpass123"), "new password must be different from the current password")
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		_, err := svc.ChangePassword(ctx, testUser+1, port.ChangePasswordRequest{CurrentPassword: "oldpass123", NewPassword: "newpass123"})
		if !errors.Is(err, port.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		svc, repo, c := newTestAuthService(t, "oldpass123")
		oldRefresh := signedRefreshToken(t, time.Now().Add(-time.Minute))

		resp, err := svc.ChangePassword(ctx, testUser, port.ChangePasswordRequest{CurrentPassword: "oldpass123", NewPassword: "newpass123"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Token == "" || resp.RefreshToken == "" {
			t.Fatal("want a fresh token pair")
		}

		stored := repo.users[testUser]
		if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("newpass123")) != nil {
			t.Fatal("new password not stored")
		}
		if stored.PasswordChangedAt == nil {
			t.Fatal("password_changed_at not set")
		}
		raw, _ := c.Get(ctx, passwordChangedKeyPrefix+strconv.FormatInt(testUser, 10))
		if string(raw) != strconv.FormatInt(stored.PasswordChangedAt.Unix(), 10) {
			t.Fatalf("cached change time %q, want %d", raw, stored.PasswordChangedAt.Unix())
		}

		if _, err := svc.Login(ctx, port.LoginRequest{Email: "a@example.com", Password: "oldpass123"}); err == nil {
			t.Fatal("old password still logs in")
		}
		if _, err := svc.Login(ctx, port.LoginRequest{Email: "a@example.com", Password: "newpass123"}); err != nil {
			t.Fatalf("new password does not log in: %v", err)
		}

		if _, err := svc.Refresh(ctx, oldRefresh); err == nil {
			t.Fatal("refresh token issued before the change still works")
		}
		if _, err := svc.Refresh(ctx, resp.RefreshToken); err != nil {
			t.Fatalf("new refresh token rejected: %v", err)
		}
	})
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	ctx := context.Background()
	login := func(t *testing.T, svc port.AuthService) *port.AuthResponse {
		t.Helper()
		resp, err := svc.Login(ctx, port.LoginRequest{Email: "a@example.com", Password: "oldpass123"})
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	t.Run("with access token", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		session := login(t, svc)
		if err := svc.Logout(ctx, session.Token, session.RefreshToken); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Refresh(ctx, session.RefreshToken); err == nil {
			t.Fatal("refresh token still works after logout")
		}
	})

	t.Run("refresh token only", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		session := login(t, svc)
		if err := svc.Logout(ctx, "", session.RefreshToken); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Refresh(ctx, session.RefreshToken); err == nil {
			t.Fatal("refresh token still works after logout")
		}
	})

	t.Run("other sessions keep working", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		first, second := login(t, svc), login(t, svc)
		if err := svc.Logout(ctx, first.Token, first.RefreshToken); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Refresh(ctx, second.RefreshToken); err != nil {
			t.Fatalf("other session's refresh token rejected: %v", err)
		}
	})
}

func TestRefreshRotatesRefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("old token works within the grace period", func(t *testing.T) {
		svc, _, _ := newTestAuthService(t, "oldpass123")
		old := signedRefreshToken(t, time.Now())
		first, err := svc.Refresh(ctx, old)
		if err != nil {
			t.Fatal(err)
		}
		if first.RefreshToken == old {
			t.Fatal("want a new refresh token")
		}
		// A second tab refreshing at the same moment with the same token.
		if _, err := svc.Refresh(ctx, old); err != nil {
			t.Fatalf("concurrent refresh rejected: %v", err)
		}
		if _, err := svc.Refresh(ctx, first.RefreshToken); err != nil {
			t.Fatalf("new refresh token rejected: %v", err)
		}
	})

	t.Run("old token is rejected after the grace period", func(t *testing.T) {
		svc, _, c := newTestAuthService(t, "oldpass123")
		old := signedRefreshToken(t, time.Now())
		firstUse := time.Now().Add(-refreshReuseGrace - time.Second)
		if err := c.Set(ctx, rotatedKeyPrefix+"old", []byte(strconv.FormatInt(firstUse.Unix(), 10)), time.Hour); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Refresh(ctx, old); err == nil {
			t.Fatal("exchanged refresh token still works after the grace period")
		}
	})
}
