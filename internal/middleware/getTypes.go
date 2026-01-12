package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"

	"github.com/gofiber/fiber/v2"
)

func SetJournalTypes(ctx *fiber.Ctx) error {
	var types []*models.JournalType
	err := db.Order("name ASC").Find(&types).Error
	if err != nil {
		logger.Error("Error getting journal types: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals("journalTypes", &types)
	return ctx.Next()
}

func SetRatings(ctx *fiber.Ctx) error {
	var ratings []*models.Rating
	err := db.Order("value DESC").Find(&ratings).Error
	if err != nil {
		logger.Error("Error getting ratings: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals("ratings", &ratings)
	return ctx.Next()
}
