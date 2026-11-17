package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"journal/internal/utils"
)

const (
	DefaultReminderEmailTime   = "08:00"
	ReminderEmailMessageMaxLen = 280
)

var reminderEmailTimePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// ValidReminderEmailTime reports whether s is a zero-padded 24h "HH:MM".
func ValidReminderEmailTime(s string) bool {
	return reminderEmailTimePattern.MatchString(s)
}

// ReminderEmailMinutes converts "HH:MM" to minutes after midnight.
func ReminderEmailMinutes(s string) (int, bool) {
	if !ValidReminderEmailTime(s) {
		return 0, false
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// NormalizeTimezone returns tz if it is a loadable IANA name, otherwise "UTC".
func NormalizeTimezone(tz string) string {
	if tz == "" {
		return "UTC"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "UTC"
	}
	return tz
}

// CleanReminderEmailMessage trims and validates a custom message. An empty
// result means "use the default line".
func CleanReminderEmailMessage(s string) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if utf8.RuneCountInString(s) > ReminderEmailMessageMaxLen {
		return "", errors.New("Message must be 280 characters or fewer")
	}
	for _, r := range s {
		if (r < 0x20 && r != '\n') || r == 0x7f {
			return "", errors.New("Message contains unsupported characters")
		}
	}
	return s, nil
}

// SetReminderEmailMessage encrypts and stores msg (empty clears it).
func (u *User) SetReminderEmailMessage(msg string) error {
	if msg == "" {
		u.ReminderEmailMessage = nil
		return nil
	}
	enc, err := utils.Encrypt(msg)
	if err != nil {
		return err
	}
	u.ReminderEmailMessage = &enc
	return nil
}

// ReminderEmailMessageText returns the decrypted custom message ("" if none).
func (u *User) ReminderEmailMessageText() string {
	if u.ReminderEmailMessage == nil || *u.ReminderEmailMessage == "" {
		return ""
	}
	dec, err := utils.Decrypt(*u.ReminderEmailMessage)
	if err != nil {
		return ""
	}
	return dec
}

// ReminderEmailSettings is the slice of a user that decides whether a
// reminder email is due. Kept separate from User so the decision is a
// pure function of plain values.
type ReminderEmailSettings struct {
	Enabled    bool
	Subscribed bool
	Time       string     // "HH:MM" in the user's own timezone
	Timezone   string     // IANA name; empty or invalid means UTC
	LastSentOn *time.Time // date column; only Y-M-D is meaningful
}

func (u *User) ReminderEmailSettings() ReminderEmailSettings {
	return ReminderEmailSettings{
		Enabled:    u.ReminderEmailEnabled,
		Subscribed: u.IsSubscribed(),
		Time:       u.ReminderEmailTime,
		Timezone:   u.Timezone,
		LastSentOn: u.ReminderEmailLastSentOn,
	}
}

// ReminderEmailDue reports whether a reminder email should be queued at `now`,
// and the user's local date ("2006-01-02") to record as the last-sent day.
// It is due once the local clock has reached the chosen time, at most once per
// local day. There is deliberately no catch-up or missed-day logic: if the
// server was down at the chosen time, the email goes out later that same day,
// and never for a day that has already ended.
func ReminderEmailDue(s ReminderEmailSettings, now time.Time) (bool, string) {
	if !s.Enabled || !s.Subscribed {
		return false, ""
	}
	want, ok := ReminderEmailMinutes(s.Time)
	if !ok {
		return false, ""
	}
	loc, err := time.LoadLocation(NormalizeTimezone(s.Timezone))
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	today := local.Format("2006-01-02")
	if s.LastSentOn != nil && s.LastSentOn.Format("2006-01-02") == today {
		return false, today
	}
	// Compare wall-clock minutes (not instants) so a chosen time that falls in a
	// DST gap (e.g. 02:30 on spring-forward) fires as soon as the clock passes it.
	if local.Hour()*60+local.Minute() < want {
		return false, today
	}
	return true, today
}
