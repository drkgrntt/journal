package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func rateLimitApp(max int) *fiber.App {
	app := fiber.New()
	app.Post("/try", RateLimit(max, time.Minute), func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func tryFrom(t *testing.T, app *fiber.App, realIP string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/try", nil)
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode
}

func TestRateLimitBlocksAfterMax(t *testing.T) {
	app := rateLimitApp(3)
	for i := 0; i < 3; i++ {
		if got := tryFrom(t, app, "1.1.1.1"); got != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i+1, got)
		}
	}
	if got := tryFrom(t, app, "1.1.1.1"); got != http.StatusTooManyRequests {
		t.Fatalf("request 4 = %d, want 429", got)
	}
}

func TestRateLimitIsPerClient(t *testing.T) {
	app := rateLimitApp(1)
	tryFrom(t, app, "1.1.1.1")
	if got := tryFrom(t, app, "1.1.1.1"); got != http.StatusTooManyRequests {
		t.Fatalf("same client = %d, want 429", got)
	}
	if got := tryFrom(t, app, "2.2.2.2"); got != http.StatusOK {
		t.Fatalf("other client = %d, want 200", got)
	}
}

func TestClientKey(t *testing.T) {
	tests := []struct {
		name, realIP, xff, want string
	}{
		{"real ip wins", "9.9.9.9", "1.1.1.1, 2.2.2.2", "9.9.9.9"},
		{"last forwarded entry, not a spoofed first one", "", "6.6.6.6, 2.2.2.2", "2.2.2.2"},
		{"single forwarded entry", "", "3.3.3.3", "3.3.3.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var got string
			app.Get("/", func(c *fiber.Ctx) error { got = clientKey(c); return nil })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if _, err := app.Test(req); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("clientKey = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	for _, hsts := range []bool{false, true} {
		app := fiber.New()
		app.Use(SecurityHeaders(hsts))
		app.Get("/", func(c *fiber.Ctx) error { return c.SendString("x") })
		res, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		if res.Header.Get("X-Frame-Options") != "DENY" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("missing framing/sniffing headers: %v", res.Header)
		}
		csp := res.Header.Get("Content-Security-Policy")
		for _, want := range []string{"frame-ancestors 'none'", "object-src 'none'", "script-src 'self'"} {
			if !strings.Contains(csp, want) {
				t.Errorf("CSP %q is missing %q", csp, want)
			}
		}
		if strings.Contains(csp, "script-src *") || strings.Contains(csp, "http:") {
			t.Errorf("CSP allows scripts from other origins: %q", csp)
		}
		if got := res.Header.Get("Strict-Transport-Security") != ""; got != hsts {
			t.Errorf("HSTS present = %v, want %v", got, hsts)
		}
	}
}

func TestWindowLimiterResetsAfterWindow(t *testing.T) {
	l := &windowLimiter{max: 1, window: time.Minute, hits: map[string]*hitWindow{}}
	now := time.Now()
	if !l.allow("a", now) {
		t.Fatal("first request blocked")
	}
	if l.allow("a", now.Add(30*time.Second)) {
		t.Fatal("second request inside the window allowed")
	}
	if !l.allow("a", now.Add(61*time.Second)) {
		t.Fatal("request after the window blocked")
	}
}
