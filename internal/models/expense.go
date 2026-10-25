package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Expense struct {
	ID      int64           `gorm:"primaryKey" json:"-"`
	UUID    string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID  int64           `gorm:"index" json:"-"`
	AssetID int64           `gorm:"index" json:"-"`
	Amount  decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`

	// CategoryID references a user-defined EXPENSE category. CategoryUUID and CategoryName are
	// read-only, filled by the repository's join; "category" keeps the old field name for clients.
	CategoryID   int64  `gorm:"index" json:"-"`
	CategoryUUID string `gorm:"->;column:category_uuid" json:"categoryUuid"`
	CategoryName string `gorm:"->;column:category_name" json:"category"`

	Note      *string   `json:"note,omitempty"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"createdAt"`

	// The CASH asset this expense draws from; read-only, filled by the repository's join.
	AssetUUID string `gorm:"->;column:asset_uuid" json:"assetUuid"`
	AssetName string `gorm:"->;column:asset_name" json:"assetName"`
}
