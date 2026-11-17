package models

import (
	"strings"
	"testing"
	"time"
)

// dueSettings builds enabled, subscribed settings for 08:00 in tz.
func dueSettings(tz string) ReminderEmailSettings {
	return ReminderEmailSettings{Enabled: true, Subscribed: true, Time: "08:00", Timezone: tz}
}

func parseAt(t *testing.T, layout, value string) time.Time {
	t.Helper()
	v, err := time.Parse(layout, value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func day(t *testing.T, d string) *time.Time {
	v := parseAt(t, "2006-01-02", d)
	return &v
}

func TestReminderEmailDue(t *testing.T) {
	utc := "2006-01-02 15:04"
	tests := []struct {
		name     string
		mutate   func(*ReminderEmailSettings)
		nowUTC   string
		wantDue  bool
		wantDate string
	}{
		{"before time, UTC", nil, "2025-06-10 07:59", false, "2025-06-10"},
		{"exactly at time, UTC", nil, "2025-06-10 08:00", true, "2025-06-10"},
		{"late in day, UTC", nil, "2025-06-10 23:30", true, "2025-06-10"},
		{"empty timezone is UTC", func(s *ReminderEmailSettings) { s.Timezone = "" }, "2025-06-10 08:00", true, "2025-06-10"},
		{"invalid timezone falls back to UTC", func(s *ReminderEmailSettings) { s.Timezone = "Not/AZone" }, "2025-06-10 08:00", true, "2025-06-10"},
		{"New York, 08:00 EDT is 12:00 UTC (not yet)", func(s *ReminderEmailSettings) { s.Timezone = "America/New_York" }, "2025-06-10 11:59", false, "2025-06-10"},
		{"New York, 08:00 EDT is 12:00 UTC (due)", func(s *ReminderEmailSettings) { s.Timezone = "America/New_York" }, "2025-06-10 12:00", true, "2025-06-10"},
		{"Tokyo local date is ahead of UTC", func(s *ReminderEmailSettings) { s.Timezone = "Asia/Tokyo" }, "2025-06-09 23:00", true, "2025-06-10"},
		{"Tokyo before local time", func(s *ReminderEmailSettings) { s.Timezone = "Asia/Tokyo" }, "2025-06-09 22:59", false, "2025-06-10"},
		{"already sent today (local)", func(s *ReminderEmailSettings) { s.LastSentOn = day(t, "2025-06-10") }, "2025-06-10 20:00", false, "2025-06-10"},
		{"sent yesterday, due today", func(s *ReminderEmailSettings) { s.LastSentOn = day(t, "2025-06-09") }, "2025-06-10 08:00", true, "2025-06-10"},
		{"sent long ago: no catch-up, one email", func(s *ReminderEmailSettings) { s.LastSentOn = day(t, "2025-05-01") }, "2025-06-10 09:00", true, "2025-06-10"},
		{"sent today in local date, UTC already tomorrow", func(s *ReminderEmailSettings) {
			s.Timezone = "America/Los_Angeles"
			s.LastSentOn = day(t, "2025-06-10")
		}, "2025-06-11 03:00", false, "2025-06-10"},
		{"disabled", func(s *ReminderEmailSettings) { s.Enabled = false }, "2025-06-10 12:00", false, ""},
		{"lapsed (not subscribed)", func(s *ReminderEmailSettings) { s.Subscribed = false }, "2025-06-10 12:00", false, ""},
		{"bad stored time", func(s *ReminderEmailSettings) { s.Time = "8am" }, "2025-06-10 12:00", false, ""},
		// US spring forward 2025-03-09: 02:00 -> 03:00 in New York, so 02:30 never exists.
		{"DST gap: chosen 02:30, clock jumps to 03:00", func(s *ReminderEmailSettings) {
			s.Timezone = "America/New_York"
			s.Time = "02:30"
		}, "2025-03-09 07:00", true, "2025-03-09"}, // 07:00 UTC = 03:00 EDT
		{"DST gap: chosen 02:30, before the jump", func(s *ReminderEmailSettings) {
			s.Timezone = "America/New_York"
			s.Time = "02:30"
		}, "2025-03-09 06:59", false, "2025-03-09"}, // 01:59 EST
		// US fall back 2025-11-02: 01:00-02:00 happens twice; once sent, the date guard holds.
		{"DST overlap: second 01:30 does not resend", func(s *ReminderEmailSettings) {
			s.Timezone = "America/New_York"
			s.Time = "01:30"
			s.LastSentOn = day(t, "2025-11-02")
		}, "2025-11-02 06:30", false, "2025-11-02"}, // 01:30 EST (second pass)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := dueSettings("")
			if tt.mutate != nil {
				tt.mutate(&s)
			}
			due, date := ReminderEmailDue(s, parseAt(t, utc, tt.nowUTC))
			if due != tt.wantDue {
				t.Errorf("due = %v, want %v", due, tt.wantDue)
			}
			if date != tt.wantDate {
				t.Errorf("date = %q, want %q", date, tt.wantDate)
			}
		})
	}
}

func TestReminderEmailSettingsUsesRealSubscription(t *testing.T) {
	trial := time.Now().UTC().Add(24 * time.Hour)
	ended := time.Now().UTC().Add(-GracePeriod - 24*time.Hour)

	active := &User{SubscriptionStatus: SubscriptionTrialing, TrialEndsAt: &trial, ReminderEmailEnabled: true, ReminderEmailTime: "08:00"}
	lapsed := &User{SubscriptionStatus: SubscriptionTrialing, TrialEndsAt: &ended, ReminderEmailEnabled: true, ReminderEmailTime: "08:00"}
	free := &User{ReminderEmailEnabled: true, ReminderEmailTime: "08:00"}

	now := time.Now().UTC()
	// A moment on today's date late enough that the time-of-day check passes.
	late := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, time.UTC)

	if due, _ := ReminderEmailDue(active.ReminderEmailSettings(), late); !due {
		t.Error("trialing user should be due")
	}
	if due, _ := ReminderEmailDue(lapsed.ReminderEmailSettings(), late); due {
		t.Error("lapsed user must not be due")
	}
	if due, _ := ReminderEmailDue(free.ReminderEmailSettings(), late); due {
		t.Error("user with no plan must not be due")
	}
}

func TestValidReminderEmailTime(t *testing.T) {
	for in, want := range map[string]bool{
		"08:00": true, "00:00": true, "23:59": true,
		"24:00": false, "8:00": false, "08:60": false, "": false, "08:00:00": false, "ab:cd": false,
	} {
		if got := ValidReminderEmailTime(in); got != want {
			t.Errorf("ValidReminderEmailTime(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizeTimezone(t *testing.T) {
	for in, want := range map[string]string{
		"": "UTC", "Nope": "UTC", "America/Chicago": "America/Chicago", "UTC": "UTC",
	} {
		if got := NormalizeTimezone(in); got != want {
			t.Errorf("NormalizeTimezone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanReminderEmailMessage(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"trims", "  hello  ", "hello", false},
		{"newline ok", "a\nb", "a\nb", false},
		{"exactly max", strings.Repeat("a", 280), strings.Repeat("a", 280), false},
		{"too long", strings.Repeat("a", 281), "", true},
		{"counts runes not bytes", strings.Repeat("é", 280), strings.Repeat("é", 280), false},
		{"control char", "a\x00b", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanReminderEmailMessage(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
