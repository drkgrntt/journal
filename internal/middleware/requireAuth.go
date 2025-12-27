package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth(ctx *fiber.Ctx) error {
	user := utils.GetLocal[models.User](ctx, "currentUser")
	if user == nil {
		logger.Warn("No current user found")
		return ctx.Redirect("/auth/login")
	}
	return ctx.Next()
}
