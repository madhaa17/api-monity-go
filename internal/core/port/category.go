package port

import (
	"context"
	"monity/internal/models"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	// CreateDefaults inserts models.DefaultCategories for the user, skipping names that already exist.
	CreateDefaults(ctx context.Context, userID int64) error
	GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Category, error)
	// GetByName matches name case-insensitively within the user's categories of the given type.
	GetByName(ctx context.Context, userID int64, categoryType models.CategoryType, name string) (*models.Category, error)
	// ListByUserID returns the user's categories ordered by type then name; nil categoryType means all types.
	ListByUserID(ctx context.Context, userID int64, categoryType *models.CategoryType) ([]models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id int64) error
	// CountUsage returns how many incomes and expenses reference the category.
	CountUsage(ctx context.Context, id int64) (int64, error)
	// Reassign moves every income and expense from one category to another.
	Reassign(ctx context.Context, fromID, toID int64) error
}

type CategoryService interface {
	ListCategories(ctx context.Context, userID int64, categoryType *models.CategoryType) ([]models.Category, error)
	GetCategory(ctx context.Context, userID int64, uuid string) (*models.Category, error)
	CreateCategory(ctx context.Context, userID int64, req CreateCategoryRequest) (*models.Category, error)
	UpdateCategory(ctx context.Context, userID int64, uuid string, req UpdateCategoryRequest) (*models.Category, error)
	// DeleteCategory removes a category. If it is still in use, reassignTo (a category UUID of the
	// same type) is required and its transactions are moved there first.
	DeleteCategory(ctx context.Context, userID int64, uuid string, reassignTo string) error
}

type CreateCategoryRequest struct {
	Type  models.CategoryType `json:"type"`
	Name  string              `json:"name"`
	Color *string             `json:"color,omitempty"`
	Icon  *string             `json:"icon,omitempty"`
}

// UpdateCategoryRequest changes only the fields that are set. An empty Color or Icon clears it.
// The type of a category cannot be changed.
type UpdateCategoryRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}
