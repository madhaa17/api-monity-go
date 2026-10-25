package handler

import (
	"net/http/httptest"
	"testing"
)

func Test_parseTransactionFilter(t *testing.T) {
	const catA = "11111111-1111-1111-1111-111111111111"
	const catB = "22222222-2222-2222-2222-222222222222"

	t.Run("all filters", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/expenses?date_from=2026-10-01&category="+catA+","+catB+"&asset="+catA+"&q=+kopi+&min_amount=10.5&max_amount=100", nil)
		f, err := parseTransactionFilter(r)
		if err != nil {
			t.Fatal(err)
		}
		if f.DateFrom == nil || f.DateFrom.Format("2006-01-02") != "2026-10-01" || f.DateTo != nil {
			t.Errorf("dates = %v, %v; want only date_from", f.DateFrom, f.DateTo)
		}
		if len(f.CategoryUUIDs) != 2 || f.AssetUUID != catA || f.Search != "kopi" {
			t.Errorf("got %+v", f)
		}
		if f.MinAmount.String() != "10.5" || f.MaxAmount.String() != "100" {
			t.Errorf("amounts = %v, %v", f.MinAmount, f.MaxAmount)
		}
	})

	t.Run("month", func(t *testing.T) {
		f, err := parseTransactionFilter(httptest.NewRequest("GET", "/expenses?month=2026-02", nil))
		if err != nil {
			t.Fatal(err)
		}
		if f.DateFrom.Format("2006-01-02") != "2026-02-01" || f.DateTo.Format("2006-01-02") != "2026-02-28" {
			t.Errorf("month range = %v .. %v", f.DateFrom, f.DateTo)
		}
	})

	errorCases := map[string]string{
		"date_from=2026-13-01": "date_from must be YYYY-MM-DD",
		"date_to=yesterday":    "date_to must be YYYY-MM-DD",
		"month=2026-1":         "month must be YYYY-MM",
		"category=abc":         "category must be a comma-separated list of category UUIDs",
		"asset=1":              "asset must be an asset UUID",
		"min_amount=ten":       "min_amount must be a number",
		"max_amount=1e":        "max_amount must be a number",
	}
	for query, want := range errorCases {
		t.Run(query, func(t *testing.T) {
			_, err := parseTransactionFilter(httptest.NewRequest("GET", "/expenses?"+query, nil))
			if err == nil || err.Error() != want {
				t.Fatalf("err = %v, want %q", err, want)
			}
		})
	}
}
