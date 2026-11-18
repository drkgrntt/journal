package controllers

import (
	"io"
	"journal/internal/models"
	"journal/internal/testutil"
	"journal/internal/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// testCryptoKey is a 32-byte AES-256 key for tests that touch encrypted models.
const testCryptoKey = "0123456789abcdef0123456789abcdef"

// newAuthedTestApp builds a Fiber app where every request is already
// authenticated as user (what middleware.DeserializeToken does in production),
// with the given controllers' API routes registered against db.
func newAuthedTestApp(t testing.TB, db *gorm.DB, user *models.User, cs ...Controller) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(utils.LocalCurrentUser, user)
		return c.Next()
	})
	for _, c := range cs {
		c.Init(db, app)
		c.RegisterApiRoutes()
	}
	return app
}

// compedUser inserts a user whose paid features are on.
func compedUser(t testing.TB, tx *gorm.DB, email string) *models.User {
	t.Helper()
	return testutil.User(t, tx, email, func(u *models.User) { u.SubscriptionStatus = models.SubscriptionComped })
}

// send performs a form-encoded request and returns the status and body.
func send(t testing.TB, app *fiber.App, method, path string, form url.Values) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func is2xx(code int) bool { return code >= http.StatusOK && code < http.StatusMultipleChoices }

// rawRow returns the row exactly as stored (bypassing model hooks and the
// soft-delete scope) as JSON text, or "" when there is none.
func rawRow(t testing.TB, tx *gorm.DB, table, id string) string {
	t.Helper()
	var rows []string
	if err := tx.Raw("SELECT to_jsonb(t)::text FROM "+table+" t WHERE id = ?", id).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return ""
	}
	return rows[0]
}

// rawCol returns one column as stored, bypassing model hooks.
func rawCol(t testing.TB, tx *gorm.DB, table, col, id string) string {
	t.Helper()
	var vals []string
	if err := tx.Raw("SELECT "+col+" FROM "+table+" WHERE id = ?", id).Scan(&vals).Error; err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 {
		t.Fatalf("%s.%s for %s: got %d rows", table, col, id, len(vals))
	}
	return vals[0]
}

func rawCount(t testing.TB, tx *gorm.DB, table, where string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := tx.Raw("SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

// committedUsers returns the shared test database (not a rolled-back
// transaction) and a constructor for comped users in it. createJournal opens
// its own transaction with db.Begin(), which GORM refuses on a testutil.Tx
// transaction, so tests that POST a journal must commit. Everything those
// users created is hard-deleted when the test ends.
func committedUsers(t *testing.T) (*gorm.DB, func() *models.User) {
	t.Helper()
	db := testutil.DB(t)
	var ids []string
	t.Cleanup(func() {
		if len(ids) == 0 {
			return
		}
		for _, table := range []string{"action_items", "thankfuls", "journals", "reminders", "recurring_action_items", "entry_templates", "custom_journal_types"} {
			if err := db.Exec("DELETE FROM "+table+" WHERE creator_id IN ?", ids).Error; err != nil {
				t.Errorf("cleanup %s: %v", table, err)
			}
		}
		if err := db.Exec("DELETE FROM users WHERE id IN ?", ids).Error; err != nil {
			t.Errorf("cleanup users: %v", err)
		}
	})
	return db, func() *models.User {
		u := compedUser(t, db, uuid.NewString()+"@example.com")
		ids = append(ids, u.ID.String())
		return u
	}
}
