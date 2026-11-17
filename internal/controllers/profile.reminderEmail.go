package controllers

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type UpdateReminderEmailBody struct {
	Enabled string `form:"enabled"`
	Time    string `form:"time"`
	Message string `form:"message"`
}

// updateReminderEmail saves the reminder-email settings. Turning it on or
// changing it needs an active plan (the same rule as middleware.RequireSubscription,
// applied inline); turning it off is always allowed so lapsed users can opt out.
func (c *ProfileController) updateReminderEmail(ctx *fiber.Ctx) error {
	var body UpdateReminderEmailBody
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if currentUser == nil {
		return ctx.SendStatus(http.StatusUnauthorized)
	}

	enabled := body.Enabled == "true" || body.Enabled == "on"

	if !enabled {
		// Disabling only: leave the saved time and message alone.
		err := c.db.Model(&models.User{}).Where("id = ?", currentUser.ID).
			Update("reminder_email_enabled", false).Error
		if err != nil {
			logger.Warn(err.Error())
			return ctx.Status(http.StatusInternalServerError).SendString("Error updating reminder email")
		}
		return ctx.SendStatus(http.StatusOK)
	}

	if !currentUser.IsSubscribed() {
		return ctx.Status(http.StatusForbidden).SendString("This is part of the plan, and your plan isn't active right now")
	}
	if !models.ValidReminderEmailTime(body.Time) {
		return ctx.Status(http.StatusBadRequest).SendString("Time must look like 08:00")
	}
	message, err := models.CleanReminderEmailMessage(body.Message)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err := currentUser.SetReminderEmailMessage(message); err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error updating reminder email")
	}

	// The job runner has no request, so remember the browser's timezone now.
	timezone := models.NormalizeTimezone(ctx.Cookies("tz", "UTC"))

	err = c.db.Model(&models.User{}).Where("id = ?", currentUser.ID).Updates(map[string]any{
		"reminder_email_enabled": true,
		"reminder_email_time":    body.Time,
		"reminder_email_message": currentUser.ReminderEmailMessage,
		"timezone":               timezone,
	}).Error
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error updating reminder email")
	}
	return ctx.SendStatus(http.StatusOK)
}
