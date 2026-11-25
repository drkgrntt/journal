package controllers

import (
	"journal/internal/models"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func thankfulForm(text, clientID string) url.Values {
	v := form("text", text)
	if clientID != "" {
		v.Set("clientId", clientID)
	}
	return v
}

func thankfulCall(t *testing.T, db *gorm.DB, u *models.User, method, path string, v url.Values) (int, string, string) {
	t.Helper()
	return actionItemRequest(t, newAuthedTestApp(t, db, u, &ThankfulController{}), method, path, v)
}

func thankfulLoad(t *testing.T, db *gorm.DB, u *models.User) []models.Thankful {
	t.Helper()
	var items []models.Thankful
	if err := db.Where("creator_id = ?", u.ID).Order("created_at").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	return items
}

func TestThankfulClientIDDedup(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	const path = "/api/thankfuls"

	t.Run("same user and client id is one row, latest text wins", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("first", "c-1"))
		thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("second", "c-1"))
		items := thankfulLoad(t, db, u)
		if len(items) != 1 || items[0].Text != "second" {
			t.Fatalf("items = %+v", items)
		}
		_, meta := models.CastMetadata[models.ThankfulMetadata](items[0].Metadata)
		if meta.ClientID != "c-1" {
			t.Errorf("client id = %q", meta.ClientID)
		}
	})

	t.Run("different client ids make two rows", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("a", "c-1"))
		thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("b", "c-2"))
		if n := len(thankfulLoad(t, db, u)); n != 2 {
			t.Errorf("rows = %d", n)
		}
	})

	t.Run("two users with the same client id never collide", func(t *testing.T) {
		db, newUser := committedUsers(t)
		a, b := newUser(), newUser()
		thankfulCall(t, db, a, http.MethodPost, path, thankfulForm("A's", "shared"))
		thankfulCall(t, db, b, http.MethodPost, path, thankfulForm("B's", "shared"))
		for _, u := range []*models.User{a, b} {
			if n := len(thankfulLoad(t, db, u)); n != 1 {
				t.Fatalf("user %s rows = %d", u.ID, n)
			}
		}
		if got := thankfulLoad(t, db, a)[0].Text; got != "A's" {
			t.Errorf("A's row overwritten: %q", got)
		}
	})

	t.Run("client id over 64 chars is a 400", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		code, _, _ := thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("a", strings.Repeat("x", 65)))
		if code != http.StatusBadRequest {
			t.Errorf("code = %d", code)
		}
		if n := len(thankfulLoad(t, db, u)); n != 0 {
			t.Errorf("rows = %d", n)
		}
	})

	t.Run("X-Item-Id matches the row on create and update", func(t *testing.T) {
		db, newUser := committedUsers(t)
		u := newUser()
		code, _, id := thankfulCall(t, db, u, http.MethodPost, path, thankfulForm("a", ""))
		if !is2xx(code) {
			t.Fatal(code)
		}
		items := thankfulLoad(t, db, u)
		if id == "" || id != items[0].ID.String() {
			t.Fatalf("create header %q, row %s", id, items[0].ID)
		}
		_, _, id = thankfulCall(t, db, u, http.MethodPut, path+"/"+items[0].ID.String(), thankfulForm("b", ""))
		if id != items[0].ID.String() {
			t.Errorf("update header %q", id)
		}
	})
}
