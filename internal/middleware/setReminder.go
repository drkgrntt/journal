package middleware

import (
	"hash/fnv"
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// SetDailyReminder picks one of the user's reminders for today and stores it in
// the "reminder" local. The pick is derived from the user and their local date, so
// it stays the same across refreshes without anything being stored or tracked.
func SetDailyReminder(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	if currentUser == nil {
		return ctx.Next()
	}

	var reminders []*models.Reminder
	err := db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at, id").
		Find(&reminders).Error
	if err != nil {
		logger.Error("Error getting reminders: ", "error message", err.Error())
		return ctx.Next()
	}
	if len(reminders) == 0 {
		return ctx.Next()
	}

	today := utils.GetDateInLocal(ctx, nil).Format("2006-01-02")
	h := fnv.New32a()
	h.Write([]byte(currentUser.ID.String() + today))
	reminder := reminders[int(h.Sum32()%uint32(len(reminders)))]

	ctx.Locals("reminder", reminder)
	return ctx.Next()
}
