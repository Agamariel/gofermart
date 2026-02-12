package utils

import "testing"

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{"valid simple", "79927398713", true},
		{"valid short", "42", true}, // 4*2=8, 8+2=10
		{"invalid", "79927398714", false},
		{"non digit", "12a45", false},
		{"empty", "", true}, // сумма 0 делится на 10
		{"leading zeros", "0000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateLuhn(tt.number); got != tt.want {
				t.Errorf("ValidateLuhn(%s) = %v, want %v", tt.number, got, tt.want)
			}
		})
	}
}
