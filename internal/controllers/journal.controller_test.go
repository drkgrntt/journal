package controllers

import (
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// journalForm is a valid new-entry body. Pass draftID "" to omit it.
func journalForm(entry, draftID string) url.Values {
	v := form("date", "1700000000000", "entry", entry, "journalType", "1", "rating", "1", "autosave", "true")
	if draftID != "" {
		v.Set("draftId", draftID)
	}
	return v
}

func postJournal(t *testing.T, db *gorm.DB, u *models.User, entry, draftID string) {
	t.Helper()
	code, body := send(t, newAuthedTestApp(t, db, u, &JournalController{}), http.MethodPost, "/api/journal", journalForm(entry, draftID))
	if !is2xx(code) {
		t.Fatalf("POST /api/journal: %d %s", code, body)
	}
}

func TestCreateJournalDraftDedup(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)

	t.Run("same user and draft id is one row, updated with the latest text", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		postJournal(t, db, u, "first text", "draft-1")
		postJournal(t, db, u, "second text", "draft-1")

		if n := rawCount(t, db, "journals", "creator_id = ?", u.ID); n != 1 {
			t.Fatalf("rows = %d, want 1", n)
		}
		var j models.Journal
		if err := db.Where("creator_id = ?", u.ID).First(&j).Error; err != nil {
			t.Fatal(err)
		}
		if j.Entry != "second text" {
			t.Errorf("entry = %q, want the second text", j.Entry)
		}
		_, meta := models.CastMetadata[models.JournalMetadata](j.Metadata)
		if meta.DraftID != "draft-1" {
			t.Errorf("draft id not kept in metadata: %q", meta.DraftID)
		}
	})

	t.Run("different draft ids make different rows", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		postJournal(t, db, u, "one", "draft-1")
		postJournal(t, db, u, "two", "draft-2")
		if n := rawCount(t, db, "journals", "creator_id = ?", u.ID); n != 2 {
			t.Errorf("rows = %d, want 2", n)
		}
	})

	t.Run("different users with the same draft id do not collide", func(t *testing.T) {
		db, newUser := committedUsers(t)
		a, b := newUser(), newUser()
		postJournal(t, db, a, "A's entry "+secretA, "shared-draft")
		postJournal(t, db, b, "B's entry", "shared-draft")

		for _, u := range []*models.User{a, b} {
			if n := rawCount(t, db, "journals", "creator_id = ?", u.ID); n != 1 {
				t.Errorf("user %s has %d rows, want 1", u.ID, n)
			}
		}
		var j models.Journal
		if err := db.Where("creator_id = ?", a.ID).First(&j).Error; err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(j.Entry, secretA) {
			t.Errorf("B's post overwrote A's entry: %q", j.Entry)
		}
	})

	t.Run("no draft id always creates a new row", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		postJournal(t, db, u, "same", "")
		postJournal(t, db, u, "same", "")
		if n := rawCount(t, db, "journals", "creator_id = ?", u.ID); n != 2 {
			t.Errorf("rows = %d, want 2", n)
		}
	})

	t.Run("a draft id only matches live entries of the same user", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		postJournal(t, db, u, "first", "draft-1")
		var j models.Journal
		if err := db.Where("creator_id = ?", u.ID).First(&j).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Delete(&j).Error; err != nil {
			t.Fatal(err)
		}
		// The entry was deleted, so a late retry of the same draft makes a new one.
		postJournal(t, db, u, "again", "draft-1")
		if n := rawCount(t, db, "journals", "creator_id = ? AND deleted_at IS NULL", u.ID); n != 1 {
			t.Errorf("live rows = %d, want 1", n)
		}
	})
}

func TestJournalEncryptedAtRest(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	u := newUser()

	postJournal(t, db, u, "plain "+secretA, "draft-enc")
	var j models.Journal
	if err := db.Where("creator_id = ?", u.ID).First(&j).Error; err != nil {
		t.Fatal(err)
	}
	id := j.ID.String()

	assertEncrypted := func(step, plain string) {
		t.Helper()
		raw := rawCol(t, db, "journals", "entry", id)
		if raw == "" || strings.Contains(raw, plain) || strings.Contains(raw, "plain") {
			t.Errorf("%s: stored entry is not ciphertext: %q", step, raw)
		}
		if rawCol(t, db, "journals", "is_encrypted", id) != "true" {
			t.Errorf("%s: is_encrypted not set", step)
		}
		var loaded models.Journal
		if err := db.First(&loaded, "id = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if loaded.Entry != plain {
			t.Errorf("%s: loaded entry = %q, want %q", step, loaded.Entry, plain)
		}
	}
	assertEncrypted("create", "plain "+secretA)

	// Update through the API (the draft-id path), then through the id route.
	postJournal(t, db, u, "edited "+secretA, "draft-enc")
	assertEncrypted("update via draft id", "edited "+secretA)

	app := newAuthedTestApp(t, db, u, &JournalController{})
	code, body := send(t, app, http.MethodPut, "/api/journal/"+id, journalForm("again "+secretA, ""))
	if !is2xx(code) {
		t.Fatalf("PUT: %d %s", code, body)
	}
	assertEncrypted("update via PUT", "again "+secretA)
	if !strings.Contains(body, "again "+secretA) {
		t.Errorf("API response should carry the plaintext for the owner: %s", body)
	}
}

