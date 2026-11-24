package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

// DeserializeToken looks up the session the x-token cookie belongs to and, if
// it is live, sets the current user. It never blocks the request. A session's
// token is replaced about once a day (the previous one keeps working for a few
// minutes), which also extends it, so regular use stays signed in.
func DeserializeToken(ctx *fiber.Ctx) error {
	token := ctx.Cookies(utils.SessionCookieName)
	if token == "" {
		return ctx.Next()
	}

	now := time.Now().UTC()
	session, err := models.FindSession(DB(), token, now)
	if err != nil {
		return ctx.Next()
	}

	var user models.User
	err = DB().Where("id = ?", session.UserID).
		Preload("CustomJournalTypes").
		First(&user).Error
	if err != nil {
		logger.Error("Error getting user: ", err.Error())
		return ctx.Next()
	}

	if session.NeedsRotation(token, now) {
		if fresh, rotated, err := models.RotateSession(DB(), session, now); err != nil {
			logger.Error("Error rotating session: ", err.Error())
		} else if rotated {
			utils.SetSessionCookie(ctx, fresh)
		}
	}

	ctx.Locals(utils.LocalCurrentUser, &user)
	ctx.Locals(utils.LocalSessionID, &session.ID)
	return ctx.Next()
}
