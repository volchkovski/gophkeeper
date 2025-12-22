// Package validator provides input validation utilities for GophKeeper.
package validator

import (
	"regexp"
	"strings"
	"unicode"
)

// Validator provides validation methods.
type Validator struct {
	errors []string
}

// New creates a new Validator.
func New() *Validator {
	return &Validator{
		errors: make([]string, 0),
	}
}

// Valid returns true if there are no validation errors.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// Errors returns all validation errors.
func (v *Validator) Errors() []string {
	return v.errors
}

// AddError adds a validation error.
func (v *Validator) AddError(message string) {
	v.errors = append(v.errors, message)
}

// Check adds an error if the condition is false.
func (v *Validator) Check(ok bool, message string) {
	if !ok {
		v.AddError(message)
	}
}

// Required checks that a string is not empty.
func (v *Validator) Required(value, field string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field + " is required")
	}
}

// MinLength checks minimum length.
func (v *Validator) MinLength(value string, min int, field string) {
	if len(value) < min {
		v.AddError(field + " must be at least " + string(rune('0'+min)) + " characters")
	}
}

// MaxLength checks maximum length.
func (v *Validator) MaxLength(value string, max int, field string) {
	if len(value) > max {
		v.AddError(field + " must be at most " + string(rune('0'+max)) + " characters")
	}
}

// Email validates email format.
func (v *Validator) Email(value, field string) {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.AddError(field + " must be a valid email address")
	}
}

// ValidUsername validates username format.
func ValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	// Only allow alphanumeric and underscores
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// ValidPassword validates password strength.
func ValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	return true
}

// ValidSecretName validates secret name.
func ValidSecretName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	return true
}

// ValidCardNumber validates credit card number (basic Luhn check).
func ValidCardNumber(number string) bool {
	// Remove spaces and dashes
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "-", "")

	if len(number) < 13 || len(number) > 19 {
		return false
	}

	// Check if all characters are digits
	for _, r := range number {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	// Luhn algorithm
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}

	return sum%10 == 0
}

// ValidExpiryDate validates card expiry date format (MM/YY).
func ValidExpiryDate(expiry string) bool {
	if len(expiry) != 5 {
		return false
	}
	if expiry[2] != '/' {
		return false
	}
	month := expiry[:2]
	year := expiry[3:]

	for _, r := range month + year {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	m := int(month[0]-'0')*10 + int(month[1]-'0')
	return m >= 1 && m <= 12
}

// ValidCVV validates CVV (3 or 4 digits).
func ValidCVV(cvv string) bool {
	if len(cvv) < 3 || len(cvv) > 4 {
		return false
	}
	for _, r := range cvv {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

