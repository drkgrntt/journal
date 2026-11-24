package controllers

import (
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func sessionCount(t *testing.T, tx *gorm.DB, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	tx.Model(&models.Session{}).Where("user_id = ?", userID).Count(&n)
	return n
}

func TestSessionIsFoundByItsTokenAndStoredOnlyAsAHash(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s1@example.com")
		now := time.Now().UTC()
		token, err := models.CreateSession(tx, u.ID, now)
		if err != nil {
			t.Fatal(err)
		}
		got, err := models.FindSession(tx, token, now)
		if err != nil || got.UserID != u.ID {
			t.Fatalf("FindSession = %+v, %v", got, err)
		}
		var raw int64
		tx.Model(&models.Session{}).Where("token_hash = ? OR previous_token_hash = ?", token, token).Count(&raw)
		if raw != 0 {
			t.Error("the raw token is stored in the table")
		}
		for name, bad := range map[string]string{"empty": "", "unknown": "not-a-token"} {
			if _, err := models.FindSession(tx, bad, now); err == nil {
				t.Errorf("%s token found a session", name)
			}
		}
	})
}

func TestSessionExpires(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s2@example.com")
		now := time.Now().UTC()
		token, _ := models.CreateSession(tx, u.ID, now)
		if _, err := models.FindSession(tx, token, now.Add(models.SessionLifetime-time.Minute)); err != nil {
			t.Errorf("session expired early: %v", err)
		}
		if _, err := models.FindSession(tx, token, now.Add(models.SessionLifetime+time.Minute)); err == nil {
			t.Error("expired session still found")
		}
	})
}

func TestRotationKeepsThePreviousTokenOnlyBriefly(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s3@example.com")
		start := time.Now().UTC()
		old, _ := models.CreateSession(tx, u.ID, start)
		s, _ := models.FindSession(tx, old, start)

		rotatedAt := start.Add(models.SessionRotateAfter)
		fresh, rotated, err := models.RotateSession(tx, s, rotatedAt)
		if err != nil || !rotated || fresh == "" || fresh == old {
			t.Fatalf("RotateSession = %q, %v, %v", fresh, rotated, err)
		}
		if _, err := models.FindSession(tx, fresh, rotatedAt); err != nil {
			t.Errorf("new token not accepted: %v", err)
		}
		if _, err := models.FindSession(tx, old, rotatedAt.Add(models.SessionOverlap-time.Second)); err != nil {
			t.Errorf("previous token rejected during the overlap: %v", err)
		}
		if _, err := models.FindSession(tx, old, rotatedAt.Add(models.SessionOverlap+time.Second)); err == nil {
			t.Error("previous token still accepted after the overlap")
		}
		// Rotation extends the session.
		if _, err := models.FindSession(tx, fresh, rotatedAt.Add(models.SessionLifetime-time.Minute)); err != nil {
			t.Errorf("rotation didn't extend the session: %v", err)
		}
	})
}

func TestOnlyOneOfTwoSimultaneousRotationsWins(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s4@example.com")
		start := time.Now().UTC()
		token, _ := models.CreateSession(tx, u.ID, start)
		// Two requests both loaded the session before either rotated it.
		a, _ := models.FindSession(tx, token, start)
		b, _ := models.FindSession(tx, token, start)
		later := start.Add(models.SessionRotateAfter)
		if _, ok, _ := models.RotateSession(tx, a, later); !ok {
			t.Fatal("first rotation failed")
		}
		if tok, ok, err := models.RotateSession(tx, b, later); ok || tok != "" || err != nil {
			t.Errorf("second rotation = %q, %v, %v; want it skipped quietly", tok, ok, err)
		}
	})
}

