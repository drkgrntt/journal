package web

import (
	"go-starter/internal/utils"

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
