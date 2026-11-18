package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func SetJournalTypes(ctx *fiber.Ctx) error {
	var types []*models.JournalType
	err := DB().Where("code != ?", "custom").Order("name ASC").Find(&types).Error
	if err != nil {
		logger.Error("Error getting journal types: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals(utils.LocalJournalTypes, &types)
	return ctx.Next()
}

func SetRatings(ctx *fiber.Ctx) error {
	var ratings []*models.Rating
	err := DB().Order("value DESC").Find(&ratings).Error
	if err != nil {
		logger.Error("Error getting ratings: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals(utils.LocalRatings, &ratings)
	return ctx.Next()
}
