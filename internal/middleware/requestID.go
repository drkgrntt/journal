package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"journal/internal/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// RequestID gives every request a short random reference. It is not derived
// from anything about the person, and it is not shared across requests, so it
// can't link anyone's activity together. It is sent back as X-Request-ID,
// stored in the request log, and added to plain-text 5xx responses so a person
// who reports a problem can quote it and the matching request can be found.
// Any ID the client sends is ignored.
func RequestID(ctx *fiber.Ctx) error {
	id := newRequestID()
	ctx.Locals(utils.LocalRequestID, &id)
	ctx.Set("X-Request-ID", id)

	err := ctx.Next()

	if ctx.Response().StatusCode() >= 500 {
		res := ctx.Response()
		if strings.HasPrefix(string(res.Header.ContentType()), "text/plain") {
			res.SetBodyString(fmt.Sprintf("%s (reference: %s)", strings.TrimSpace(string(res.Body())), id))
		}
	}
	return err
}

func newRequestID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
