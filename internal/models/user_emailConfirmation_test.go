package models

import (
	"testing"
	"time"
)

func TestEmailConfirmed(t *testing.T) {
	now := time.Now()
	if (&User{}).EmailConfirmed() {
		t.Error("a new user should be unconfirmed")
	}
	if !(&User{EmailConfirmedAt: &now}).EmailConfirmed() {
		t.Error("a user with a confirmation time should be confirmed")
	}
}

func TestCanResendEmailConfirmation(t *testing.T) {
	now := time.Now()
	recent := now.Add(-10 * time.Second)
	old := now.Add(-2 * time.Minute)

	if !(&User{}).CanResendEmailConfirmation(now) {
		t.Error("never sent: should be able to send")
	}
	if (&User{EmailConfirmationSentAt: &recent}).CanResendEmailConfirmation(now) {
		t.Error("sent 10s ago: should wait")
	}
	if !(&User{EmailConfirmationSentAt: &old}).CanResendEmailConfirmation(now) {
		t.Error("sent 2m ago: should be able to resend")
	}
}
