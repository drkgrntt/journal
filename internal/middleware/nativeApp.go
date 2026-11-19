package middleware

import (
	"strings"

	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// NativeAppUserAgent is the token the native shell (the Capacitor project in
// mobile/) appends to its webview's user agent via `appendUserAgent`.
const NativeAppUserAgent = "ContourApp/"

// DetectNativeApp records whether the request came from the native shell, as a
// *bool under LocalIsNativeApp. It reads only the user agent and stores nothing,
// so it adds nothing to the request log.
//
// This is a presentation switch, not a security boundary: a user agent is
// client-controlled. It decides which links and billing controls to show, never
// what someone is allowed to do.
func DetectNativeApp(ctx *fiber.Ctx) error {
	isNative := strings.Contains(string(ctx.Context().UserAgent()), NativeAppUserAgent)
	ctx.Locals(utils.LocalIsNativeApp, &isNative)
	return ctx.Next()
}

// NativeAppHome sends the native shell straight to the app instead of the
// marketing landing page; everyone else falls through to the page.
func NativeAppHome(ctx *fiber.Ctx) error {
	if native := utils.GetLocal[bool](ctx, utils.LocalIsNativeApp); native != nil && *native {
		return ctx.Redirect("/dashboard")
	}
	return ctx.Next()
}
