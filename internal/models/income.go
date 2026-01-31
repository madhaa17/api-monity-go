package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Income struct {
	ID        int64           `db:"id" json:"id"`
	UUID      string          `db:"uuid" json:"uuid"`
	UserID    int64           `db:"user_id" json:"userId"`
	Amount    decimal.Decimal `db:"amount" json:"amount"`
	Source    string          `db:"source" json:"source"`
	Note      *string         `db:"note" json:"note,omitempty"`
	Date      time.Time       `db:"date" json:"date"`
	CreatedAt time.Time       `db:"created_at" json:"createdAt"`
}