func TestDeletingSessions(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s5@example.com")
		other := testutil.User(t, tx, "s5b@example.com")
		now := time.Now().UTC()
		t1, _ := models.CreateSession(tx, u.ID, now)
		t2, _ := models.CreateSession(tx, u.ID, now)
		models.CreateSession(tx, other.ID, now)

		if err := models.DeleteSessionByToken(tx, t1); err != nil {
			t.Fatal(err)
		}
		if _, err := models.FindSession(tx, t1, now); err == nil {
			t.Error("logged-out session still found")
		}
		if _, err := models.FindSession(tx, t2, now); err != nil {
			t.Error("logging one session out ended another")
		}

		keep, _ := models.FindSession(tx, t2, now)
		t3, _ := models.CreateSession(tx, u.ID, now)
		if err := models.DeleteUserSessions(tx, u.ID, &keep.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := models.FindSession(tx, t3, now); err == nil {
			t.Error("other session survived DeleteUserSessions")
		}
		if _, err := models.FindSession(tx, t2, now); err != nil {
			t.Error("the kept session was deleted")
		}
		if sessionCount(t, tx, other.ID) != 1 {
			t.Error("another user's session was deleted")
		}

		if err := models.DeleteUserSessions(tx, u.ID, nil); err != nil {
			t.Fatal(err)
		}
		if sessionCount(t, tx, u.ID) != 0 {
			t.Error("sessions left after signing out everywhere")
		}
	})
}

func TestDeleteExpiredSessions(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "s6@example.com")
		now := time.Now().UTC()
		models.CreateSession(tx, u.ID, now.Add(-models.SessionLifetime-time.Hour)) // expired
		live, _ := models.CreateSession(tx, u.ID, now)
		n, err := models.DeleteExpiredSessions(tx, now)
		if err != nil || n != 1 {
			t.Fatalf("deleted %d, %v; want 1", n, err)
		}
		if _, err := models.FindSession(tx, live, now); err != nil {
			t.Error("a live session was pruned")
		}
	})
}

// --- DeserializeToken, the real middleware ---

func cookieRequest(t *testing.T, app *fiber.App, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "x-token", Value: token})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestMiddlewareDoesNotRotateAFreshSession(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "m1@example.com")
		token, _ := models.CreateSession(tx, u.ID, time.Now().UTC())
		resp := cookieRequest(t, sessionApp(t, tx), token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		if cookieNamed(resp, "x-token") != nil {
			t.Error("a fresh session got a new cookie")
		}
	})
}

func TestMiddlewareRotatesAnOldSessionAndHonorsTheOverlap(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "m2@example.com")
		old, _ := models.CreateSession(tx, u.ID, time.Now().UTC().Add(-2*models.SessionRotateAfter))
		app := sessionApp(t, tx)

		resp := cookieRequest(t, app, old)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		c := cookieNamed(resp, "x-token")
		if c == nil || c.Value == "" || c.Value == old || c.Path != "/" || !c.HttpOnly {
			t.Fatalf("expected a fresh cookie, got %+v", c)
		}
		if got := cookieRequest(t, app, c.Value).StatusCode; got != http.StatusOK {
			t.Errorf("new token = %d, want 200", got)
		}
		if got := cookieRequest(t, app, old).StatusCode; got != http.StatusOK {
			t.Errorf("old token inside the overlap = %d, want 200", got)
		}
		// The old token riding out its overlap must not rotate again.
		if cookieNamed(cookieRequest(t, app, old), "x-token") != nil {
			t.Error("the previous token triggered another rotation")
		}

		tx.Model(&models.Session{}).Where("user_id = ?", u.ID).Update("previous_valid_until", time.Now().Add(-time.Minute))
		if got := cookieRequest(t, app, old).StatusCode; got != http.StatusUnauthorized {
			t.Errorf("old token after the overlap = %d, want 401", got)
		}
	})
}

func TestMiddlewareRejectsUnknownAndExpiredCookies(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "m3@example.com")
		expired, _ := models.CreateSession(tx, u.ID, time.Now().UTC().Add(-models.SessionLifetime-time.Hour))
		app := sessionApp(t, tx)
		for name, tok := range map[string]string{"unknown": "nope", "expired": expired, "a JWT": "aaa.bbb.ccc"} {
			if got := cookieRequest(t, app, tok).StatusCode; got != http.StatusUnauthorized {
				t.Errorf("%s cookie = %d, want 401", name, got)
			}
		}
	})
}
