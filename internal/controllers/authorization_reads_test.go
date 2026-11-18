package controllers

import (
	"encoding/json"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const secretB = "B-PRIVATE-MARKER"

// loaderApp serves path with the given loader handlers followed by a terminal
// handler that returns the named locals as JSON, so the loaders can be tested
// without rendering a page. Every request is authenticated as user.
func loaderApp(t *testing.T, db *gorm.DB, user *models.User, c Controller, path string, keys []string, loaders ...fiber.Handler) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(ctx *fiber.Ctx) error {
		ctx.Locals(utils.LocalCurrentUser, user)
		return ctx.Next()
	})
	c.Init(db, app)
	terminal := func(ctx *fiber.Ctx) error {
		out := map[string]any{}
		for _, k := range keys {
			out[k] = ctx.Locals(k)
		}
		return ctx.JSON(out)
	}
	app.Get(path, append(loaders, terminal)...)
	return app
}

// fetch GETs path and returns the body, failing unless the status is 200.
func fetch(t *testing.T, app *fiber.App, path string) string {
	t.Helper()
	code, body := send(t, app, http.MethodGet, path, nil)
	if code != http.StatusOK {
		t.Fatalf("GET %s: %d %s", path, code, body)
	}
	return body
}

// onlyOwn asserts body holds own and not other.
func onlyOwn(t *testing.T, what, body, own, other string) {
	t.Helper()
	if !strings.Contains(body, own) {
		t.Errorf("%s: missing own marker %q in %s", what, own, body)
	}
	if strings.Contains(body, other) {
		t.Errorf("%s: leaked other user's marker %q in %s", what, other, body)
	}
}

func journalAt(t *testing.T, db *gorm.DB, u *models.User, date time.Time, marker string) *models.Journal {
	t.Helper()
	j := &models.Journal{Base: baseFor(u), Date: &date, Entry: "entry " + marker, JournalTypeID: 1, RatingID: 1}
	if err := db.Create(j).Error; err != nil {
		t.Fatal(err)
	}
	// Neighbouring entries are ordered by created_at, so make it follow date.
	if err := db.Exec("UPDATE journals SET created_at = ? WHERE id = ?", date, j.ID).Error; err != nil {
		t.Fatal(err)
	}
	return j
}

func pair(a, b *models.User) [2]struct {
	u         *models.User
	own, them string
} {
	return [2]struct {
		u         *models.User
		own, them string
	}{{a, secretA, secretB}, {b, secretB, secretA}}
}

