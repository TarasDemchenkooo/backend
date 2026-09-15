package application

import (
	"regexp"
	"testing"
)

func TestNewVerificationCode(t *testing.T) {
	pattern := regexp.MustCompile(`^\d{6}$`)
	seen := make(map[string]struct{})
	leadingZero := false

	for range 5000 {
		code, err := newVerificationCode()
		if err != nil {
			t.Fatalf("newVerificationCode() error = %v", err)
		}

		if !pattern.MatchString(code) {
			t.Fatalf("code = %q, want exactly 6 digits", code)
		}

		if code[0] == '0' {
			leadingZero = true
		}

		seen[code] = struct{}{}
	}

	if !leadingZero {
		t.Error("no code starts with 0 in 5000 attempts, want leading zeros to be kept")
	}

	if len(seen) < 4900 {
		t.Errorf("distinct codes = %d of 5000, want nearly all to differ", len(seen))
	}
}
