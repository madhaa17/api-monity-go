package handler

import (
	"regexp"
	"testing"
)

func Test_resolveDateFilter(t *testing.T) {
	tests := []struct {
		name          string
		dateParam     string
		tzParam       string
		wantDateRegex string
		wantTimezone  string
	}{
		{
			name:          "empty date param",
			dateParam:     "",
			tzParam:       "",
			wantDateRegex: "^$",
			wantTimezone:  "",
		},
		{
			name:          "empty date with tz ignored",
			dateParam:     "",
			tzParam:       "Asia/Jakarta",
			wantDateRegex: "^$",
			wantTimezone:  "",
		},
		{
			name:          "today without tz returns server now",
			dateParam:     "today",
			tzParam:       "",
			wantDateRegex: `^\d{4}-\d{2}-\d{2}$`,
			wantTimezone:  "",
		},
		{
			name:          "today with tz returns server now in tz",
			dateParam:     "today",
			tzParam:       "Asia/Jakarta",
			wantDateRegex: `^\d{4}-\d{2}-\d{2}$`,
			wantTimezone:  "Asia/Jakarta",
		},
		{
			name:          "explicit date YYYY-MM-DD",
			dateParam:     "2026-01-15",
			tzParam:       "",
			wantDateRegex: "^2026-01-15$",
			wantTimezone:  "",
		},
		{
			name:          "explicit date with valid tz",
			dateParam:     "2026-01-15",
			tzParam:       "Asia/Jakarta",
			wantDateRegex: "^2026-01-15$",
			wantTimezone:  "Asia/Jakarta",
		},
		{
			name:          "explicit date with invalid tz no timezone",
			dateParam:     "2026-01-15",
			tzParam:       "Invalid/Timezone",
			wantDateRegex: "^2026-01-15$",
			wantTimezone:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotTz := resolveDateFilter(tt.dateParam, tt.tzParam)

			matched, err := regexp.MatchString(tt.wantDateRegex, gotDate)
			if err != nil {
				t.Fatalf("invalid wantDateRegex %q: %v", tt.wantDateRegex, err)
			}
			if !matched {
				t.Errorf("resolveDateFilter() dateFilter = %q, want match %q", gotDate, tt.wantDateRegex)
			}
			if gotTz != tt.wantTimezone {
				t.Errorf("resolveDateFilter() timezone = %q, want %q", gotTz, tt.wantTimezone)
			}
		})
	}
}
