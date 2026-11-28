package controllers

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/subscription"
	"gorm.io/gorm"
)

type DeleteAccountBody struct {
	Password string `form:"password"`
}

// deleteAccount removes the current user and everything they own, for good. It
// only ever acts on the signed-in account, and asks for the password again.
func (c *ProfileController) deleteAccount(ctx *fiber.Ctx) error {
	var body DeleteAccountBody
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if err := currentUser.ComparePasswords(body.Password); err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Password is incorrect")
	}

	// Billing goes first: an account that's gone can't be used to stop charges.
	cancel := c.cancelBilling
	if cancel == nil {
		cancel = cancelSubscriptions
	}
	if err := cancel(currentUser); err != nil {
		logger.Error("Error canceling subscription before account deletion", "error", err)
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Your subscription couldn't be canceled, so nothing was deleted. Try again in a while, or reach out through the feedback form.")
	}

	if err := c.db.Transaction(func(tx *gorm.DB) error { return eraseUser(tx, currentUser) }); err != nil {
		logger.Error("Error deleting account", "error", err)
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Your account couldn't be deleted. Nothing was removed.")
	}

	utils.ClearSessionCookie(ctx)
	// No HX-Redirect: the page first wipes what's stored on the device, then
	// leaves (see utils/offline.js).
	return ctx.SendStatus(http.StatusNoContent)
}

// eraseUser hard deletes every row the user owns, then the user. Only sessions
// have a foreign key that would cascade, so each table is listed. Queued and
// sent email jobs are included because they hold the address.
func eraseUser(tx *gorm.DB, user *models.User) error {
	owned := []any{
		&models.ActionItem{},
		&models.Thankful{},
		&models.Journal{},
		&models.Reminder{},
		&models.EntryTemplate{},
		&models.RecurringActionItem{},
		&models.CustomJournalType{},
		&models.Job{},
	}
	for _, model := range owned {
		if err := tx.Unscoped().Where("creator_id = ?", user.ID).Delete(model).Error; err != nil {
			return err
		}
	}
	if err := models.DeleteUserSessions(tx, user.ID, nil); err != nil {
		return err
	}
	return tx.Unscoped().Where("id = ?", user.ID).Delete(&models.User{}).Error
}

// cancelSubscriptions ends every live subscription of the user's Stripe
// customer. It asks Stripe rather than trusting StripeSubscriptionID, which is
// only filled in once the webhook arrives. The customer record is kept for the
// payment history.
func cancelSubscriptions(user *models.User) error {
	if user.StripeCustomerID == "" {
		return nil
	}
	// The list leaves out subscriptions that are already canceled.
	iter := subscription.List(&stripe.SubscriptionListParams{Customer: stripe.String(user.StripeCustomerID)})
	for iter.Next() {
		if _, err := subscription.Cancel(iter.Subscription().ID, nil); err != nil {
			return err
		}
	}
	return iter.Err()
}
