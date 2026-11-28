package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"journal/internal/models"
	"journal/internal/testutil"

	"gorm.io/gorm"
)

// "Everything you've written" includes thankfuls and action items that aren't
// attached to an entry, or whose entry has since been deleted, once each.
func TestExportIncludesUnattachedItems(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	testutil.Tx(t, func(tx *gorm.DB) {
		user := testutil.User(t, tx, "export@example.com")
		other := testutil.User(t, tx, "other-export@example.com")

		kept := journalFor(t, tx, user)
		goneID := journalFor(t, tx, user)
		var keptJournal, gone models.Journal
		tx.First(&keptJournal, "id = ?", kept)
		tx.First(&gone, "id = ?", goneID)

		items := []any{
			&models.Thankful{Base: baseFor(user), Text: "THANKS-ATTACHED", JournalID: &keptJournal.ID},
			&models.Thankful{Base: baseFor(user), Text: "THANKS-ALONE"},
			&models.Thankful{Base: baseFor(user), Text: "THANKS-ORPHAN", JournalID: &gone.ID},
			&models.ActionItem{Base: baseFor(user), Text: "ACTION-ALONE"},
			&models.ActionItem{Base: baseFor(user), Text: "ACTION-ORPHAN", JournalID: &gone.ID},
			&models.Thankful{Base: baseFor(other), Text: "SOMEONE-ELSE"},
			&models.ActionItem{Base: baseFor(other), Text: "SOMEONE-ELSE"},
		}
		for _, item := range items {
			if err := tx.Create(item).Error; err != nil {
				t.Fatal(err)
			}
		}
		if err := tx.Unscoped().Delete(&gone).Error; err != nil {
			t.Fatal(err)
		}

		app := newAuthedTestApp(t, tx, user, &ProfileController{})
		for _, format := range []string{"markdown", "json"} {
			code, body := send(t, app, http.MethodGet, "/api/profile/export?format="+format, nil)
			if code != http.StatusOK {
				t.Fatalf("%s: status %d: %s", format, code, body)
			}
			for _, want := range []string{"THANKS-ATTACHED", "THANKS-ALONE", "THANKS-ORPHAN", "ACTION-ALONE", "ACTION-ORPHAN"} {
				if n := strings.Count(body, want); n != 1 {
					t.Errorf("%s: %s appears %d times, want 1", format, want, n)
				}
			}
			if strings.Contains(body, "SOMEONE-ELSE") {
				t.Errorf("%s: another user's writing was exported", format)
			}
			if format == "json" {
				var out []exportedJournal
				if err := json.Unmarshal([]byte(body), &out); err != nil {
					t.Fatal(err)
				}
				if len(out) != 2 || out[1].Topic != unattachedTopic {
					t.Errorf("want one entry then the unattached group, got %+v", out)
				}
			}
		}
	})
}
