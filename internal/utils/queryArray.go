package utils

import "github.com/gofiber/fiber/v2"

func QueryArray(c *fiber.Ctx, key string) []string {
	queryArgs := c.Context().QueryArgs()
	values := queryArgs.PeekMulti(key)

	result := make([]string, len(values))
	for i, v := range values {
		result[i] = string(v)
	}
	return result
}
