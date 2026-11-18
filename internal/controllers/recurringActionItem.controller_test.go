package controllers

import (
	"journal/internal/jobs"
	"journal/internal/models"
	"journal/internal/testutil"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var routineNow = time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)

const routineText = "stretch before bed (private routine text)"

func routineController(t *testing.T, tx *gorm.DB) *RecurringActionItemController {
	t.Helper()
	t.Setenv("CRYPTO_KEY", "0123456789abcdef0123456789abcdef")
	jobs.SetDB(tx)
	t.Cleanup(func() { jobs.SetDB(nil) })
	cronsEnabled = false // no background scheduler per test controller
	c := &RecurringActionItemController{}
	c.Init(tx, fiber.New())
	return c
}

func confirmedUser(t *testing.T, tx *gorm.DB, email string, mutate ...func(*models.User)) *models.User {
	t.Helper()
	confirmed := routineNow.Add(-time.Hour)
	all := append([]func(*models.User){func(u *models.User) {
		u.SubscriptionStatus = models.SubscriptionComped
		u.EmailConfirmedAt = &confirmed
	}}, mutate...)
	return testutil.User(t, tx, email, all...)
}

// dueRoutineFor inserts a daily routine owned by u that started three days before
// routineNow.
func dueRoutineFor(t *testing.T, tx *gorm.DB, u *models.User, mutate ...func(*models.RecurringActionItem)) *models.RecurringActionItem {
	t.Helper()
	start := routineNow.Add(-3 * 24 * time.Hour)
	r := &models.RecurringActionItem{
		Base:      &models.Base{CreatorID: &u.ID, LastUpdaterID: &u.ID},
		Text:      routineText,
		Frequency: 24 * time.Hour,
		StartsAt:  &start,
	}
	for _, m := range mutate {
		m(r)
	}
	if err := tx.Create(r).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func itemsFor(t *testing.T, tx *gorm.DB, r *models.RecurringActionItem) int64 {
	t.Helper()
	return rawCount(t, tx, "action_items", "recurring_action_item_id = ?", r.ID)
}

func emailJobsFor(t *testing.T, tx *gorm.DB, u *models.User) int64 {
	t.Helper()
	return rawCount(t, tx, "jobs", "creator_id = ?", u.ID)
}

func TestCreateActionItems(t *testing.T) {
	tests := []struct {
		name      string
		owner     func(*models.User)
		routine   func(*models.RecurringActionItem)
		wantItems int64
	}{
		{"subscribed owner, never come up: created", nil, nil, 1},
		{"lapsed owner: waits", func(u *models.User) {
			u.SubscriptionStatus = models.SubscriptionCanceled
			ended := routineNow.Add(-100 * 24 * time.Hour)
			u.CurrentPeriodEnd = &ended
		}, nil, 0},
		{"not started yet: waits", nil, func(r *models.RecurringActionItem) {
			later := routineNow.Add(time.Hour)
			r.StartsAt = &later
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				c := routineController(t, tx)
				var ownerMut []func(*models.User)
				if tt.owner != nil {
					ownerMut = append(ownerMut, tt.owner)
				}
				u := confirmedUser(t, tx, "routine-"+uuid.NewString()[:8]+"@example.com", ownerMut...)
				var routMut []func(*models.RecurringActionItem)
				if tt.routine != nil {
					routMut = append(routMut, tt.routine)
				}
				r := dueRoutineFor(t, tx, u, routMut...)

				c.createActionItemsAt(routineNow)

				if got := itemsFor(t, tx, r); got != tt.wantItems {
					t.Fatalf("action items = %d, want %d", got, tt.wantItems)
				}
			})
		})
	}
}

func TestCreateActionItemsCreatesTheRightItemOnce(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		c := routineController(t, tx)
		u := confirmedUser(t, tx, "once@example.com")
		r := dueRoutineFor(t, tx, u)

		c.createActionItemsAt(routineNow)
		c.createActionItemsAt(routineNow.Add(time.Minute)) // the next cron tick: the first is still open

		var items []models.ActionItem
		if err := tx.Where("recurring_action_item_id = ?", r.ID).Find(&items).Error; err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("action items = %d, want exactly 1 after two ticks", len(items))
		}
		got := items[0]
		if got.CreatorID == nil || *got.CreatorID != u.ID {
			t.Errorf("creator = %v, want the routine's owner", got.CreatorID)
		}
		if got.Text != routineText {
			t.Errorf("text = %q, want the routine's text (decrypted on load)", got.Text)
		}
		if raw := rawCol(t, tx, "action_items", "text", got.ID.String()); strings.Contains(raw, "private routine text") {
			t.Error("the action item text is stored in plaintext")
		}
	})
}

func TestCreateActionItemsEmailsOwners(t *testing.T) {
	emailMe := func(r *models.RecurringActionItem) { r.EmailMe = true }
	tests := []struct {
		name     string
		owner    func(*models.User)
		routines []func(*models.RecurringActionItem)
		wantJobs int64
	}{
		{"email me, confirmed address: one email", nil, []func(*models.RecurringActionItem){emailMe}, 1},
		{"two of their routines come up together: still one email", nil,
			[]func(*models.RecurringActionItem){emailMe, emailMe}, 1},
		{"one of two asks for email: one email", nil,
			[]func(*models.RecurringActionItem){emailMe, func(*models.RecurringActionItem) {}}, 1},
		{"not asked for: no email", nil, []func(*models.RecurringActionItem){func(*models.RecurringActionItem) {}}, 0},
		{"asked for but address not confirmed: no email", func(u *models.User) { u.EmailConfirmedAt = nil },
			[]func(*models.RecurringActionItem){emailMe}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				c := routineController(t, tx)
				var ownerMut []func(*models.User)
				if tt.owner != nil {
					ownerMut = append(ownerMut, tt.owner)
				}
				u := confirmedUser(t, tx, "emailme-"+uuid.NewString()[:8]+"@example.com", ownerMut...)
				for _, m := range tt.routines {
					dueRoutineFor(t, tx, u, m)
				}

				c.createActionItemsAt(routineNow)

				if got := emailJobsFor(t, tx, u); got != tt.wantJobs {
					t.Fatalf("email jobs = %d, want %d", got, tt.wantJobs)
				}
				// Jobs are stored unencrypted, so the routine's text must never be in one.
				if n := rawCount(t, tx, "jobs", "creator_id = ? AND metadata::text LIKE ?", u.ID, "%private routine text%"); n != 0 {
					t.Error("a job row contains the routine's text")
				}
			})
		})
	}
}

// An owner whose plan has lapsed gets neither the item nor the email.
func TestCreateActionItemsLapsedOwnerGetsNoEmail(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		c := routineController(t, tx)
		u := confirmedUser(t, tx, "lapsed-email@example.com", func(u *models.User) {
			u.SubscriptionStatus = models.SubscriptionCanceled
			ended := routineNow.Add(-100 * 24 * time.Hour)
			u.CurrentPeriodEnd = &ended
		})
		dueRoutineFor(t, tx, u, func(r *models.RecurringActionItem) { r.EmailMe = true })

		c.createActionItemsAt(routineNow)

		if got := emailJobsFor(t, tx, u); got != 0 {
			t.Fatalf("email jobs = %d for a lapsed owner, want 0", got)
		}
	})
}
