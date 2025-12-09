package middleware

import (
	"go-starter/internal/database"
	"go-starter/internal/logger"
	"go-starter/internal/models"
	"go-starter/internal/utils"

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

	db := database.New()

	var user models.User
	err = db.DB.Where("user_id = ?", data.UserID).
		First(&user).Error
	if err != nil {
		logger.Error("Error getting user: ", err.Error())
		return ctx.Next()
	}

	ctx.Locals("currentUser", user)
	return ctx.Next()
}
