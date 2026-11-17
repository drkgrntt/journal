package controllers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"journal/internal/emails"
	"journal/internal/jobs"
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// sendEmailConfirmation queues the confirmation email and stamps the send time
// (used to throttle resends). It does nothing for an already-confirmed user.
func sendEmailConfirmation(db *gorm.DB, user *models.User) error {
	if user.EmailConfirmed() {
		return nil
	}
	token, err := utils.CreateEmailConfirmToken(user.ID, user.Email)
	if err != nil {
		return err
	}
	data := &jobs.EmailData{
		Name: emails.CONFIRM_EMAIL,
		Recipients: []*emails.EmailRecipient{
			{Email: user.Email, Name: user.FullName()},
		},
		Variables: &emails.ConfirmEmailVariables{
			RootUrl: os.Getenv("ROOT_URL"),
			Token:   token,
		},
	}
	if _, err := jobs.ScheduleEmailJob(user.ID, data, time.Now()); err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("id = ?", user.ID).
		Update("email_confirmation_sent_at", time.Now()).Error
}

// resendEmailConfirmation is the profile page's "Send confirmation" button.
func (c *ProfileController) resendEmailConfirmation(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if currentUser == nil {
		return ctx.SendStatus(http.StatusUnauthorized)
	}
	if currentUser.EmailConfirmed() {
		return ctx.Status(http.StatusOK).SendString("Your email is already confirmed.")
	}
	if !currentUser.CanResendEmailConfirmation(time.Now()) {
		return ctx.Status(http.StatusTooManyRequests).SendString("A confirmation email was just sent. Give it a minute.")
	}
	if err := sendEmailConfirmation(c.db, currentUser); err != nil {
		logger.Error("Error sending email confirmation", "error", err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error sending the confirmation email")
	}
	return ctx.Status(http.StatusOK).SendString("Check your email for a confirmation link.")
}

// confirmEmail handles the link in the confirmation email. It needs no login,
// since the link may be opened on another device. The token is bound to the
// address it was issued for, so it stops working if that address changes.
func (c *AuthController) confirmEmail(ctx *fiber.Ctx) error {
	confirmed := false
	ctx.Locals(utils.LocalEmailConfirmed, &confirmed)

	userID, email, err := utils.ValidateEmailConfirmToken(ctx.Query("token"))
	if err != nil {
		return ctx.Next()
	}

	var user models.User
	if err := c.db.First(&user, "id = ?", userID).Error; err != nil {
		return ctx.Next()
	}
	if strings.ToLower(user.Email) != email {
		return ctx.Next()
	}

	if !user.EmailConfirmed() {
		if err := c.db.Model(&models.User{}).Where("id = ?", user.ID).
			Update("email_confirmed_at", time.Now()).Error; err != nil {
			logger.Error("Error confirming email", "error", err.Error())
			return ctx.Next()
		}
	}
	confirmed = true
	return ctx.Next()
}
