package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

func SetJournalTypes(ctx *fiber.Ctx) error {
	var types []*models.JournalType
	err := db.Where("code != ?", "custom").Order("name ASC").Find(&types).Error
	if err != nil {
		logger.Error("Error getting journal types: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals(utils.LocalJournalTypes, &types)
	return ctx.Next()
}

func SetRatings(ctx *fiber.Ctx) error {
	var ratings []*models.Rating
	err := db.Order("value DESC").Find(&ratings).Error
	if err != nil {
		logger.Error("Error getting ratings: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals(utils.LocalRatings, &ratings)
	return ctx.Next()
}

func SetFeatures(ctx *fiber.Ctx) error {
	var features []*models.Feature
	err := db.Where("enabled_at < ?", time.Now().UTC()).Find(&features).Error
	if err != nil {
		logger.Error("Error getting features: ", "error message", err.Error())
		return ctx.Next()
	}

	ctx.Locals(utils.LocalFeatures, &features)
	return ctx.Next()
}
