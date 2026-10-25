package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode/utf8"

	"monity/internal/core/port"
	"monity/internal/models"
)

const (
	maxCategoryNameLen = 50
	maxCategoryIconLen = 50
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type CategoryService struct {
	repo port.CategoryRepository
	tx   port.Transactor
}

func NewCategoryService(repo port.CategoryRepository, tx port.Transactor) port.CategoryService {
	return &CategoryService{repo: repo, tx: tx}
}

func (s *CategoryService) ListCategories(ctx context.Context, userID int64, categoryType *models.CategoryType) ([]models.Category, error) {
	if categoryType != nil && !categoryType.Valid() {
		return nil, port.NewValidationError("type must be INCOME or EXPENSE")
	}
	categories, err := s.repo.ListByUserID(ctx, userID, categoryType)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	if categories == nil {
		categories = []models.Category{}
	}
	return categories, nil
}

func (s *CategoryService) GetCategory(ctx context.Context, userID int64, uuid string) (*models.Category, error) {
	category, err := s.repo.GetByUUID(ctx, uuid, userID)
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	if category == nil {
		return nil, fmt.Errorf("category: %w", port.ErrNotFound)
	}
	return category, nil
}

func (s *CategoryService) CreateCategory(ctx context.Context, userID int64, req port.CreateCategoryRequest) (*models.Category, error) {
	if !req.Type.Valid() {
		return nil, port.NewValidationError("type must be INCOME or EXPENSE")
	}
	name, err := normalizeCategoryName(req.Name)
	if err != nil {
		return nil, err
	}
	color, err := normalizeCategoryColor(req.Color)
	if err != nil {
		return nil, err
	}
	icon, err := normalizeCategoryIcon(req.Icon)
	if err != nil {
		return nil, err
	}

	category := &models.Category{UserID: userID, Type: req.Type, Name: name, Color: color, Icon: icon}
	if err := s.repo.Create(ctx, category); err != nil {
		return nil, categoryWriteError(err)
	}
	slog.Info("category_created", "user_id", userID, "type", category.Type, "name", category.Name)
	return category, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, userID int64, uuid string, req port.UpdateCategoryRequest) (*models.Category, error) {
	category, err := s.GetCategory(ctx, userID, uuid)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		if category.Name, err = normalizeCategoryName(*req.Name); err != nil {
			return nil, err
		}
	}
	if req.Color != nil {
		if category.Color, err = normalizeCategoryColor(req.Color); err != nil {
			return nil, err
		}
	}
	if req.Icon != nil {
		if category.Icon, err = normalizeCategoryIcon(req.Icon); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Update(ctx, category); err != nil {
		return nil, categoryWriteError(err)
	}
	return category, nil
}

func (s *CategoryService) DeleteCategory(ctx context.Context, userID int64, uuid string, reassignTo string) error {
	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		category, err := s.GetCategory(ctx, userID, uuid)
		if err != nil {
			return err
		}

		if strings.TrimSpace(reassignTo) != "" {
			target, err := s.repo.GetByUUID(ctx, reassignTo, userID)
			if err != nil {
				return fmt.Errorf("get target category: %w", err)
			}
			if target == nil {
				return port.NewValidationError("reassign_to category not found")
			}
			if target.ID == category.ID {
				return port.NewValidationError("reassign_to must be a different category")
			}
			if target.Type != category.Type {
				return port.NewValidationError("reassign_to must be a category of the same type")
			}
			if err := s.repo.Reassign(ctx, category.ID, target.ID); err != nil {
				return fmt.Errorf("reassign category: %w", err)
			}
		} else {
			used, err := s.repo.CountUsage(ctx, category.ID)
			if err != nil {
				return fmt.Errorf("count category usage: %w", err)
			}
			if used > 0 {
				return port.NewValidationError(fmt.Sprintf(
					"category is used by %d transaction(s); pass reassign_to with another category to move them", used))
			}
		}

		if err := s.repo.Delete(ctx, category.ID); err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		slog.Info("category_deleted", "user_id", userID, "uuid", uuid, "reassign_to", reassignTo)
		return nil
	})
}

// resolveCategory finds the user's category of categoryType, by UUID when given, otherwise by
// name (case-insensitive). It returns a validation error when nothing matches.
func resolveCategory(ctx context.Context, repo port.CategoryRepository, userID int64, categoryType models.CategoryType, uuid, name string) (*models.Category, error) {
	var (
		category *models.Category
		err      error
	)
	switch {
	case strings.TrimSpace(uuid) != "":
		category, err = repo.GetByUUID(ctx, uuid, userID)
	case strings.TrimSpace(name) != "":
		category, err = repo.GetByName(ctx, userID, categoryType, strings.TrimSpace(name))
	default:
		return nil, port.NewValidationError("categoryUuid is required")
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	if category == nil {
		return nil, port.NewValidationError("category not found")
	}
	if category.Type != categoryType {
		return nil, port.NewValidationError(fmt.Sprintf("category must be an %s category", categoryType))
	}
	return category, nil
}

func categoryWriteError(err error) error {
	if errors.Is(err, port.ErrConflict) {
		return port.NewValidationError("a category with this name already exists")
	}
	return fmt.Errorf("save category: %w", err)
}

func normalizeCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", port.NewValidationError("name is required")
	}
	if utf8.RuneCountInString(name) > maxCategoryNameLen {
		return "", port.NewValidationError(fmt.Sprintf("name must be at most %d characters", maxCategoryNameLen))
	}
	return name, nil
}

// normalizeCategoryColor returns nil for an empty color (clearing it) and validates #RRGGBB otherwise.
func normalizeCategoryColor(color *string) (*string, error) {
	if color == nil || strings.TrimSpace(*color) == "" {
		return nil, nil
	}
	c := strings.TrimSpace(*color)
	if !hexColorRegex.MatchString(c) {
		return nil, port.NewValidationError("color must be a hex color like #22C55E")
	}
	return &c, nil
}

// normalizeCategoryIcon returns nil for an empty icon (clearing it) and checks the length otherwise.
func normalizeCategoryIcon(icon *string) (*string, error) {
	if icon == nil || strings.TrimSpace(*icon) == "" {
		return nil, nil
	}
	i := strings.TrimSpace(*icon)
	if utf8.RuneCountInString(i) > maxCategoryIconLen {
		return nil, port.NewValidationError(fmt.Sprintf("icon must be at most %d characters", maxCategoryIconLen))
	}
	return &i, nil
}
