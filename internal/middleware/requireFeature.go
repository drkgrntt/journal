package middleware

import (
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// RequireFeature blocks the request unless the current user owns and has
// enabled the feature with the given code. Must run after RequireAuth.
func RequireFeature(code string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
		if user == nil || !user.HasFeature(code) {
			return ctx.Status(http.StatusForbidden).SendString("This feature isn't enabled on your account")
		}
		return ctx.Next()
	}
}

// RequireSubscription blocks the request unless the current user's paid
// features are on (trial, active plan, lifetime, comped, or within the grace
// period). Must run after RequireAuth.
func RequireSubscription(ctx *fiber.Ctx) error {
	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if user == nil || !user.IsSubscribed() {
		return ctx.Status(http.StatusForbidden).SendString("This is part of the plan, and your plan isn't active right now")
	}
	return ctx.Next()
}
