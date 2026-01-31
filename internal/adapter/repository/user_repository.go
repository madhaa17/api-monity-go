package repository

import (
	"context"
	"errors"
	"fmt"

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

	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return fmt.Errorf("create user: %w", result.Error)
	}
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
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
	result := r.db.WithContext(ctx).First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", result.Error)
	}
	return &user, nil
}