// Every other model that claims encryption at rest stores ciphertext in the
// raw column and returns plaintext through the model.
func TestModelsEncryptedAtRest(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	const plain = "very private words " + secretA

	cases := []struct {
		name, table string
		cols        []string
		create      func(t *testing.T, tx *gorm.DB, u *models.User) string
		load        func(tx *gorm.DB, id string) ([]string, error)
		update      func(tx *gorm.DB, id string) error
	}{
		{"actionItem", "action_items", []string{"text"},
			func(t *testing.T, tx *gorm.DB, u *models.User) string {
				a := &models.ActionItem{Base: baseFor(u), Text: plain}
				if err := tx.Create(a).Error; err != nil {
					t.Fatal(err)
				}
				return a.ID.String()
			},
			func(tx *gorm.DB, id string) ([]string, error) {
				var m models.ActionItem
				err := tx.First(&m, "id = ?", id).Error
				return []string{m.Text}, err
			},
			func(tx *gorm.DB, id string) error {
				var m models.ActionItem
				if err := tx.First(&m, "id = ?", id).Error; err != nil {
					return err
				}
				m.Text = plain
				return tx.Save(&m).Error
			}},
		{"thankful", "thankfuls", []string{"text"},
			func(t *testing.T, tx *gorm.DB, u *models.User) string {
				m := &models.Thankful{Base: baseFor(u), Text: plain}
				if err := tx.Create(m).Error; err != nil {
					t.Fatal(err)
				}
				return m.ID.String()
			},
			func(tx *gorm.DB, id string) ([]string, error) {
				var m models.Thankful
				err := tx.First(&m, "id = ?", id).Error
				return []string{m.Text}, err
			},
			func(tx *gorm.DB, id string) error {
				var m models.Thankful
				if err := tx.First(&m, "id = ?", id).Error; err != nil {
					return err
				}
				m.Text = plain
				return tx.Save(&m).Error
			}},
		{"reminder", "reminders", []string{"text"},
			func(t *testing.T, tx *gorm.DB, u *models.User) string {
				m := &models.Reminder{Base: baseFor(u), Text: plain}
				if err := tx.Create(m).Error; err != nil {
					t.Fatal(err)
				}
				return m.ID.String()
			},
			func(tx *gorm.DB, id string) ([]string, error) {
				var m models.Reminder
				err := tx.First(&m, "id = ?", id).Error
				return []string{m.Text}, err
			},
			func(tx *gorm.DB, id string) error {
				var m models.Reminder
				if err := tx.First(&m, "id = ?", id).Error; err != nil {
					return err
				}
				m.Text = plain
				return tx.Save(&m).Error
			}},
		{"recurringActionItem", "recurring_action_items", []string{"text"},
			func(t *testing.T, tx *gorm.DB, u *models.User) string {
				start := time.Now().UTC()
				m := &models.RecurringActionItem{Base: baseFor(u), Text: plain, Frequency: time.Hour, StartsAt: &start}
				if err := tx.Create(m).Error; err != nil {
					t.Fatal(err)
				}
				return m.ID.String()
			},
			func(tx *gorm.DB, id string) ([]string, error) {
				var m models.RecurringActionItem
				err := tx.First(&m, "id = ?", id).Error
				return []string{m.Text}, err
			},
			func(tx *gorm.DB, id string) error {
				var m models.RecurringActionItem
				if err := tx.First(&m, "id = ?", id).Error; err != nil {
					return err
				}
				m.Text = plain
				return tx.Save(&m).Error
			}},
		{"entryTemplate", "entry_templates", []string{"name", "body"},
			func(t *testing.T, tx *gorm.DB, u *models.User) string {
				m := &models.EntryTemplate{Base: baseFor(u), Name: plain, Body: plain}
				if err := tx.Create(m).Error; err != nil {
					t.Fatal(err)
				}
				return m.ID.String()
			},
			func(tx *gorm.DB, id string) ([]string, error) {
				var m models.EntryTemplate
				err := tx.First(&m, "id = ?", id).Error
				return []string{m.Name, m.Body}, err
			},
			func(tx *gorm.DB, id string) error {
				var m models.EntryTemplate
				if err := tx.First(&m, "id = ?", id).Error; err != nil {
					return err
				}
				m.Name, m.Body = plain, plain
				return tx.Save(&m).Error
			}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				u := compedUser(t, tx, "a@example.com")
				id := tc.create(t, tx, u)

				check := func(step string) {
					t.Helper()
					for _, col := range tc.cols {
						raw := rawCol(t, tx, tc.table, col, id)
						if raw == "" || strings.Contains(raw, "private") || strings.Contains(raw, secretA) {
							t.Errorf("%s: %s.%s is not ciphertext: %q", step, tc.table, col, raw)
						}
					}
					if rawCol(t, tx, tc.table, "is_encrypted", id) != "true" {
						t.Errorf("%s: is_encrypted not set", step)
					}
					got, err := tc.load(tx, id)
					if err != nil {
						t.Fatal(err)
					}
					for _, g := range got {
						if g != plain {
							t.Errorf("%s: loaded %q, want plaintext", step, g)
						}
					}
				}
				check("create")
				first := rawCol(t, tx, tc.table, tc.cols[0], id)
				if err := tc.update(tx, id); err != nil {
					t.Fatal(err)
				}
				check("update")
				if rawCol(t, tx, tc.table, tc.cols[0], id) == first {
					t.Error("re-encrypting identical text produced identical ciphertext (nonce reuse?)")
				}
			})
		})
	}
}
