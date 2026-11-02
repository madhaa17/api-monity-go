package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// MarketDailyClose is a symbol's closing price on one trading day, in the currency it trades in
// (stocks) or the default currency (crypto, whose day closes at 00:00 UTC).
type MarketDailyClose struct {
	AssetType  AssetType       `gorm:"primaryKey;type:asset_type" json:"assetType"`
	Symbol     string          `gorm:"primaryKey" json:"symbol"`
	Currency   string          `gorm:"primaryKey;type:varchar(10)" json:"currency"`
	Date       time.Time       `gorm:"primaryKey;type:date" json:"date"`
	Close      decimal.Decimal `gorm:"type:decimal(20,8)" json:"close"`
	Source     string          `json:"source"`
	RecordedAt time.Time       `json:"recordedAt"`
}
