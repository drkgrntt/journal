package web

import (
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func HasMore(c *fiber.Ctx) bool {
	hasMore := utils.GetLocal[bool](c, "hasMore")
	if hasMore == nil {
		return false
	}
	return *hasMore
}

func NextPage(c *fiber.Ctx) int {
	nextPage := utils.GetLocal[int](c, "nextPage")
	if nextPage == nil {
		return 0
	}
	return *nextPage
}

func NextDate(c *fiber.Ctx) string {
	nextDate := utils.GetLocal[string](c, "nextDate")
	if nextDate == nil {
		return ""
	}
	return *nextDate
}

func PrevDate(c *fiber.Ctx) string {
	prevDate := utils.GetLocal[string](c, "prevDate")
	if prevDate == nil {
		return ""
	}
	return *prevDate
}

func CurrentUser(c *fiber.Ctx) *models.User {
	return utils.GetLocal[models.User](c, "currentUser")
}

type LabelValue[T any] struct {
	Label string
	Value T
}
