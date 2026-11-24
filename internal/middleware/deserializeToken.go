package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func DeserializeToken(ctx *fiber.Ctx) error {
	token := ctx.Cookies("x-token")

	if token == "" {
		return ctx.Next()
	}

	data, err := utils.ValidateToken(token)
	if err != nil {
		logger.Error("Error validating token: ", err.Error())
		return ctx.Next()
	}

	var user models.User
	err = DB().Where("id = ?", data.UserID).
		Preload("CustomJournalTypes").
		First(&user).Error
	if err != nil {
		logger.Error("Error getting user: ", err.Error())
		return ctx.Next()
	}

	// A password change or reset bumps the version, ending older sessions.
	if user.TokenVersion != data.Version {
		return ctx.Next()
	}

	ctx.Locals(utils.LocalCurrentUser, &user)
	return ctx.Next()
}
