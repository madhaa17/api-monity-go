package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type ActivityService interface {
	ListActivities(ctx context.Context, userID int64, groupBy string) (*ActivityResponse, error)
}

// ActivityItem is one entry in a group: either income or expense, in chronological order.
type ActivityItem struct {
	Type      string          `json:"type"` // "income" or "expense"
	UUID      string          `json:"uuid"`
	Amount    decimal.Decimal `json:"amount"`
	Date      time.Time       `json:"date"`
	CreatedAt time.Time       `json:"createdAt"`
	Note      *string         `json:"note,omitempty"`
	Source    string          `json:"source,omitempty"`   // income only
	Category  string          `json:"category,omitempty"`  // expense only
}

type ActivityGroup struct {
	Key   string         `json:"key"`
	Items []ActivityItem `json:"items"`
}

type ActivityResponse struct {
	Groups []ActivityGroup `json:"groups"`
}
