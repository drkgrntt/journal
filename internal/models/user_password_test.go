package models

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		ok       bool
	}{
		{"empty", "", false},
		{"seven characters", "1234567", false},
		{"eight characters", "12345678", true},
		{"72 bytes", strings.Repeat("a", 72), true},
		{"73 bytes", strings.Repeat("a", 73), false},
		{"multibyte counts bytes", strings.Repeat("é", 37), false}, // 74 bytes
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePassword(tt.password) == ""; got != tt.ok {
				t.Errorf("ValidatePassword(%q) accepted = %v, want %v", tt.password, got, tt.ok)
			}
		})
	}
}
