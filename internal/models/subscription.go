package models

import "time"

const (
	SubscriptionTrialing = "trialing"
	SubscriptionActive   = "active"
	SubscriptionPastDue  = "past_due"
	SubscriptionCanceled = "canceled"
	SubscriptionLifetime = "lifetime"
	SubscriptionComped   = "comped"
)

const (
	// TrialLength is how long a new account has the paid features, no card.
	TrialLength = 60 * 24 * time.Hour
	// GracePeriod is how long the paid features keep working after a trial or
	// paid period ends, before they go quiet. Nothing is ever deleted.
	GracePeriod = 14 * 24 * time.Hour
)

// startTrial puts a brand-new account on the trial. It leaves any status that
// was set explicitly (seeds, tests, comped accounts) alone.
func (u *User) startTrial(now time.Time) {
	if u.SubscriptionStatus != "" {
		return
	}
	u.SubscriptionStatus = SubscriptionTrialing
	u.TrialEndsAt = &[]time.Time{now.Add(TrialLength)}[0]
}

// IsSubscribed reports whether the paid features are on for this user,
// including the grace period after a trial or paid period ends.
func (u *User) IsSubscribed() bool {
	return u.isSubscribedAt(time.Now().UTC())
}

// InGracePeriod reports whether the user's trial or paid period has ended but
// the paid features are still on.
func (u *User) InGracePeriod() bool {
	return u.inGracePeriodAt(time.Now().UTC())
}

// AccessEndsAt is when the trial or paid period ends (before any grace), or
// nil if it has no end or no access to end.
func (u *User) AccessEndsAt() *time.Time {
	switch u.SubscriptionStatus {
	case SubscriptionTrialing:
		return u.TrialEndsAt
	case SubscriptionActive, SubscriptionPastDue, SubscriptionCanceled:
		return u.CurrentPeriodEnd
	}
	return nil
}

func (u *User) isSubscribedAt(now time.Time) bool {
	switch u.SubscriptionStatus {
	case SubscriptionLifetime, SubscriptionComped:
		return true
	case SubscriptionTrialing, SubscriptionActive, SubscriptionPastDue, SubscriptionCanceled:
		end := u.AccessEndsAt()
		if end == nil {
			// Stripe says it's live and hasn't told us when it ends yet.
			return u.SubscriptionStatus == SubscriptionActive || u.SubscriptionStatus == SubscriptionPastDue
		}
		return now.Before(end.Add(GracePeriod))
	}
	return false
}

func (u *User) inGracePeriodAt(now time.Time) bool {
	end := u.AccessEndsAt()
	return end != nil && !now.Before(*end) && u.isSubscribedAt(now)
}
