package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Asset struct {
	ID        int64           `gorm:"primaryKey" json:"id"`
	UUID      string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID    int64           `gorm:"index" json:"userId"`
	Name      string          `json:"name"`
	Type      AssetType       `gorm:"type:asset_type" json:"type"`
	Quantity  decimal.Decimal `gorm:"type:decimal(20,8)" json:"quantity"`
	Symbol    *string         `json:"symbol,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type AssetPriceHistory struct {
	ID         int64          `db:"id" json:"id"`
	UUID       string         `db:"uuid" json:"uuid"`
	AssetID    int64          `db:"asset_id" json:"assetId"`
	Price      decimal.Decimal `db:"price" json:"price"`
	Source     string         `db:"source" json:"source"`
	RecordedAt time.Time     `db:"recorded_at" json:"recordedAt"`
}
