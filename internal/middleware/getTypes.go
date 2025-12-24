package middleware

import (
	"go-starter/internal/database"
	"go-starter/internal/logger"
	"go-starter/internal/models"

	"github.com/gofiber/fiber/v2"
)

func SetJournalTypes(ctx *fiber.Ctx) error {
	db := database.New()

	var types []*models.JournalType
	err := db.DB.Order("name ASC").Find(&types).Error
	if err != nil {
		logger.Error("Error getting journal types: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals("journalTypes", &types)
	return ctx.Next()
}

func SetRatings(ctx *fiber.Ctx) error {
	db := database.New()

	var ratings []*models.Rating
	err := db.DB.Order("value DESC").Find(&ratings).Error
	if err != nil {
		logger.Error("Error getting ratings: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals("ratings", &ratings)
	return ctx.Next()
}
