package controllers

import (
	"fmt"
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	secretA = "A-PRIVATE-MARKER"
	attackB = "B-ATTACK-MARKER"
)

func baseFor(u *models.User) *models.Base {
	return &models.Base{CreatorID: &u.ID, LastUpdaterID: &u.ID}
}

// Builders: each inserts one record owned by u whose plaintext contains
// secretA, and returns its id.

func journalFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	now := time.Now().UTC()
	j := &models.Journal{Base: baseFor(u), Date: &now, Entry: "entry " + secretA, JournalTypeID: 1, RatingID: 1}
	if err := tx.Create(j).Error; err != nil {
		t.Fatal(err)
	}
	return j.ID.String()
}

func actionItemFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	a := &models.ActionItem{Base: baseFor(u), Text: "todo " + secretA}
	if err := tx.Create(a).Error; err != nil {
		t.Fatal(err)
	}
	return a.ID.String()
}

func thankfulFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	th := &models.Thankful{Base: baseFor(u), Text: "thanks " + secretA}
	if err := tx.Create(th).Error; err != nil {
		t.Fatal(err)
	}
	return th.ID.String()
}

func reminderFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	r := &models.Reminder{Base: baseFor(u), Text: "remember " + secretA}
	if err := tx.Create(r).Error; err != nil {
		t.Fatal(err)
	}
	return r.ID.String()
}

func routineFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	start := time.Now().UTC()
	r := &models.RecurringActionItem{Base: baseFor(u), Text: "routine " + secretA, Frequency: 24 * time.Hour, StartsAt: &start}
	if err := tx.Create(r).Error; err != nil {
		t.Fatal(err)
	}
	return r.ID.String()
}

func entryTemplateFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	e := &models.EntryTemplate{Base: baseFor(u), Name: "name " + secretA, Body: "body " + secretA}
	if err := tx.Create(e).Error; err != nil {
		t.Fatal(err)
	}
	return e.ID.String()
}

func customTypeFor(t *testing.T, tx *gorm.DB, u *models.User) string {
	t.Helper()
	c := &models.CustomJournalType{Base: baseFor(u), Name: secretA}
	if err := tx.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	return c.ID.String()
}

func form(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i < len(kv); i += 2 {
		v.Add(kv[i], kv[i+1])
	}
	return v
}

type authzCase struct {
	name       string
	table      string
	controller func() Controller
	create     func(*testing.T, *gorm.DB, *models.User) string
	method     string
	path       string // %s is the record id
	body       func(marker string) url.Values
}

func authzCases() []authzCase {
	journalBody := func(m string) url.Values {
		return form("entry", m, "journalType", "1", "rating", "1", "autosave", "true")
	}
	textBody := func(m string) url.Values { return form("text", m) }
	routineBody := func(m string) url.Values { return form("text", m, "hourlyFrequency", "24") }
	templateBody := func(m string) url.Values { return form("name", m, "body", m) }
	nameBody := func(m string) url.Values { return form("name", m) }

	type spec struct {
		name, table, path string
		controller        func() Controller
		create            func(*testing.T, *gorm.DB, *models.User) string
		body              func(string) url.Values
	}
	specs := []spec{
		{"journal", "journals", "/api/journal/%s", func() Controller { return &JournalController{} }, journalFor, journalBody},
		{"actionItem", "action_items", "/api/action-items/%s", func() Controller { return &ActionItemController{} }, actionItemFor, textBody},
		{"thankful", "thankfuls", "/api/thankfuls/%s", func() Controller { return &ThankfulController{} }, thankfulFor, textBody},
		{"reminder", "reminders", "/api/reminders/%s", func() Controller { return &ReminderController{} }, reminderFor, textBody},
		{"recurringActionItem", "recurring_action_items", "/api/recurring-action-items/%s", func() Controller { return &RecurringActionItemController{} }, routineFor, routineBody},
		{"entryTemplate", "entry_templates", "/api/entry-templates/%s", func() Controller { return &EntryTemplateController{} }, entryTemplateFor, templateBody},
		{"customJournalType", "custom_journal_types", "/api/custom-journal-types/%s", func() Controller { return &CustomJournalTypeController{} }, customTypeFor, nameBody},
	}
	var cases []authzCase
	for _, s := range specs {
		for _, m := range []string{http.MethodPut, http.MethodDelete} {
			cases = append(cases, authzCase{s.name + "/" + m, s.table, s.controller, s.create, m, s.path, s.body})
		}
	}
	return cases
}

// TestCrossUserAuthorization: user B must not be able to read, change or
// delete a record user A owns, through any id-addressed API route. After each
// refused attempt the same request made by A must succeed, so a pass can't be
// because every request fails.
func TestCrossUserAuthorization(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	for _, tc := range authzCases() {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				a := compedUser(t, tx, "a@example.com")
				b := compedUser(t, tx, "b@example.com")
				id := tc.create(t, tx, a)
				path := fmt.Sprintf(tc.path, id)
				before := rawRow(t, tx, tc.table, id)
				if before == "" {
					t.Fatal("record was not created")
				}

				// B tries it.
				appB := newAuthedTestApp(t, tx, b, tc.controller())
				code, body := send(t, appB, tc.method, path, tc.body(attackB))
				if code != http.StatusNotFound && code != http.StatusForbidden {
					t.Errorf("B got status %d, want 404 or 403", code)
				}
				if strings.Contains(body, secretA) {
					t.Errorf("response to B leaks A's data: %s", body)
				}
				if after := rawRow(t, tx, tc.table, id); after != before {
					t.Errorf("A's record changed or was deleted by B:\nbefore %s\nafter  %s", before, after)
				}

				// Control: the same request from A works.
				appA := newAuthedTestApp(t, tx, a, tc.controller())
				code, body = send(t, appA, tc.method, path, tc.body("A-EDITED"))
				// Handlers save first and then render a templ fragment, which
				// fails under app.Test (the request context is already
				// cancelled). The save is what matters here.
				if !is2xx(code) && body != "context canceled" {
					t.Fatalf("A got status %d on own record, want 2xx: %.300s", code, body)
				}
				after := rawRow(t, tx, tc.table, id)
				if tc.method == http.MethodDelete {
					if rawCount(t, tx, tc.table, "id = ? AND deleted_at IS NULL", id) != 0 {
						t.Error("A's own delete did not delete")
					}
				} else if after == before {
					t.Error("A's own update did not change the record")
				}
			})
		})
	}
}

