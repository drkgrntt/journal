package middleware

import (
	"github.com/gofiber/fiber/v2"
	"zgo.at/isbot"
)

func FilterBots(ctx *fiber.Ctx) error {
	useragent := string(ctx.Context().UserAgent())
	result := isbot.UserAgent(useragent)
	if isbot.Is(result) {
		return ctx.SendStatus(fiber.StatusForbidden)
	}
	return ctx.Next()
}
