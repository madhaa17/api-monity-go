package repository

import (
	"testing"

	"monity/internal/core/port"
)

func Test_escapeLike(t *testing.T) {
	for in, want := range map[string]string{"kopi": "kopi", "50%": `50\%`, "a_b": `a\_b`, `c:\x`: `c:\\x`} {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}

func Test_transactionTable_order(t *testing.T) {
	tests := map[port.TransactionSort]string{
		{Field: "date", Desc: true}:   "expenses.date DESC, expenses.created_at DESC, expenses.id DESC",
		{Field: "date"}:               "expenses.date ASC, expenses.created_at ASC, expenses.id ASC",
		{Field: "amount", Desc: true}: "expenses.amount DESC, expenses.date DESC, expenses.id DESC",
	}
	for sort, want := range tests {
		if got := expensesTable.order(sort); got != want {
			t.Errorf("order(%+v) = %q, want %q", sort, got, want)
		}
	}
}
