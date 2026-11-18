package jobs

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"journal/internal/emails"
	"journal/internal/models"
	"journal/internal/testutil"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	testCryptoKey = "0123456789abcdef0123456789abcdef" // 32 bytes
	// Kiritimati is UTC+14 all year, so its local date is often ahead of UTC.
	aheadTZ = "Pacific/Kiritimati"
)

// useDB points the jobs package at db for the test and restores it after.
func useDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	SetDB(db)
	t.Cleanup(func() { SetDB(nil) })
}

var emailSeq int

func uniqueEmail() string {
	emailSeq++
	return fmt.Sprintf("reminder-%s-%d@example.com", uuid.NewString()[:8], emailSeq)
}

// dueUser builds a subscribed, confirmed user with the reminder email on at
// 00:00 in a timezone ahead of UTC, so it is due whenever the test runs.
func dueUser(t *testing.T, tx *gorm.DB, mutate ...func(*models.User)) *models.User {
	t.Helper()
	confirmed := time.Now().Add(-time.Hour)
	all := append([]func(*models.User){func(u *models.User) {
		u.SubscriptionStatus = models.SubscriptionComped
		u.EmailConfirmedAt = &confirmed
		u.ReminderEmailEnabled = true
		u.ReminderEmailTime = "00:00"
		u.Timezone = aheadTZ
	}}, mutate...)
	return testutil.User(t, tx, uniqueEmail(), all...)
}

func localToday(tz string) string {
	loc, _ := time.LoadLocation(tz)
	return time.Now().In(loc).Format("2006-01-02")
}

func reminderJobs(t *testing.T, db *gorm.DB, userID uuid.UUID) []models.Job {
	t.Helper()
	var jobs []models.Job
	if err := db.Where("creator_id = ? AND type = ?", userID, EMAIL_JOB_TYPE).Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	return jobs
}

func lastSentOn(t *testing.T, db *gorm.DB, userID uuid.UUID) *string {
	t.Helper()
	var s *string
	row := db.Raw("SELECT to_char(reminder_email_last_sent_on, 'YYYY-MM-DD') FROM users WHERE id = ?", userID).Row()
	if err := row.Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestScheduleDueReminderEmails_Queues(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		useDB(t, tx)
		u := dueUser(t, tx)

		scheduleDueReminderEmails()

		jobs := reminderJobs(t, tx, u.ID)
		if len(jobs) != 1 {
			t.Fatalf("jobs = %d, want 1", len(jobs))
		}
		if jobs[0].ScheduledAt == nil || jobs[0].ProcessedAt != nil {
			t.Errorf("job should be scheduled and unprocessed: %+v", jobs[0])
		}
		err, data := models.CastMetadata[EmailData](jobs[0].Metadata)
		if err != nil {
			t.Fatal(err)
		}
		if data.Name != emails.REMINDER_EMAIL || len(data.Recipients) != 1 || data.Recipients[0].Email != u.Email {
			t.Errorf("unexpected job data: %+v", data)
		}
		got := lastSentOn(t, tx, u.ID)
		if want := localToday(aheadTZ); got == nil || *got != want {
			t.Errorf("last_sent_on = %v, want local date %s", got, want)
		}

		// Same day again: nothing more.
		scheduleDueReminderEmails()
		if n := len(reminderJobs(t, tx, u.ID)); n != 1 {
			t.Errorf("after second run jobs = %d, want 1", n)
		}
	})
}

func TestScheduleDueReminderEmails_NotQueued(t *testing.T) {
	// Local minute-of-day now, to build a "not yet due" time. Skip the one
	// minute a day where no later time exists.
	loc, _ := time.LoadLocation(aheadTZ)
	nowLocal := time.Now().In(loc)
	later := "23:59"
	notYetOK := !(nowLocal.Hour() == 23 && nowLocal.Minute() == 59)

	longAgo := time.Now().Add(-100 * 24 * time.Hour)
	tests := []struct {
		name   string
		mutate func(*models.User)
		skip   bool
	}{
		{"disabled", func(u *models.User) { u.ReminderEmailEnabled = false }, false},
		{"lapsed", func(u *models.User) {
			u.SubscriptionStatus = models.SubscriptionCanceled
			u.CurrentPeriodEnd = &longAgo
		}, false},
		{"unconfirmed", func(u *models.User) { u.EmailConfirmedAt = nil }, false},
		{"not yet due locally", func(u *models.User) { u.ReminderEmailTime = later }, !notYetOK},
		{"already sent today", func(u *models.User) {
			d, _ := time.Parse("2006-01-02", localToday(aheadTZ))
			u.ReminderEmailLastSentOn = &d
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skip("wall clock makes this case impossible right now")
			}
			testutil.Tx(t, func(tx *gorm.DB) {
				useDB(t, tx)
				u := dueUser(t, tx, tc.mutate)
				before := lastSentOn(t, tx, u.ID)

				scheduleDueReminderEmails()

				if n := len(reminderJobs(t, tx, u.ID)); n != 0 {
					t.Errorf("jobs = %d, want 0", n)
				}
				after := lastSentOn(t, tx, u.ID)
				if (before == nil) != (after == nil) || (before != nil && *before != *after) {
					t.Errorf("last_sent_on changed: %v -> %v", before, after)
				}
			})
		})
	}
}

