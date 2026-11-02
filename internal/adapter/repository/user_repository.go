package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monity/internal/core/port"
	"monity/internal/models"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) port.UserRepository {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	// Default role if not set
	if user.Role == "" {
		user.Role = models.UserRoleUser
	}

	result := conn(ctx, r.db).Create(user)
	if result.Error != nil {
		return fmt.Errorf("create user: %w", result.Error)
	}
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	result := conn(ctx, r.db).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil if not found
		}
		return nil, fmt.Errorf("get user by email: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	var user models.User
	result := conn(ctx, r.db).First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", result.Error)
	}
	return &user, nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string, changedAt time.Time) error {
	result := conn(ctx, r.db).Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"password":            passwordHash,
		"password_changed_at": changedAt,
		"updated_at":          changedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update user password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return port.ErrNotFound
	}
	return nil
}
