package validator

import (
	"net/mail"
	"regexp"
	"time"
	"unicode"
)

// IsValidEmail checks if the email is valid per RFC 5322
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidPassword checks password strength for regular users
// Must be at least 8 characters and contain at least one digit
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasDigit := false
	for _, ch := range password {
		if unicode.IsDigit(ch) {
			hasDigit = true
			break
		}
	}

	return hasDigit
}

// IsValidAdminPassword checks password strength for admin users
// Must be at least 12 characters with uppercase, lowercase, digit, and special char
func IsValidAdminPassword(password string) bool {
	if len(password) < 12 {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// IsValidUsername checks if username is alphanumeric with underscores, 3-50 chars
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	return matched
}

// IsValidDateString checks if the date string is in ISO format and valid
func IsValidDateString(dateStr string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// IsInPast checks if the date is in the past
func IsInPast(t time.Time) bool {
	return t.Before(time.Now())
}

// IsInEnum checks if value is in the allowed set
func IsInEnum(value string, allowed []string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

// ContainsDigit checks if the string contains at least one digit
func ContainsDigit(s string) bool {
	for _, ch := range s {
		if unicode.IsDigit(ch) {
			return true
		}
	}
	return false
}
