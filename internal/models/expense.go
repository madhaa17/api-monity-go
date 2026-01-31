package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Expense struct {
	ID        int64            `db:"id" json:"id"`
	UUID      string           `db:"uuid" json:"uuid"`
	UserID    int64             `db:"user_id" json:"userId"`
	Amount    decimal.Decimal  `db:"amount" json:"amount"`
	Category  ExpenseCategory  `db:"category" json:"category"`
	Note      *string          `db:"note" json:"note,omitempty"`
	Date      time.Time        `db:"date" json:"date"`
	CreatedAt time.Time        `db:"created_at" json:"createdAt"`
}
