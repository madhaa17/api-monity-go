package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type DebtPayment struct {
	ID        int64           `gorm:"primaryKey" json:"-"`
	UUID      string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	DebtID    int64           `gorm:"index" json:"-"`
	Amount    decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`
	Date      time.Time       `json:"date"`
	Note      *string         `json:"note,omitempty"`
	AssetID   *int64          `gorm:"index" json:"-"`
	CreatedAt time.Time       `json:"createdAt"`

	Debt *Debt `gorm:"foreignKey:DebtID" json:"debt,omitempty"`
}

func (DebtPayment) TableName() string { return "debt_payments" }
