package port

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionFilter narrows income and expense lists and summaries. Zero values mean "no filter".
type TransactionFilter struct {
	DateFrom      *time.Time // inclusive, start of day (UTC)
	DateTo        *time.Time // inclusive: the whole day is included
	CategoryUUIDs []string   // match any of these categories
	AssetUUID     string
	Search        string // case-insensitive substring of the note (and source, for incomes)
	MinAmount     *decimal.Decimal
	MaxAmount     *decimal.Decimal
}

// TransactionSort is a validated sort order for income and expense lists.
type TransactionSort struct {
	Field string // "date" or "amount"
	Desc  bool
}

// DefaultTransactionSort lists newest transactions first.
var DefaultTransactionSort = TransactionSort{Field: "date", Desc: true}

// SummaryGroupBy is the period size of TransactionSummary.Series.
type SummaryGroupBy string

const (
	SummaryGroupByDay   SummaryGroupBy = "day"
	SummaryGroupByMonth SummaryGroupBy = "month"
)

// TransactionSummary aggregates the incomes or expenses matching a TransactionFilter.
type TransactionSummary struct {
	Total      decimal.Decimal `json:"total"`
	Count      int64           `json:"count"`
	Average    decimal.Decimal `json:"average"`
	ByCategory []CategoryTotal `json:"byCategory"` // largest total first
	GroupBy    SummaryGroupBy  `json:"groupBy"`
	Series     []PeriodTotal   `json:"series"` // oldest period first; periods without transactions are omitted
}

// PeriodTotal is one day (YYYY-MM-DD) or month (YYYY-MM) in a summary series.
type PeriodTotal struct {
	Period string          `json:"period"`
	Total  decimal.Decimal `json:"total"`
	Count  int64           `json:"count"`
}
