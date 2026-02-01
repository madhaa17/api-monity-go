package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type SavingGoal struct {
	ID            int64           `gorm:"primaryKey" json:"-"`
	UUID          string          `gorm:"type:uuid;default:gen_random_uuid()" json:"uuid"`
	UserID        int64           `gorm:"index" json:"-"`
	Title         string          `json:"title"`
	TargetAmount  decimal.Decimal `gorm:"type:decimal(20,2)" json:"targetAmount"`
	CurrentAmount decimal.Decimal `gorm:"type:decimal(20,2);default:0" json:"currentAmount"`
	Deadline      *time.Time      `json:"deadline,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}
