package middleware

import (
	"go-starter/internal/logger"
	"go-starter/internal/models"
	"go-starter/internal/utils"

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
