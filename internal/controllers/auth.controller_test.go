package controllers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"journal/internal/jobs"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/testutil"
	"journal/internal/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// setAuthEnv gives the test its own signing keys and token settings.
func setAuthEnv(t *testing.T) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	t.Setenv("ACCESS_TOKEN_PRIVATE_KEY", base64.StdEncoding.EncodeToString(privPEM))
	t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pubPEM))
	t.Setenv("CRYPTO_KEY", testCryptoKey)
}

func authApp(t *testing.T, tx *gorm.DB) *fiber.App {
	t.Helper()
	// Forgot-password queues an email job through the jobs package's own handle.
	jobs.SetDB(tx)
	t.Cleanup(func() { jobs.SetDB(nil) })
	return newTestApp(t, tx, &AuthController{})
}

func post(t *testing.T, app *fiber.App, method, path string, form url.Values) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func bodyOf(resp *http.Response) string {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func cookieNamed(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// sessionApp answers with the user the x-token cookie resolves to, using the
// real DeserializeToken middleware.
func sessionApp(t *testing.T, tx *gorm.DB) *fiber.App {
	t.Helper()
	middleware.SetDB(tx)
	t.Cleanup(func() { middleware.SetDB(nil) })
	app := fiber.New()
	app.Use(middleware.DeserializeToken)
	app.Get("/me", func(c *fiber.Ctx) error {
		if u := utils.GetLocal[models.User](c, utils.LocalCurrentUser); u != nil {
			return c.SendString(u.Email)
		}
		return c.SendStatus(http.StatusUnauthorized)
	})
	return app
}

func sessionStatus(t *testing.T, app *fiber.App, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "x-token", Value: token})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}

func TestLogoutEndsTheSessionAndClearsTheCookie(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "logout@example.com")
		mine, _ := models.CreateSession(tx, u.ID, time.Now().UTC())
		theirs, _ := models.CreateSession(tx, u.ID, time.Now().UTC())

		req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "x-token", Value: mine})
		resp, err := authApp(t, tx).Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		c := cookieNamed(resp, "x-token")
		if c == nil {
			t.Fatal("no x-token cookie in the logout response")
		}
		if c.Path != "/" {
			t.Errorf("Path = %q, want \"/\"", c.Path)
		}
		if c.Value != "" || c.Expires.After(time.Now()) || c.Expires.IsZero() {
			t.Errorf("cookie not cleared: value=%q expires=%v", c.Value, c.Expires)
		}
		sessions := sessionApp(t, tx)
		if got := sessionStatus(t, sessions, mine); got != http.StatusUnauthorized {
			t.Errorf("the logged-out token still works (%d): logout only cleared the cookie", got)
		}
		if got := sessionStatus(t, sessions, theirs); got != http.StatusOK {
			t.Errorf("logging out ended the user's other session (%d)", got)
		}
	})
}

func TestLoginSetsASessionThatWorks(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		testutil.User(t, tx, "login@example.com")
		resp := post(t, authApp(t, tx), http.MethodPost, "/api/auth/login",
			url.Values{"email": {"login@example.com"}, "password": {"password"}})
		c := cookieNamed(resp, "x-token")
		if c == nil || c.Path != "/" || !c.HttpOnly {
			t.Fatalf("bad session cookie: %+v", c)
		}
		if got := sessionStatus(t, sessionApp(t, tx), c.Value); got != http.StatusOK {
			t.Errorf("session status = %d, want 200", got)
		}
	})
}

func TestRegisterRejectsWeakPasswordAndBadEmail(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		app := authApp(t, tx)
		for name, form := range map[string]url.Values{
			"empty password": {"email": {"a@example.com"}, "password": {""}},
			"short password": {"email": {"a@example.com"}, "password": {"short"}},
			"no email":       {"email": {"nope"}, "password": {"longenough1"}},
		} {
			t.Run(name, func(t *testing.T) {
				resp := post(t, app, http.MethodPost, "/api/auth/register", form)
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("status = %d, want 400", resp.StatusCode)
				}
			})
		}
		var n int64
		tx.Model(&models.User{}).Where("email = ?", "a@example.com").Count(&n)
		if n != 0 {
			t.Error("a user was created despite the rejection")
		}
	})
}

func TestForgotDoesNotRevealWhetherTheEmailExists(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		testutil.User(t, tx, "known@example.com")
		app := authApp(t, tx)
		a := post(t, app, http.MethodPost, "/api/auth/forgot", url.Values{"email": {"known@example.com"}})
		b := post(t, app, http.MethodPost, "/api/auth/forgot", url.Values{"email": {"unknown@example.com"}})
		if a.StatusCode != b.StatusCode || bodyOf(a) != bodyOf(b) {
			t.Error("known and unknown addresses get different answers")
		}
		if b.StatusCode != http.StatusAccepted {
			t.Errorf("status = %d, want 202", b.StatusCode)
		}
	})
}

func resetForm(token, pw string) url.Values {
	return url.Values{"token": {token}, "password": {pw}, "confirmPassword": {pw}}
}

