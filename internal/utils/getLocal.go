package utils

import (
	"go-starter/internal/logger"

	"github.com/gofiber/fiber/v2"
)

// GetLocal retrieves a value of any specified type from the context locals.
func GetLocal[T any](ctx *fiber.Ctx, key string) *T {
	valueInterface := ctx.Locals(key)
	if valueInterface == nil {
		logger.Warn("No locals found for key: ", key)
		return nil
	}
	value, ok := valueInterface.(*T)
	if !ok {
		logger.Warn("Unable to cast interface for key: ", key)
		return nil
	}

	return value
}
