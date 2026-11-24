package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimit allows max requests per window per client, then answers 429. The
// counts live in memory only and are never written anywhere, so this keeps the
// request log anonymous (see LogAccess). A restart clears them. It is a plain
// fixed window; no dependency needed.
func RateLimit(max int, window time.Duration) fiber.Handler {
	l := &windowLimiter{max: max, window: window, hits: map[string]*hitWindow{}}
	return func(ctx *fiber.Ctx) error {
		if !l.allow(clientKey(ctx), time.Now()) {
			return ctx.Status(http.StatusTooManyRequests).SendString("Too many tries. Give it a few minutes and try again.")
		}
		return ctx.Next()
	}
}

type hitWindow struct {
	start time.Time
	count int
}

type windowLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*hitWindow
}

func (l *windowLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Drop expired windows so the map can't grow without bound.
	if len(l.hits) > 10000 {
		for k, h := range l.hits {
			if now.Sub(h.start) >= l.window {
				delete(l.hits, k)
			}
		}
	}

	h := l.hits[key]
	if h == nil || now.Sub(h.start) >= l.window {
		l.hits[key] = &hitWindow{start: now, count: 1}
		return true
	}
	h.count++
	return h.count <= l.max
}

// clientKey identifies a client for rate limiting. The app sits behind nginx,
// so the socket address is always the proxy's. Prefer X-Real-IP (set by the
// proxy to the real peer); otherwise the LAST X-Forwarded-For entry, which is
// the one the proxy appended (earlier entries are client-supplied).
func clientKey(ctx *fiber.Ctx) string {
	if ip := strings.TrimSpace(ctx.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := ctx.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
			return ip
		}
	}
	return ctx.IP()
}

// SecurityHeaders sets response headers that stop clickjacking, MIME sniffing
// and referrer leaks. HSTS is sent only in production (it needs HTTPS).
//
// script-src only allows scripts from this site, which stops an injected
// <script src="https://elsewhere"> from loading. It keeps 'unsafe-inline' and
// 'unsafe-eval' on purpose: pages use inline handlers (onclick, hx-on:*) and
// htmx evaluates hx-on, so dropping them means rewriting those handlers and
// checking every screen in a browser. No other origin hosts a script here.
func SecurityHeaders(hsts bool) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		ctx.Set("X-Content-Type-Options", "nosniff")
		ctx.Set("X-Frame-Options", "DENY")
		ctx.Set("Referrer-Policy", "same-origin")
		ctx.Set("Content-Security-Policy", "frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'; script-src 'self' 'unsafe-inline' 'unsafe-eval'")
		if hsts {
			ctx.Set("Strict-Transport-Security", "max-age=31536000")
		}
		return ctx.Next()
	}
}
