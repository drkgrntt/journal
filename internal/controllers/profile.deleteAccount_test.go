package controllers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"journal/internal/models"
	"journal/internal/testutil"

	"gorm.io/gorm"
)

const deletePath = "/api/profile/delete-account"

// everything inserts one of each thing an account can own, plus a soft-deleted
// entry, a queued email, and a session.
func everything(t *testing.T, tx *gorm.DB, u *models.User) {
	t.Helper()
	for _, c := range authzCases() {
		if c.method == http.MethodDelete {
			c.create(t, tx, u)
		}
	}
	old := journalFor(t, tx, u)
	if err := tx.Exec("UPDATE journals SET deleted_at = now() WHERE id = ?", old).Error; err != nil {
		t.Fatal(err)
	}
	job := &models.Job{Base: baseFor(u), Type: "email"}
	if err := tx.Create(job).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := models.CreateSession(tx, u.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// ownedRows counts rows belonging to u in every table that has an owner column,
// read from the schema so a table added later is checked too.
func ownedRows(t *testing.T, tx *gorm.DB, u *models.User) map[string]int64 {
	t.Helper()
	var cols []struct{ TableName, ColumnName string }
	err := tx.Raw(`SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND column_name IN ('creator_id', 'user_id')`).Scan(&cols).Error
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]int64{}
	for _, c := range cols {
		if n := rawCount(t, tx, c.TableName, c.ColumnName+" = ?", u.ID); n > 0 {
			found[c.TableName] += n
		}
	}
	if n := rawCount(t, tx, "users", "id = ?", u.ID); n > 0 {
		found["users"] = n
	}
	return found
}

func deleteApp(t *testing.T, tx *gorm.DB, u *models.User, cancel func(*models.User) error) func(password string) (int, string) {
	app := newAuthedTestApp(t, tx, u, &ProfileController{cancelBilling: cancel})
	return func(password string) (int, string) {
		return send(t, app, http.MethodPost, deletePath, form("password", password))
	}
}

func noBilling(*models.User) error { return nil }

func TestDeleteAccountRemovesEverythingOfTheirs(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	testutil.Tx(t, func(tx *gorm.DB) {
		leaving := compedUser(t, tx, "leaving@example.com")
		staying := compedUser(t, tx, "staying@example.com")
		everything(t, tx, leaving)
		// custom topic names are unique across users
		tx.Exec("UPDATE custom_journal_types SET name = 'leaving' WHERE creator_id = ?", leaving.ID)
		everything(t, tx, staying)
		before := ownedRows(t, tx, staying)
		if len(before) < 10 {
			t.Fatalf("fixture is thin: %v", before)
		}

		code, body := deleteApp(t, tx, leaving, noBilling)("password")
		if !is2xx(code) {
			t.Fatalf("status %d: %s", code, body)
		}
		if left := ownedRows(t, tx, leaving); len(left) != 0 {
			t.Errorf("rows left behind: %v", left)
		}
		after := ownedRows(t, tx, staying)
		for table, n := range before {
			if after[table] != n {
				t.Errorf("another account's %s went from %d to %d", table, n, after[table])
			}
		}
	})
}

func TestDeleteAccountRefusals(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	cases := []struct {
		name, password string
		billing        error
		want           int
		billingCalled  bool
	}{
		{"wrong password", "not-it", nil, http.StatusBadRequest, false},
		{"no password", "", nil, http.StatusBadRequest, false},
		{"subscription can't be canceled", "password", errors.New("stripe down"), http.StatusInternalServerError, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				user := compedUser(t, tx, "refused@example.com")
				everything(t, tx, user)
				before := ownedRows(t, tx, user)
				called := false

				code, _ := deleteApp(t, tx, user, func(*models.User) error {
					called = true
					return c.billing
				})(c.password)

				if code != c.want {
					t.Errorf("status %d, want %d", code, c.want)
				}
				if called != c.billingCalled {
					t.Errorf("billing called = %v, want %v", called, c.billingCalled)
				}
				after := ownedRows(t, tx, user)
				for table, n := range before {
					if after[table] != n {
						t.Errorf("%s went from %d to %d", table, n, after[table])
					}
				}
			})
		})
	}
}

func TestDeleteAccountNeedsLogin(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		code, _ := send(t, newTestApp(t, tx, &ProfileController{}), http.MethodPost, deletePath, form("password", "password"))
		if code != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", code)
		}
	})
}
