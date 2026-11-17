package models

import (
	"testing"
	"time"
)

var testNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func userWith(status string, trialEnds, periodEnd *time.Time) *User {
	return &User{SubscriptionStatus: status, TrialEndsAt: trialEnds, CurrentPeriodEnd: periodEnd}
}

func at(offset time.Duration) *time.Time {
	t := testNow.Add(offset)
	return &t
}

func TestSubscriptionAccess(t *testing.T) {
	day := 24 * time.Hour
	cases := []struct {
		name       string
		user       *User
		subscribed bool
		grace      bool
	}{
		{"trial running", userWith(SubscriptionTrialing, at(10*day), nil), true, false},
		{"trial ended, in grace", userWith(SubscriptionTrialing, at(-3*day), nil), true, true},
		{"trial ended, past grace", userWith(SubscriptionTrialing, at(-15*day), nil), false, false},
		{"active, period running", userWith(SubscriptionActive, nil, at(10*day)), true, false},
		{"active, no period yet", userWith(SubscriptionActive, nil, nil), true, false},
		{"past due keeps access", userWith(SubscriptionPastDue, nil, at(-2*day)), true, true},
		{"active but webhook missed", userWith(SubscriptionActive, nil, at(-30*day)), false, false},
		{"canceled, period remaining", userWith(SubscriptionCanceled, nil, at(5*day)), true, false},
		{"canceled, in grace", userWith(SubscriptionCanceled, nil, at(-1*day)), true, true},
		{"canceled, past grace", userWith(SubscriptionCanceled, nil, at(-20*day)), false, false},
		{"lifetime", userWith(SubscriptionLifetime, nil, nil), true, false},
		{"comped", userWith(SubscriptionComped, nil, nil), true, false},
		{"unset status", userWith("", nil, nil), false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.isSubscribedAt(testNow); got != c.subscribed {
				t.Errorf("isSubscribedAt = %v, want %v", got, c.subscribed)
			}
			if got := c.user.inGracePeriodAt(testNow); got != c.grace {
				t.Errorf("inGracePeriodAt = %v, want %v", got, c.grace)
			}
		})
	}
}

func TestStartTrial(t *testing.T) {
	u := &User{}
	u.startTrial(testNow)
	if u.SubscriptionStatus != SubscriptionTrialing || u.TrialEndsAt == nil || !u.TrialEndsAt.Equal(testNow.Add(TrialLength)) {
		t.Fatalf("new user not on a %v trial: %+v", TrialLength, u)
	}

	comped := &User{SubscriptionStatus: SubscriptionComped}
	comped.startTrial(testNow)
	if comped.SubscriptionStatus != SubscriptionComped || comped.TrialEndsAt != nil {
		t.Fatalf("explicit status was overwritten: %+v", comped)
	}
}
