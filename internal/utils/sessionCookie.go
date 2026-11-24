package utils

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SessionCookieName is the cookie holding the random session token.
const SessionCookieName = "x-token"

// SessionCookieLifetime matches models.SessionLifetime (utils can't import
// models). Each rotation re-sends the cookie, so it slides with use.
const SessionCookieLifetime = 30 * 24 * time.Hour

func sessionCookie(value string, expires time.Time, maxAge int) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
	}
}

// SetSessionCookie sends the session token to the browser.
func SetSessionCookie(ctx *fiber.Ctx, token string) {
	ctx.Cookie(sessionCookie(token, time.Now().Add(SessionCookieLifetime), int(SessionCookieLifetime.Seconds())))
}

// ClearSessionCookie must repeat the Path the cookie was set with, or the
// browser treats it as a different cookie and the session survives.
func ClearSessionCookie(ctx *fiber.Ctx) {
	ctx.Cookie(sessionCookie("", time.Unix(0, 0), -1))
}
