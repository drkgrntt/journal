package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func nativeAppTestApp() *fiber.App {
	app := fiber.New()
	app.Use(DetectNativeApp)
	app.Get("/", NativeAppHome, func(c *fiber.Ctx) error { return c.SendString("landing") })
	return app
}

func TestNativeAppHome(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		wantCode int
	}{
		{"native shell goes to the app", "Mozilla/5.0 (Linux; Android 14) ContourApp/1.0", http.StatusFound},
		{"browser sees the landing page", "Mozilla/5.0 (X11; Linux x86_64) Firefox/130.0", http.StatusOK},
		{"no user agent sees the landing page", "", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("User-Agent", tt.ua)
			res, err := nativeAppTestApp().Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tt.wantCode {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.wantCode)
			}
			if tt.wantCode == http.StatusFound {
				if loc := res.Header.Get("Location"); loc != "/dashboard" {
					t.Fatalf("Location = %q, want /dashboard", loc)
				}
			}
		})
	}
}
