package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"monity/internal/core/port"
	"monity/internal/models"
	"monity/internal/pkg/validation"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) port.CategoryRepository {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) Create(ctx context.Context, category *models.Category) error {
	if err := conn(ctx, r.db).Create(category).Error; err != nil {
		return fmt.Errorf("create category: %w", translateUniqueViolation(err))
	}
	return nil
}

func (r *CategoryRepo) CreateDefaults(ctx context.Context, userID int64) error {
	var categories []models.Category
	for categoryType, names := range models.DefaultCategories {
		for _, name := range names {
			categories = append(categories, models.Category{UserID: userID, Type: categoryType, Name: name})
		}
	}
	err := conn(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(&categories).Error
	if err != nil {
		return fmt.Errorf("create default categories: %w", err)
	}
	return nil
}

func (r *CategoryRepo) GetByUUID(ctx context.Context, uuid string, userID int64) (*models.Category, error) {
	if !validation.ValidUUID(uuid) {
		return nil, nil // a malformed UUID cannot match; avoid a database cast error
	}
	return r.first(conn(ctx, r.db).Where("uuid = ? AND user_id = ?", uuid, userID))
}

func (r *CategoryRepo) GetByName(ctx context.Context, userID int64, categoryType models.CategoryType, name string) (*models.Category, error) {
	return r.first(conn(ctx, r.db).Where("user_id = ? AND type = ? AND lower(name) = ?", userID, categoryType, strings.ToLower(name)))
}

func (r *CategoryRepo) first(q *gorm.DB) (*models.Category, error) {
	var category models.Category
	if err := q.First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &category, nil
}

func (r *CategoryRepo) ListByUserID(ctx context.Context, userID int64, categoryType *models.CategoryType) ([]models.Category, error) {
	q := conn(ctx, r.db).Where("user_id = ?", userID)
	if categoryType != nil {
		q = q.Where("type = ?", *categoryType)
	}
	var categories []models.Category
	if err := q.Order("type, lower(name)").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return categories, nil
}

func (r *CategoryRepo) Update(ctx context.Context, category *models.Category) error {
	if err := conn(ctx, r.db).Save(category).Error; err != nil {
		return fmt.Errorf("update category: %w", translateUniqueViolation(err))
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id int64) error {
	result := conn(ctx, r.db).Delete(&models.Category{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete category: %w", port.ErrNotFound)
	}
	return nil
}

func (r *CategoryRepo) CountUsage(ctx context.Context, id int64) (int64, error) {
	var count int64
	err := conn(ctx, r.db).Raw(
		"SELECT (SELECT count(*) FROM expenses WHERE category_id = ?) + (SELECT count(*) FROM incomes WHERE category_id = ?)",
		id, id,
	).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count category usage: %w", err)
	}
	return count, nil
}

func (r *CategoryRepo) Reassign(ctx context.Context, fromID, toID int64) error {
	db := conn(ctx, r.db)
	if err := db.Model(&models.Expense{}).Where("category_id = ?", fromID).Update("category_id", toID).Error; err != nil {
		return fmt.Errorf("reassign expenses: %w", err)
	}
	if err := db.Model(&models.Income{}).Where("category_id = ?", fromID).Update("category_id", toID).Error; err != nil {
		return fmt.Errorf("reassign incomes: %w", err)
	}
	return nil
}

// translateUniqueViolation maps a Postgres unique violation to port.ErrConflict.
func translateUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return port.ErrConflict
	}
	return err
}
