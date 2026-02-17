package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type ReceivablePayment struct {
	ID            int64           `gorm:"primaryKey" json:"-"`
	UUID          string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	ReceivableID  int64           `gorm:"index" json:"-"`
	Amount        decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`
	Date          time.Time       `json:"date"`
	Note          *string         `json:"note,omitempty"`
	AssetID       *int64          `gorm:"index" json:"-"`
	CreatedAt     time.Time       `json:"createdAt"`

	Receivable *Receivable `gorm:"foreignKey:ReceivableID" json:"receivable,omitempty"`
}

func (ReceivablePayment) TableName() string { return "receivable_payments" }
