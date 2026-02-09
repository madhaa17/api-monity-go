package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Expense struct {
	ID        int64           `gorm:"primaryKey" json:"-"`
	UUID      string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID    int64           `gorm:"index" json:"-"`
	AssetID   int64           `gorm:"index" json:"-"`
	Amount    decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`
	Category  ExpenseCategory `gorm:"type:expense_category" json:"category"`
	Note      *string         `json:"note,omitempty"`
	Date      time.Time       `json:"date"`
	CreatedAt time.Time       `json:"createdAt"`

	// Belongs-to: the CASH asset this expense draws from
	Asset *Asset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}