func TestResetLinkWorksOnceAndEndsOldSessions(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "reset@example.com")
		app := authApp(t, tx)
		sessions := sessionApp(t, tx)
		oldSession, _ := models.CreateSession(tx, u.ID, time.Now().UTC())
		link, _ := utils.CreateResetToken(u.ID, u.TokenVersion)

		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(link, "brand-new-pass")); resp.StatusCode != http.StatusOK {
			t.Fatalf("first use = %d, want 200", resp.StatusCode)
		}
		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(link, "another-pass-1")); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("second use = %d, want 400 (link must be single-use)", resp.StatusCode)
		}

		var got models.User
		tx.First(&got, "id = ?", u.ID)
		if got.ComparePasswords("brand-new-pass") != nil {
			t.Error("password was not changed by the first use")
		}
		if got.ComparePasswords("another-pass-1") == nil {
			t.Error("password was changed by the replayed link")
		}
		if code := sessionStatus(t, sessions, oldSession); code != http.StatusUnauthorized {
			t.Errorf("old session = %d, want 401 after a reset", code)
		}
	})
}

func TestResetRejectsSessionCookieAndWeakPassword(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "reset2@example.com")
		app := authApp(t, tx)
		session, _ := models.CreateSession(tx, u.ID, time.Now().UTC())
		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(session, "brand-new-pass")); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("session token as reset token = %d, want 400", resp.StatusCode)
		}

		link, _ := utils.CreateResetToken(u.ID, u.TokenVersion)
		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(link, "short")); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("weak password = %d, want 400", resp.StatusCode)
		}
		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(link, "")); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("empty password = %d, want 400", resp.StatusCode)
		}
		// The rejected tries must not have used up the link.
		if resp := post(t, app, http.MethodPut, "/api/auth/reset", resetForm(link, "brand-new-pass")); resp.StatusCode != http.StatusOK {
			t.Errorf("link after rejected tries = %d, want 200", resp.StatusCode)
		}
	})
}

func TestResetTokenCannotBeUsedAsALoginCookie(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "cookie@example.com")
		link, _ := utils.CreateResetToken(u.ID, u.TokenVersion)
		if code := sessionStatus(t, sessionApp(t, tx), link); code != http.StatusUnauthorized {
			t.Errorf("reset token as cookie = %d, want 401", code)
		}
	})
}

// sessionAuthedApp is newAuthedTestApp plus the session id DeserializeToken
// would set, for handlers that act on "this session".
func sessionAuthedApp(t *testing.T, tx *gorm.DB, u *models.User, sessionID uuid.UUID, cs ...Controller) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(utils.LocalCurrentUser, u)
		c.Locals(utils.LocalSessionID, &sessionID)
		return c.Next()
	})
	for _, c := range cs {
		c.Init(tx, app)
		c.RegisterApiRoutes()
	}
	return app
}

func TestPasswordChangeEndsOtherSessionsButKeepsThisOne(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "change@example.com")
		now := time.Now().UTC()
		this, _ := models.CreateSession(tx, u.ID, now)
		elsewhere, _ := models.CreateSession(tx, u.ID, now)
		thisSession, _ := models.FindSession(tx, this, now)

		app := sessionAuthedApp(t, tx, u, thisSession.ID, &ProfileController{})
		resp := post(t, app, http.MethodPut, "/api/profile/password", url.Values{
			"currentPassword": {"password"}, "newPassword": {"a-new-password"}, "confirmNewPassword": {"a-new-password"},
		})
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", resp.StatusCode)
		}
		sessions := sessionApp(t, tx)
		if code := sessionStatus(t, sessions, elsewhere); code != http.StatusUnauthorized {
			t.Errorf("other session = %d, want 401", code)
		}
		if code := sessionStatus(t, sessions, this); code != http.StatusOK {
			t.Errorf("this session = %d, want 200 (it should stay signed in)", code)
		}
	})
}

func TestSignOutEverywhereEndsEverySessionAndClearsTheCookie(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "everywhere@example.com")
		other := testutil.User(t, tx, "bystander@example.com")
		now := time.Now().UTC()
		this, _ := models.CreateSession(tx, u.ID, now)
		elsewhere, _ := models.CreateSession(tx, u.ID, now)
		bystander, _ := models.CreateSession(tx, other.ID, now)
		thisSession, _ := models.FindSession(tx, this, now)

		app := sessionAuthedApp(t, tx, u, thisSession.ID, &ProfileController{})
		resp := post(t, app, http.MethodPost, "/api/profile/sign-out-everywhere", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		if resp.Header.Get("HX-Redirect") != "/auth/login" {
			t.Errorf("HX-Redirect = %q", resp.Header.Get("HX-Redirect"))
		}
		if c := cookieNamed(resp, "x-token"); c == nil || c.Value != "" || c.Path != "/" {
			t.Errorf("cookie not cleared: %+v", c)
		}
		sessions := sessionApp(t, tx)
		for name, tok := range map[string]string{"this": this, "other": elsewhere} {
			if code := sessionStatus(t, sessions, tok); code != http.StatusUnauthorized {
				t.Errorf("%s session = %d, want 401", name, code)
			}
		}
		if code := sessionStatus(t, sessions, bystander); code != http.StatusOK {
			t.Errorf("another user's session = %d, want 200", code)
		}
	})
}

func TestPasswordChangeRejectsWeakPassword(t *testing.T) {
	setAuthEnv(t)
	testutil.Tx(t, func(tx *gorm.DB) {
		u := testutil.User(t, tx, "weak@example.com")
		app := newAuthedTestApp(t, tx, u, &ProfileController{})
		resp := post(t, app, http.MethodPut, "/api/profile/password", url.Values{
			"currentPassword": {"password"}, "newPassword": {"x"}, "confirmNewPassword": {"x"},
		})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
	})
}
