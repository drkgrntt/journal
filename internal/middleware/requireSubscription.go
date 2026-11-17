package middleware

import (
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

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