// B's list/read endpoints must only show B's records.
func TestEntryTemplatePickerIsScopedToUser(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	testutil.Tx(t, func(tx *gorm.DB) {
		a := compedUser(t, tx, "a@example.com")
		b := compedUser(t, tx, "b@example.com")
		id := entryTemplateFor(t, tx, a)

		code, body := send(t, newAuthedTestApp(t, tx, b, &EntryTemplateController{}), http.MethodGet, "/api/entry-templates", nil)
		if code != http.StatusOK || strings.Contains(body, secretA) || strings.Contains(body, id) {
			t.Errorf("B's picker: status %d, body %s", code, body)
		}
		code, body = send(t, newAuthedTestApp(t, tx, a, &EntryTemplateController{}), http.MethodGet, "/api/entry-templates", nil)
		if code != http.StatusOK || !strings.Contains(body, secretA) {
			t.Errorf("A's picker: status %d, body %s", code, body)
		}
	})
}

// Creating a record that points at someone else's journal, custom topic,
// action item or thankful must not link to it.
func TestCrossUserReferences(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)

	t.Run("action item and thankful cannot attach to another user's journal", func(t *testing.T) {
		testutil.Tx(t, func(tx *gorm.DB) {
			a := compedUser(t, tx, "a@example.com")
			b := compedUser(t, tx, "b@example.com")
			journalID := journalFor(t, tx, a)
			for _, tc := range []struct {
				path, table string
				c           Controller
			}{
				{"/api/action-items", "action_items", &ActionItemController{}},
				{"/api/thankfuls", "thankfuls", &ThankfulController{}},
			} {
				code, body := send(t, newAuthedTestApp(t, tx, b, tc.c), http.MethodPost, tc.path, form("text", attackB, "journalId", journalID))
				if is2xx(code) || strings.Contains(body, secretA) {
					t.Errorf("%s: B attached to A's journal: %d %s", tc.table, code, body)
				}
				if n := rawCount(t, tx, tc.table, "journal_id = ?", journalID); n != 0 {
					t.Errorf("%s: %d rows now point at A's journal", tc.table, n)
				}
			}
		})
	})

	t.Run("journal cannot use another user's custom topic", func(t *testing.T) {
		{
			tx, newUser := committedUsers(t)
			a, b := newUser(), newUser()
			topicID := customTypeFor(t, tx, a)
			code, body := send(t, newAuthedTestApp(t, tx, b, &JournalController{}), http.MethodPost, "/api/journal",
				form("date", "1700000000000", "entry", attackB, "journalType", topicID, "rating", "1"))
			if is2xx(code) || strings.Contains(body, secretA) {
				t.Errorf("B used A's topic: %d %s", code, body)
			}
			if n := rawCount(t, tx, "journals", "custom_journal_type_id = ?", topicID); n != 0 {
				t.Errorf("%d journals reference A's topic", n)
			}
		}
	})

	t.Run("journal cannot adopt another user's action items or thankfuls", func(t *testing.T) {
		{
			tx, newUser := committedUsers(t)
			a, b := newUser(), newUser()
			actionID := actionItemFor(t, tx, a)
			thankfulID := thankfulFor(t, tx, a)
			bJournal := journalFor(t, tx, b)
			app := newAuthedTestApp(t, tx, b, &JournalController{})

			code, body := send(t, app, http.MethodPost, "/api/journal", url.Values{
				"date": {"1700000000000"}, "entry": {attackB}, "journalType": {"1"}, "rating": {"1"}, "autosave": {"true"},
				"actionItemIds": {actionID}, "thankfulIds": {thankfulID},
			})
			if !is2xx(code) || strings.Contains(body, secretA) {
				t.Errorf("create: %d %s", code, body)
			}
			code, body = send(t, app, http.MethodPut, "/api/journal/"+bJournal, url.Values{
				"date": {"1700000000000"}, "entry": {attackB}, "journalType": {"1"}, "rating": {"1"}, "autosave": {"true"},
				"actionItemIds": {actionID}, "thankfulIds": {thankfulID},
			})
			if !is2xx(code) || strings.Contains(body, secretA) {
				t.Errorf("update: %d %s", code, body)
			}
			for table, id := range map[string]string{"action_items": actionID, "thankfuls": thankfulID} {
				if n := rawCount(t, tx, table, "id = ? AND journal_id IS NOT NULL", id); n != 0 {
					t.Errorf("%s: B's journal adopted A's record", table)
				}
			}
		}
	})
}

// Unauthenticated writes are refused, never served.
func TestUnauthenticatedAPIIsRefused(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		for _, c := range []struct {
			name string
			c    Controller
			path string
		}{
			{"journal", &JournalController{}, "/api/journal/" + uuid.NewString()},
			{"reminder", &ReminderController{}, "/api/reminders/" + uuid.NewString()},
		} {
			app := newTestApp(t, tx, c.c)
			code, _ := send(t, app, http.MethodDelete, c.path, nil)
			if code != http.StatusUnauthorized {
				t.Errorf("%s: status %d, want 401", c.name, code)
			}
		}
	})
}
