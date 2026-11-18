package jobs

import (
	"testing"

	"journal/internal/models"
	"journal/internal/testutil"

	"gorm.io/gorm"
)

// The user struct is loaded by the scheduler, then the DB changes before the
// claim runs. queueReminderEmail must trust the DB, not the stale struct.
func TestQueueReminderEmail_ClaimGuards(t *testing.T) {
	tests := []struct {
		name       string
		change     map[string]any // applied to the DB after the struct was loaded
		wantQueued bool
	}{
		{"control: enabled and confirmed", nil, true},
		{"turned off after load", map[string]any{"reminder_email_enabled": false}, false},
		{"unconfirmed after load", map[string]any{"email_confirmed_at": nil}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				useDB(t, tx)
				u := dueUser(t, tx) // struct still says enabled + confirmed
				today := localToday(aheadTZ)
				if tc.change != nil {
					if err := tx.Model(&models.User{}).Where("id = ?", u.ID).Updates(tc.change).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := queueReminderEmail(u, today); err != nil {
					t.Fatal(err)
				}
				jobs := reminderJobs(t, tx, u.ID)
				got := lastSentOn(t, tx, u.ID)
				if tc.wantQueued {
					if len(jobs) != 1 {
						t.Fatalf("jobs = %d, want 1", len(jobs))
					}
					if got == nil || *got != today {
						t.Errorf("last_sent_on = %v, want %s", got, today)
					}
					return
				}
				if len(jobs) != 0 {
					t.Errorf("jobs = %d, want 0", len(jobs))
				}
				if got != nil {
					t.Errorf("last_sent_on = %s, want unchanged (nil)", *got)
				}
			})
		})
	}
}
