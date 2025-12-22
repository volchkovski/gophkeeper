package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator_New(t *testing.T) {
	v := New()
	assert.NotNil(t, v)
	assert.True(t, v.Valid())
	assert.Empty(t, v.Errors())
}

func TestValidator_AddError(t *testing.T) {
	v := New()
	v.AddError("test error")

	assert.False(t, v.Valid())
	assert.Len(t, v.Errors(), 1)
	assert.Equal(t, "test error", v.Errors()[0])
}

func TestValidator_Check(t *testing.T) {
	tests := []struct {
		name      string
		condition bool
		message   string
		wantValid bool
	}{
		{
			name:      "condition true - no error",
			condition: true,
			message:   "error message",
			wantValid: true,
		},
		{
			name:      "condition false - error added",
			condition: false,
			message:   "validation failed",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Check(tt.condition, tt.message)

			assert.Equal(t, tt.wantValid, v.Valid())
			if !tt.wantValid {
				assert.Contains(t, v.Errors(), tt.message)
			}
		})
	}
}

func TestValidator_Required(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		field     string
		wantValid bool
	}{
		{
			name:      "non-empty value",
			value:     "test",
			field:     "username",
			wantValid: true,
		},
		{
			name:      "empty value",
			value:     "",
			field:     "username",
			wantValid: false,
		},
		{
			name:      "whitespace only",
			value:     "   ",
			field:     "password",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Required(tt.value, tt.field)
			assert.Equal(t, tt.wantValid, v.Valid())
		})
	}
}

func TestValidator_MinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		wantValid bool
	}{
		{
			name:      "above minimum",
			value:     "hello",
			min:       3,
			wantValid: true,
		},
		{
			name:      "exactly minimum",
			value:     "abc",
			min:       3,
			wantValid: true,
		},
		{
			name:      "below minimum",
			value:     "ab",
			min:       3,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.MinLength(tt.value, tt.min, "field")
			assert.Equal(t, tt.wantValid, v.Valid())
		})
	}
}

func TestValidator_MaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		max       int
		wantValid bool
	}{
		{
			name:      "below maximum",
			value:     "hi",
			max:       5,
			wantValid: true,
		},
		{
			name:      "exactly maximum",
			value:     "hello",
			max:       5,
			wantValid: true,
		},
		{
			name:      "above maximum",
			value:     "hello world",
			max:       5,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.MaxLength(tt.value, tt.max, "field")
			assert.Equal(t, tt.wantValid, v.Valid())
		})
	}
}

func TestValidator_Email(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantValid bool
	}{
		{
			name:      "valid email",
			email:     "test@example.com",
			wantValid: true,
		},
		{
			name:      "valid email with subdomain",
			email:     "user@mail.example.com",
			wantValid: true,
		},
		{
			name:      "invalid - no @",
			email:     "testexample.com",
			wantValid: false,
		},
		{
			name:      "invalid - no domain",
			email:     "test@",
			wantValid: false,
		},
		{
			name:      "invalid - no local part",
			email:     "@example.com",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.Email(tt.email, "email")
			assert.Equal(t, tt.wantValid, v.Valid())
		})
	}
}

func TestValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{
			name:     "valid username",
			username: "john_doe",
			want:     true,
		},
		{
			name:     "valid alphanumeric",
			username: "user123",
			want:     true,
		},
		{
			name:     "too short",
			username: "ab",
			want:     false,
		},
		{
			name:     "too long",
			username: "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnop12345",
			want:     false,
		},
		{
			name:     "contains special chars",
			username: "user@name",
			want:     false,
		},
		{
			name:     "contains space",
			username: "user name",
			want:     false,
		},
		{
			name:     "empty",
			username: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidUsername(tt.username)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "valid password",
			password: "SecurePass123",
			want:     true,
		},
		{
			name:     "exactly 8 chars",
			password: "12345678",
			want:     true,
		},
		{
			name:     "too short",
			password: "1234567",
			want:     false,
		},
		{
			name:     "empty",
			password: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidPassword(tt.password)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidSecretName(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		want       bool
	}{
		{
			name:       "valid name",
			secretName: "my-secret",
			want:       true,
		},
		{
			name:       "valid with spaces",
			secretName: "My Secret Name",
			want:       true,
		},
		{
			name:       "empty",
			secretName: "",
			want:       false,
		},
		{
			name:       "too long",
			secretName: string(make([]byte, 256)),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidSecretName(tt.secretName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidCardNumber(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "valid Visa",
			number: "4111111111111111",
			want:   true,
		},
		{
			name:   "valid with spaces",
			number: "4111 1111 1111 1111",
			want:   true,
		},
		{
			name:   "valid with dashes",
			number: "4111-1111-1111-1111",
			want:   true,
		},
		{
			name:   "valid MasterCard",
			number: "5500000000000004",
			want:   true,
		},
		{
			name:   "invalid - wrong checksum",
			number: "4111111111111112",
			want:   false,
		},
		{
			name:   "too short",
			number: "411111111111",
			want:   false,
		},
		{
			name:   "too long",
			number: "41111111111111111111",
			want:   false,
		},
		{
			name:   "contains letters",
			number: "4111abcd11111111",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidCardNumber(tt.number)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidExpiryDate(t *testing.T) {
	tests := []struct {
		name   string
		expiry string
		want   bool
	}{
		{
			name:   "valid date",
			expiry: "12/25",
			want:   true,
		},
		{
			name:   "valid January",
			expiry: "01/30",
			want:   true,
		},
		{
			name:   "invalid month 00",
			expiry: "00/25",
			want:   false,
		},
		{
			name:   "invalid month 13",
			expiry: "13/25",
			want:   false,
		},
		{
			name:   "wrong format",
			expiry: "1225",
			want:   false,
		},
		{
			name:   "too short",
			expiry: "1/25",
			want:   false,
		},
		{
			name:   "contains letters",
			expiry: "ab/25",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidExpiryDate(tt.expiry)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidCVV(t *testing.T) {
	tests := []struct {
		name string
		cvv  string
		want bool
	}{
		{
			name: "valid 3 digits",
			cvv:  "123",
			want: true,
		},
		{
			name: "valid 4 digits (Amex)",
			cvv:  "1234",
			want: true,
		},
		{
			name: "too short",
			cvv:  "12",
			want: false,
		},
		{
			name: "too long",
			cvv:  "12345",
			want: false,
		},
		{
			name: "contains letters",
			cvv:  "12a",
			want: false,
		},
		{
			name: "empty",
			cvv:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidCVV(tt.cvv)
			assert.Equal(t, tt.want, got)
		})
	}
}
