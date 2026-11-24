package models

import (
	"strings"
	"testing"
	"time"
)

func TestSessionTokensAreRandomAndOnlyTheirHashIsKept(t *testing.T) {
	a, hashA, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := newSessionToken()
	if a == b {
		t.Error("two tokens were identical")
	}
	if len(a) < 43 { // 32 bytes, base64url without padding
		t.Errorf("token is only %d characters", len(a))
	}
	if hashA == a || strings.Contains(hashA, a) {
		t.Error("the stored value contains the token")
	}
	if hashA != HashSessionToken(a) {
		t.Error("hash is not reproducible from the token")
	}
	if HashSessionToken(a) == HashSessionToken(b) {
		t.Error("different tokens hashed the same")
	}
}

func TestSessionNeedsRotation(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	const token = "current-token"
	s := &Session{TokenHash: HashSessionToken(token), RotatedAt: now.Add(-SessionRotateAfter)}

	tests := []struct {
		name  string
		token string
		age   time.Duration
		want  bool
	}{
		{"exactly due", token, SessionRotateAfter, true},
		{"overdue", token, 40 * 24 * time.Hour, true},
		{"just under due", token, SessionRotateAfter - time.Second, false},
		{"brand new", token, 0, false},
		{"previous token never rotates", "an-older-token", SessionRotateAfter, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.RotatedAt = now.Add(-tt.age)
			if got := s.NeedsRotation(tt.token, now); got != tt.want {
				t.Errorf("NeedsRotation = %v, want %v", got, tt.want)
			}
		})
	}
}
