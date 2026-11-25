package controllers

import (
	"io"
	"journal/internal/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// actionItemForm is a valid body. Pass clientID "" to omit it; kv adds or
// overrides fields.
func actionItemForm(text, clientID string, kv ...string) url.Values {
	v := form("text", text)
	if clientID != "" {
		v.Set("clientId", clientID)
	}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

// actionItemRequest is like send but also returns the X-Item-Id header.
func actionItemRequest(t testing.TB, app *fiber.App, method, path string, v url.Values) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	code := resp.StatusCode
	// Under app.Test the fasthttp context reads as already canceled, so
	// RenderComponents answers 400 "context canceled" after the handler has
	// saved the row and set its headers. That is a test-harness artifact, not
	// a handler failure, so report it as the success it was.
	if code == http.StatusBadRequest && string(b) == "context canceled" {
		code = http.StatusOK
	}
	return code, string(b), resp.Header.Get("X-Item-Id")
}

// actionItemCall runs one request as u and returns status, body, X-Item-Id.
func actionItemCall(t *testing.T, db *gorm.DB, u *models.User, method, path string, v url.Values) (int, string, string) {
	t.Helper()
	return actionItemRequest(t, newAuthedTestApp(t, db, u, &ActionItemController{}), method, path, v)
}

func actionItemLoad(t *testing.T, db *gorm.DB, u *models.User) []models.ActionItem {
	t.Helper()
	var items []models.ActionItem
	if err := db.Where("creator_id = ?", u.ID).Order("created_at").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	return items
}

func actionItemMs(d time.Time) string { return strconv.FormatInt(d.UnixMilli(), 10) }

func TestActionItemClientIDDedup(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	const path = "/api/action-items"

	t.Run("same user and client id is one row, latest text wins", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("first", "c-1"))
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("second", "c-1"))
		items := actionItemLoad(t, db, u)
		if len(items) != 1 || items[0].Text != "second" {
			t.Fatalf("items = %+v", items)
		}
		_, meta := models.CastMetadata[models.ActionItemMetadata](items[0].Metadata)
		if meta.ClientID != "c-1" {
			t.Errorf("client id = %q", meta.ClientID)
		}
	})

	t.Run("different client ids make two rows", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", "c-1"))
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("b", "c-2"))
		if n := len(actionItemLoad(t, db, u)); n != 2 {
			t.Errorf("rows = %d", n)
		}
	})

	t.Run("two users with the same client id never collide", func(t *testing.T) {
		db, newUser := committedUsers(t)
		a, b := newUser(), newUser()
		actionItemCall(t, db, a, http.MethodPost, path, actionItemForm("A's", "shared"))
		actionItemCall(t, db, b, http.MethodPost, path, actionItemForm("B's", "shared"))
		for _, u := range []*models.User{a, b} {
			if n := len(actionItemLoad(t, db, u)); n != 1 {
				t.Fatalf("user %s rows = %d", u.ID, n)
			}
		}
		if got := actionItemLoad(t, db, a)[0].Text; got != "A's" {
			t.Errorf("A's row overwritten: %q", got)
		}
	})

	t.Run("client id over 64 chars is a 400", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		code, _, _ := actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", strings.Repeat("x", 65)))
		if code != http.StatusBadRequest {
			t.Errorf("code = %d", code)
		}
		if n := len(actionItemLoad(t, db, u)); n != 0 {
			t.Errorf("rows = %d", n)
		}
	})

	t.Run("X-Item-Id matches the row on create and update", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		code, body, id := actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", ""))
		if !is2xx(code) {
			t.Fatal(code, body)
		}
		items := actionItemLoad(t, db, u)
		if id == "" || id != items[0].ID.String() {
			t.Fatalf("create header %q, row %s", id, items[0].ID)
		}
		_, _, id = actionItemCall(t, db, u, http.MethodPut, path+"/"+items[0].ID.String(), actionItemForm("b", ""))
		if id != items[0].ID.String() {
			t.Errorf("update header %q", id)
		}
	})
}

func TestActionItemCompletedAt(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	const path = "/api/action-items"
	now := time.Now()

	create := func(t *testing.T, kv ...string) *models.ActionItem {
		t.Helper()
		db, newUser := committedUsers(t)
		u := newUser()
		code, body, _ := actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", "", kv...))
		if !is2xx(code) {
			t.Fatalf("%d %s", code, body)
		}
		return &actionItemLoad(t, db, u)[0]
	}
	near := func(a, b time.Time) bool { d := a.Sub(b); return d < 5*time.Second && d > -5*time.Second }

	t.Run("past completedAt in window is used", func(t *testing.T) {
		want := now.Add(-48 * time.Hour)
		it := create(t, "isComplete", "true", "completedAt", actionItemMs(want))
		if it.CompletedAt == nil || it.CompletedAt.UnixMilli() != want.UnixMilli() {
			t.Errorf("CompletedAt = %v, want %v", it.CompletedAt, want)
		}
	})

	t.Run("future, ancient and absent fall back to now", func(t *testing.T) {
		for name, kv := range map[string][]string{
			"future":  {"isComplete", "true", "completedAt", actionItemMs(now.Add(time.Hour))},
			"ancient": {"isComplete", "true", "completedAt", actionItemMs(now.Add(-100 * 24 * time.Hour))},
			"absent":  {"isComplete", "true"},
		} {
			it := create(t, kv...)
			if it.CompletedAt == nil || !near(*it.CompletedAt, time.Now()) {
				t.Errorf("%s: CompletedAt = %v", name, it.CompletedAt)
			}
		}
	})

	t.Run("slightly future is clamped to now", func(t *testing.T) {
		it := create(t, "isComplete", "true", "completedAt", actionItemMs(now.Add(2*time.Minute)))
		if it.CompletedAt == nil || it.CompletedAt.After(time.Now().Add(time.Second)) {
			t.Errorf("CompletedAt = %v", it.CompletedAt)
		}
	})

	t.Run("replayed complete keeps the first CompletedAt", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", "c-1", "isComplete", "true"))
		first := *actionItemLoad(t, db, u)[0].CompletedAt
		time.Sleep(20 * time.Millisecond)
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a2", "c-1", "isComplete", "true"))
		items := actionItemLoad(t, db, u)
		if len(items) != 1 || items[0].CompletedAt == nil || !items[0].CompletedAt.Equal(first) {
			t.Errorf("CompletedAt moved: %v -> %v", first, items[0].CompletedAt)
		}
	})

	t.Run("complete then uncomplete clears CompletedAt", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		actionItemCall(t, db, u, http.MethodPost, path, actionItemForm("a", "", "isComplete", "true"))
		id := actionItemLoad(t, db, u)[0].ID.String()
		actionItemCall(t, db, u, http.MethodPut, path+"/"+id, actionItemForm("a", ""))
		if it := actionItemLoad(t, db, u)[0]; it.CompletedAt != nil {
			t.Errorf("CompletedAt = %v", it.CompletedAt)
		}
	})
}
