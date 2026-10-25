package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Income struct {
	ID      int64           `gorm:"primaryKey" json:"-"`
	UUID    string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID  int64           `gorm:"index" json:"-"`
	AssetID int64           `gorm:"index" json:"-"`
	Amount  decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`
	Source  string          `json:"source"`

	// CategoryID references a user-defined INCOME category. CategoryUUID and CategoryName are
	// read-only, filled by the repository's join.
	CategoryID   int64  `gorm:"index" json:"-"`
	CategoryUUID string `gorm:"->;column:category_uuid" json:"categoryUuid"`
	CategoryName string `gorm:"->;column:category_name" json:"category"`

	Note      *string   `json:"note,omitempty"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"createdAt"`

	// The CASH asset this income goes into; read-only, filled by the repository's join.
	AssetUUID string `gorm:"->;column:asset_uuid" json:"assetUuid"`
	AssetName string `gorm:"->;column:asset_name" json:"assetName"`
}
