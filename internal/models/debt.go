package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Debt struct {
	ID         int64             `gorm:"primaryKey" json:"-"`
	UUID       string            `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID     int64             `gorm:"index" json:"-"`
	PartyName  string            `gorm:"column:party_name" json:"partyName"`
	Amount     decimal.Decimal   `gorm:"type:decimal(20,2)" json:"amount"`
	PaidAmount decimal.Decimal   `gorm:"type:decimal(20,2);default:0" json:"paidAmount"`
	DueDate    *time.Time        `json:"dueDate,omitempty"`
	Status     ObligationStatus  `gorm:"type:obligation_status" json:"status"`
	Note       *string           `json:"note,omitempty"`
	AssetID    *int64            `gorm:"index" json:"-"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`

	Asset *Asset `gorm:"foreignKey:AssetID" json:"asset,omitempty"`
}

func (Debt) TableName() string { return "debts" }
