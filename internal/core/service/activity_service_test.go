package service

import (
	"testing"
	"time"
)

func Test_normalizeGroupBy(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"day default", "", "day"},
		{"day lowercase", "day", "day"},
		{"day mixed", "Day", "day"},
		{"day with spaces", "  day  ", "day"},
		{"month lowercase", "month", "month"},
		{"month mixed", "Month", "month"},
		{"month with spaces", "  month  ", "month"},
		{"year lowercase", "year", "year"},
		{"year mixed", "Year", "year"},
		{"year with spaces", "  year  ", "year"},
		{"invalid defaults to day", "invalid", "day"},
		{"week defaults to day", "week", "day"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeGroupBy(tt.in); got != tt.want {
				t.Errorf("normalizeGroupBy(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func Test_dateMatches(t *testing.T) {
	utc := time.UTC
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skipf("Asia/Jakarta not available: %v", err)
	}

	tests := []struct {
		name       string
		t          time.Time
		dateFilter string
		loc        *time.Location
		want       bool
	}{
		{
			name:       "match in UTC",
			t:          time.Date(2026, 2, 14, 12, 0, 0, 0, utc),
			dateFilter: "2026-02-14",
			loc:        utc,
			want:       true,
		},
		{
			name:       "no match different day UTC",
			t:          time.Date(2026, 2, 14, 12, 0, 0, 0, utc),
			dateFilter: "2026-02-15",
			loc:        utc,
			want:       false,
		},
		{
			name:       "nil loc uses Local",
			t:          time.Date(2026, 2, 14, 0, 0, 0, 0, time.Local),
			dateFilter: "2026-02-14",
			loc:        nil,
			want:       true,
		},
		{
			name:       "match in Jakarta",
			t:          time.Date(2026, 2, 14, 3, 0, 0, 0, jkt),
			dateFilter: "2026-02-14",
			loc:        jkt,
			want:       true,
		},
		{
			name:       "UTC midnight in Jakarta next day",
			t:          time.Date(2026, 2, 14, 0, 0, 0, 0, utc),
			dateFilter: "2026-02-14",
			loc:        jkt,
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dateMatches(tt.t, tt.dateFilter, tt.loc); got != tt.want {
				t.Errorf("dateMatches(%v, %q, %v) = %v, want %v", tt.t, tt.dateFilter, tt.loc, got, tt.want)
			}
		})
	}
}