func TestReadsAreScopedToCurrentUser(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	a, b := newUser(), newUser()

	// One of each record per user, each carrying its owner's marker.
	for _, p := range pair(a, b) {
		journalAt(t, db, p.u, time.Now().UTC(), p.own)
		for _, rec := range []any{
			&models.ActionItem{Base: baseFor(p.u), Text: "todo " + p.own},
			&models.Thankful{Base: baseFor(p.u), Text: "thanks " + p.own},
			&models.Reminder{Base: baseFor(p.u), Text: "remember " + p.own},
			&models.CustomJournalType{Base: baseFor(p.u), Name: p.own},
		} {
			if err := db.Create(rec).Error; err != nil {
				t.Fatal(err)
			}
		}
		start := time.Now().UTC()
		if err := db.Create(&models.RecurringActionItem{Base: baseFor(p.u), Text: "routine " + p.own, Frequency: 24 * time.Hour, StartsAt: &start}).Error; err != nil {
			t.Fatal(err)
		}
	}

	j, ai := &JournalController{}, &ActionItemController{}
	for _, p := range pair(a, b) {
		t.Run("getJournals", func(t *testing.T) {
			app := loaderApp(t, db, p.u, j, "/x", []string{utils.LocalJournals}, j.getJournals)
			onlyOwn(t, "getJournals", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("setOutstandingActionItems", func(t *testing.T) {
			app := loaderApp(t, db, p.u, j, "/x", []string{utils.LocalOutstandingActionItems}, j.setOutstandingActionItems)
			onlyOwn(t, "setOutstandingActionItems", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("getActionItems", func(t *testing.T) {
			app := loaderApp(t, db, p.u, ai, "/x", []string{utils.LocalActionItems}, ai.getActionItems)
			onlyOwn(t, "getActionItems", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("getRecurringActionItems", func(t *testing.T) {
			app := loaderApp(t, db, p.u, ai, "/x", []string{utils.LocalRecurringActionItems}, ai.getRecurringActionItems)
			onlyOwn(t, "getRecurringActionItems", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("getThankfuls", func(t *testing.T) {
			th := &ThankfulController{}
			app := loaderApp(t, db, p.u, th, "/x", []string{utils.LocalThankfuls}, th.getThankfuls)
			onlyOwn(t, "getThankfuls", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("getReminders", func(t *testing.T) {
			r := &ReminderController{}
			app := loaderApp(t, db, p.u, r, "/x", []string{utils.LocalReminders}, r.getReminders)
			onlyOwn(t, "getReminders", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("getCustomJournalTypes", func(t *testing.T) {
			cj := &CustomJournalTypeController{}
			app := loaderApp(t, db, p.u, cj, "/x", []string{utils.LocalCustomJournalTypes}, cj.getCustomJournalTypes)
			onlyOwn(t, "getCustomJournalTypes", fetch(t, app, "/x"), p.own, p.them)
		})
		t.Run("exportJournals", func(t *testing.T) {
			app := newAuthedTestApp(t, db, p.u, &ProfileController{})
			for _, f := range []string{"json", "md"} {
				onlyOwn(t, "export "+f, fetch(t, app, "/api/profile/export?format="+f), p.own, p.them)
			}
		})
	}
}

func TestShuffleReminderOnlyPicksOwn(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	a, b := newUser(), newUser()
	reminderFor(t, db, a) // B has none

	rc := &ReminderController{}
	app := newAuthedTestApp(t, db, b, rc)
	app.Get("/shuffle", rc.shuffleReminder)

	code, body := send(t, app, http.MethodGet, "/shuffle", nil)
	if code != http.StatusNoContent || strings.Contains(body, secretA) {
		t.Errorf("B shuffled to %d %q, want 204 and nothing of A's", code, body)
	}

	// With one of their own, B still never gets A's.
	rid := reminderFor(t, db, b)
	code, body = send(t, app, http.MethodGet, "/shuffle?current="+rid, nil)
	if code != http.StatusNoContent || strings.Contains(body, secretA) {
		t.Errorf("B (one reminder) shuffled to %d %q, want 204 and nothing of A's", code, body)
	}
}

func TestSurroundingJournalsAreOwn(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	a, b := newUser(), newUser()
	day := func(n int) time.Time { return time.Date(2024, 3, n, 12, 0, 0, 0, time.UTC) }

	// A's entries sit right before and after B's single entry.
	journalAt(t, db, a, day(9), secretA+"-before")
	mine := journalAt(t, db, b, day(10), secretB)
	journalAt(t, db, a, day(11), secretA+"-after")

	jc := &JournalController{}
	app := loaderApp(t, db, b, jc, "/j/:id", []string{utils.LocalJournal}, jc.getJournal, jc.getSurroundingJournals)
	body := fetch(t, app, "/j/"+mine.ID.String())
	if !strings.Contains(body, secretB) || strings.Contains(body, secretA) {
		t.Errorf("neighbours of B's only entry leaked A's: %s", body)
	}

	// With B's own neighbours present, those are the ones linked.
	journalAt(t, db, b, day(8), secretB+"-prev")
	journalAt(t, db, b, day(12), secretB+"-next")
	body = fetch(t, app, "/j/"+mine.ID.String())
	if !strings.Contains(body, secretB+"-prev") || !strings.Contains(body, secretB+"-next") || strings.Contains(body, secretA) {
		t.Errorf("wrong neighbours: %s", body)
	}
}

func TestGetJournalsDayPagingIsOwn(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	a, b := newUser(), newUser()
	day := func(n int) time.Time { return time.Date(2024, 3, n, 12, 0, 0, 0, time.UTC) }

	// A has entries on days 1, 5 and 9; B only on day 5.
	journalAt(t, db, a, day(1), secretA+"-d1")
	journalAt(t, db, a, day(5), secretA+"-d5")
	journalAt(t, db, a, day(9), secretA+"-d9")
	journalAt(t, db, b, day(5), secretB)

	jc := &JournalController{}
	keys := []string{utils.LocalJournals, utils.LocalHasMore, utils.LocalPrevDate, utils.LocalNextDate}
	app := loaderApp(t, db, b, jc, "/x", keys, jc.getJournals)

	t.Run("sort=date shows B's newest day, not the newest day of everyone", func(t *testing.T) {
		body := fetch(t, app, "/x?sort=date")
		if !strings.Contains(body, secretB) || strings.Contains(body, secretA) {
			t.Errorf("sort=date: %s", body)
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(body), &out); err != nil {
			t.Fatal(err)
		}
		if out[utils.LocalHasMore] != nil && out[utils.LocalHasMore] != false {
			t.Errorf("hasMore = %v, B has only one day", out[utils.LocalHasMore])
		}
		if out[utils.LocalPrevDate] != nil || out[utils.LocalNextDate] != nil {
			t.Errorf("B has no other days but got prev=%v next=%v (A's days)", out[utils.LocalPrevDate], out[utils.LocalNextDate])
		}
	})

	t.Run("date=day has no prev or next from A's days", func(t *testing.T) {
		body := fetch(t, app, "/x?date=2024-03-05")
		var out map[string]any
		if err := json.Unmarshal([]byte(body), &out); err != nil {
			t.Fatal(err)
		}
		if out[utils.LocalPrevDate] != nil || out[utils.LocalNextDate] != nil {
			t.Errorf("prev=%v next=%v, want none (those are A's days)", out[utils.LocalPrevDate], out[utils.LocalNextDate])
		}
		if strings.Contains(body, secretA) {
			t.Errorf("leaked A: %s", body)
		}
	})
}

func TestCreateJournalDraftIDIsPerUser(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	a, b := newUser(), newUser()

	postJournal(t, db, a, "A's entry "+secretA, "draft-x")
	postJournal(t, db, b, "B's entry "+secretB, "draft-x")

	var aj, bj models.Journal
	if err := db.Where("creator_id = ?", a.ID).First(&aj).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("creator_id = ?", b.ID).First(&bj).Error; err != nil {
		t.Fatalf("B got no entry of their own: %v", err)
	}
	if aj.ID == bj.ID || !strings.Contains(aj.Entry, secretA) || strings.Contains(aj.Entry, secretB) {
		t.Errorf("A's entry changed: %q", aj.Entry)
	}
	if !strings.Contains(bj.Entry, secretB) {
		t.Errorf("B's entry = %q", bj.Entry)
	}
}