// Two runners racing for the same user must queue exactly one email. A Tx is
// one connection, so this uses real commits and removes its rows afterward.
func TestQueueReminderEmail_ConcurrentClaimsOnce(t *testing.T) {
	db := testutil.DB(t)
	useDB(t, db)
	u := dueUser(t, db)
	t.Cleanup(func() {
		db.Unscoped().Where("creator_id = ?", u.ID).Delete(&models.Job{})
		db.Unscoped().Where("id = ?", u.ID).Delete(&models.User{})
	})

	today := localToday(aheadTZ)
	const runners = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, runners)
	for i := 0; i < runners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- queueReminderEmail(u, today)
		}()
	}
	// The scheduler entry point racing too.
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			<-start
			scheduleDueReminderEmails()
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("queueReminderEmail: %v", err)
		}
	}

	if n := len(reminderJobs(t, db, u.ID)); n != 1 {
		t.Fatalf("jobs = %d, want exactly 1", n)
	}
	if got := lastSentOn(t, db, u.ID); got == nil || *got != today {
		t.Errorf("last_sent_on = %v, want %s", got, today)
	}
}

// queued runs the scheduler and returns the job it created for u.
func queued(t *testing.T, tx *gorm.DB, u *models.User) *models.Job {
	t.Helper()
	scheduleDueReminderEmails()
	jobs := reminderJobs(t, tx, u.ID)
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	return &jobs[0]
}

func prepare(t *testing.T, job *models.Job) (bool, *emails.EmailConfig) {
	t.Helper()
	config := &emails.EmailConfig{}
	skip, err := prepareReminderEmail(job, &EmailData{}, config)
	if err != nil {
		t.Fatal(err)
	}
	return skip, config
}

func TestPrepareReminderEmail_Skips(t *testing.T) {
	longAgo := time.Now().Add(-100 * 24 * time.Hour)
	tests := []struct {
		name   string
		change map[string]any
	}{
		{"turned off", map[string]any{"reminder_email_enabled": false}},
		{"lapsed", map[string]any{"subscription_status": models.SubscriptionCanceled, "current_period_end": longAgo}},
		{"unconfirmed", map[string]any{"email_confirmed_at": nil}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				useDB(t, tx)
				u := dueUser(t, tx)
				job := queued(t, tx, u)
				if err := tx.Model(&models.User{}).Where("id = ?", u.ID).Updates(tc.change).Error; err != nil {
					t.Fatal(err)
				}
				if skip, _ := prepare(t, job); !skip {
					t.Error("want skip after the user changed")
				}
			})
		})
	}
}

func TestPrepareReminderEmail_DefaultMessage(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		useDB(t, tx)
		job := queued(t, tx, dueUser(t, tx))
		skip, config := prepare(t, job)
		if skip {
			t.Fatal("unexpected skip")
		}
		if config.Subject != emails.DefaultReminderEmailSubject || config.Content == nil {
			t.Errorf("subject = %q, content nil = %v", config.Subject, config.Content == nil)
		}
	})
}

func TestPrepareReminderEmail_CustomMessageNeverInJob(t *testing.T) {
	const msg = "secret-note-xyzzy: take a breath"
	testutil.Tx(t, func(tx *gorm.DB) {
		useDB(t, tx)
		u := dueUser(t, tx, func(u *models.User) {
			t.Setenv("CRYPTO_KEY", testCryptoKey)
			if err := u.SetReminderEmailMessage(msg); err != nil {
				t.Fatal(err)
			}
		})
		if u.ReminderEmailMessage == nil || strings.Contains(*u.ReminderEmailMessage, "xyzzy") {
			t.Fatal("message should be stored encrypted")
		}
		job := queued(t, tx, u)

		// Raw stored metadata, re-read from the database.
		var raw string
		if err := tx.Raw("SELECT metadata::text FROM jobs WHERE id = ?", job.ID).Row().Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(raw, "xyzzy") || strings.Contains(raw, msg) {
			t.Errorf("job metadata contains the plaintext message: %s", raw)
		}

		skip, config := prepare(t, job)
		if skip {
			t.Fatal("unexpected skip")
		}
		if config.Subject != msg {
			t.Errorf("subject = %q, want decrypted message %q", config.Subject, msg)
		}
	})
}
