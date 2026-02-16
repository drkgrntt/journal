package utils

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func GetDateInLocal(c *fiber.Ctx, date *time.Time) *time.Time {
	tz := c.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	if date == nil {
		t := time.Now()
		date = &t
	}

	localized := date.In(loc)

	return &localized
}

func Pointer[T any](value T) *T {
	return &value
}

func Value[T any](pointer *T) T {
	return *pointer
}
