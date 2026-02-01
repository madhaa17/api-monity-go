package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Asset struct {
	ID        int64           `gorm:"primaryKey" json:"-"`
	UUID      string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID    int64           `gorm:"index" json:"-"`
	Name      string          `json:"name"`
	Type      AssetType       `gorm:"type:asset_type" json:"type"`
	Quantity  decimal.Decimal `gorm:"type:decimal(20,8)" json:"quantity"`
	Symbol    *string         `json:"symbol,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type AssetPriceHistory struct {
	ID         int64           `gorm:"primaryKey" json:"-"`
	UUID       string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	AssetID    int64           `gorm:"index" json:"-"`
	Price      decimal.Decimal `gorm:"type:decimal(20,8)" json:"price"`
	Source     string          `json:"source"`
	RecordedAt time.Time       `json:"recordedAt"`
}
