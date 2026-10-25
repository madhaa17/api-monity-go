package service

import (
	"testing"
	"time"

	"monity/internal/core/port"
)

func Test_parseTransactionSort(t *testing.T) {
	tests := []struct {
		in      string
		want    port.TransactionSort
		wantErr bool
	}{
		{"", port.DefaultTransactionSort, false},
		{"date", port.TransactionSort{Field: "date"}, false},
		{"-date", port.TransactionSort{Field: "date", Desc: true}, false},
		{" amount ", port.TransactionSort{Field: "amount"}, false},
		{"-amount", port.TransactionSort{Field: "amount", Desc: true}, false},
		{"note", port.TransactionSort{}, true},
		{"--date", port.TransactionSort{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseTransactionSort(tt.in)
			if tt.wantErr {
				wantValidation(t, err, "sort must be one of date, -date, amount, -amount")
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("parseTransactionSort(%q) = %+v, %v; want %+v", tt.in, got, err, tt.want)
			}
		})
	}
}

func Test_parseSummaryGroupBy(t *testing.T) {
	for in, want := range map[string]port.SummaryGroupBy{"": "day", "day": "day", "Month": "month"} {
		if got, err := parseSummaryGroupBy(in); err != nil || got != want {
			t.Errorf("parseSummaryGroupBy(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	_, err := parseSummaryGroupBy("week")
	wantValidation(t, err, "group_by must be day or month")
}

func Test_validateTransactionFilter(t *testing.T) {
	day := func(d int) *time.Time { v := time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC); return &v }

	if err := validateTransactionFilter(port.TransactionFilter{DateFrom: day(1), DateTo: day(1), MinAmount: decPtr("5"), MaxAmount: decPtr("5")}); err != nil {
		t.Fatalf("equal bounds should be valid: %v", err)
	}
	wantValidation(t, validateTransactionFilter(port.TransactionFilter{DateFrom: day(2), DateTo: day(1)}), "date_from must not be after date_to")
	wantValidation(t, validateTransactionFilter(port.TransactionFilter{MinAmount: decPtr("10"), MaxAmount: decPtr("9.99")}), "min_amount must not be greater than max_amount")

	long := make([]rune, maxSearchLen+1)
	for i := range long {
		long[i] = 'a'
	}
	wantValidation(t, validateTransactionFilter(port.TransactionFilter{Search: string(long)}), "search must be at most 100 characters")
}
