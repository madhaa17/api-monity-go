package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Income struct {
	ID        int64           `gorm:"primaryKey" json:"-"`
	UUID      string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID    int64           `gorm:"index" json:"-"`
	Amount    decimal.Decimal `gorm:"type:decimal(20,2)" json:"amount"`
	Source    string          `json:"source"`
	Note      *string         `json:"note,omitempty"`
	Date      time.Time       `json:"date"`
	CreatedAt time.Time       `json:"createdAt"`
}
