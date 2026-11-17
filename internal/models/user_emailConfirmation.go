package models

import "time"

// EmailConfirmationResendAfter is how long to wait before sending another
// confirmation email.
const EmailConfirmationResendAfter = time.Minute

// EmailConfirmed reports whether the user has confirmed their address.
// Reminder emails (and any other email nobody asked for in the moment) are
// only sent to a confirmed address; nothing else in the app depends on it.
func (u *User) EmailConfirmed() bool {
	return u.EmailConfirmedAt != nil
}

// CanResendEmailConfirmation reports whether enough time has passed since the
// last confirmation email.
func (u *User) CanResendEmailConfirmation(now time.Time) bool {
	return u.EmailConfirmationSentAt == nil || now.Sub(*u.EmailConfirmationSentAt) >= EmailConfirmationResendAfter
}
