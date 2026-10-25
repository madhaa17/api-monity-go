package models

import "time"

type CategoryType string

const (
	CategoryTypeIncome  CategoryType = "INCOME"
	CategoryTypeExpense CategoryType = "EXPENSE"
)

func (t CategoryType) Valid() bool {
	return t == CategoryTypeIncome || t == CategoryTypeExpense
}

// CategoryOther is the default category name that incomes fall back to when none is given.
const CategoryOther = "Other"

// DefaultCategories are created for every new user. Keep in sync with migrations/007_categories.up.sql.
var DefaultCategories = map[CategoryType][]string{
	CategoryTypeExpense: {"Food", "Transport", "Housing", "Utilities", "Health", "Entertainment", "Shopping", CategoryOther},
	CategoryTypeIncome:  {"Salary", "Bonus", "Business", "Investment", "Gift", CategoryOther},
}

type Category struct {
	ID        int64        `gorm:"primaryKey" json:"-"`
	UUID      string       `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID    int64        `gorm:"index" json:"-"`
	Type      CategoryType `gorm:"type:category_type" json:"type"`
	Name      string       `json:"name"`
	Color     *string      `json:"color,omitempty"`
	Icon      *string      `json:"icon,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}
