package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type SavingGoal struct {
	ID            int64           `db:"id" json:"id"`
	UUID          string          `db:"uuid" json:"uuid"`
	UserID        int64           `db:"user_id" json:"userId"`
	Title         string          `db:"title" json:"title"`
	TargetAmount  decimal.Decimal `db:"target_amount" json:"targetAmount"`
	CurrentAmount decimal.Decimal `db:"current_amount" json:"currentAmount"`
	Deadline      *time.Time      `db:"deadline" json:"deadline,omitempty"`
	CreatedAt     time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updatedAt"`
}
