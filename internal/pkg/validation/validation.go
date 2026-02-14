package validation

import (
	"fmt"
	"regexp"
	"unicode"
)

// Auth
const (
	MaxEmailLen    = 255
	MaxNameLen     = 200
	MinPasswordLen = 8
	MaxPasswordLen = 72 // bcrypt practical limit
)

// General string limits
const (
	MaxNoteLen         = 500
	MaxSourceLen       = 200
	MaxTitleLen        = 200
	MaxAssetNameLen    = 200
	MaxDescriptionLen  = 2000
	MaxSymbolLen       = 20
	MaxYieldPeriodLen  = 20
)

var emailRegex = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

// ValidEmail returns true if s is non-empty, has valid format (local@domain.tld), and length <= MaxEmailLen.
func ValidEmail(s string) bool {
	if s == "" || len(s) > MaxEmailLen {
		return false
	}
	return emailRegex.MatchString(s)
}

// ValidPassword returns (true, "") if password meets policy: length between MinPasswordLen and MaxPasswordLen,
// and at least one letter and one digit. Otherwise returns (false, msg).
func ValidPassword(s string) (ok bool, msg string) {
	if len(s) < MinPasswordLen {
		return false, "password must be at least 8 characters"
	}
	if len(s) > MaxPasswordLen {
		return false, "password must be at most 72 characters"
	}
	var hasLetter, hasDigit bool
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			break
		}
	}
	if !hasLetter || !hasDigit {
		return false, "password must contain at least one letter and one digit"
	}
	return true, ""
}

// CheckMaxLen returns an error if len(s) > max. If max <= 0, no check (returns nil).
func CheckMaxLen(s string, max int) error {
	if max <= 0 {
		return nil
	}
	if len(s) > max {
		return fmt.Errorf("must be at most %d characters", max)
	}
	return nil
}

// FieldTooLongError returns an error message for a field that exceeded max length.
func FieldTooLongError(field string, max int) error {
	return fmt.Errorf("%s must be at most %d characters", field, max)
}
