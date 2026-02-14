package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Asset struct {
	ID       int64           `gorm:"primaryKey" json:"-"`
	UUID     string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID   int64           `gorm:"index" json:"-"`
	Name     string          `json:"name"`
	Type     AssetType       `gorm:"type:asset_type" json:"type"`
	Quantity decimal.Decimal `gorm:"type:decimal(20,8)" json:"quantity"`
	Symbol   *string         `json:"symbol,omitempty"`

	// Purchase Information
	PurchasePrice    decimal.Decimal `gorm:"type:decimal(20,8);default:0" json:"purchasePrice"`
	PurchaseDate     time.Time       `json:"purchaseDate"`
	PurchaseCurrency string          `gorm:"type:varchar(10);default:'USD'" json:"purchaseCurrency"`
	TotalCost        decimal.Decimal `gorm:"type:decimal(20,8);default:0" json:"totalCost"`

	// Additional Costs (optional)
	TransactionFee  *decimal.Decimal `gorm:"type:decimal(20,8)" json:"transactionFee,omitempty"`
	MaintenanceCost *decimal.Decimal `gorm:"type:decimal(20,8)" json:"maintenanceCost,omitempty"`

	// Target & Planning (optional)
	TargetPrice *decimal.Decimal `gorm:"type:decimal(20,8)" json:"targetPrice,omitempty"`
	TargetDate  *time.Time       `json:"targetDate,omitempty"`

	// Real Asset Specific (optional)
	EstimatedYield *decimal.Decimal `gorm:"type:decimal(20,8)" json:"estimatedYield,omitempty"`
	YieldPeriod    *string          `gorm:"type:varchar(20)" json:"yieldPeriod,omitempty"`

	// Documentation (optional)
	Description *string `gorm:"type:text" json:"description,omitempty"`
	Notes       *string `gorm:"type:text" json:"notes,omitempty"`

	// Status
	Status    AssetStatus      `gorm:"type:varchar(20);default:'ACTIVE'" json:"status"`
	SoldAt    *time.Time       `json:"soldAt,omitempty"`
	SoldPrice *decimal.Decimal `gorm:"type:decimal(20,8)" json:"soldPrice,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AssetPriceHistory struct {
	ID         int64           `gorm:"primaryKey" json:"-"`
	UUID       string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	AssetID    int64           `gorm:"index" json:"-"`
	Price      decimal.Decimal `gorm:"type:decimal(20,8)" json:"price"`
	Source     string          `json:"source"`
	RecordedAt time.Time       `json:"recordedAt"`
}
