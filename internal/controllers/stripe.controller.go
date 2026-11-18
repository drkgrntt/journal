package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"journal/internal/logger"
	"journal/internal/models"
	appstripe "journal/internal/stripe"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/subscription"
	"github.com/stripe/stripe-go/v84/webhook"
	"gorm.io/gorm"
)

type StripeController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func init() {
	registerController(&StripeController{})
}

func (c *StripeController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("stripe")
	c.api = app.Group("api/stripe")
}

func (c *StripeController) RegisterViewRoutes() {}

func (c *StripeController) RegisterApiRoutes() {
	c.api.Post("webhook", c.handleWebhook)
}

func (c *StripeController) handleWebhook(ctx *fiber.Ctx) error {
	// Pass the request body and Stripe-Signature header to ConstructEvent, along
	// with the webhook signing key.
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(ctx.Body(), ctx.Get("Stripe-Signature"),
		endpointSecret)

	if err != nil {
		logger.Error("Error verifying webhook signature", "error", err)
		return ctx.Status(http.StatusBadRequest).SendString(fmt.Sprintf("Error verifying webhook signature: %v\n", err))
	}

	// Every handler is idempotent: Stripe can deliver an event more than once
	// and out of order.
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		err = c.handleCheckoutSessionCompleted(&event)
	case stripe.EventTypeCustomerSubscriptionCreated,
		stripe.EventTypeCustomerSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted:
		err = c.handleSubscriptionChanged(&event)
	}

	if err != nil {
		logger.Error("Error handling webhook", "type", string(event.Type), "error", err)
		return ctx.SendStatus(http.StatusBadRequest)
	}

	return ctx.SendStatus(http.StatusOK)
}

func (c *StripeController) handleSubscriptionChanged(event *stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("parsing subscription: %w", err)
	}

	status, periodEnd, ok := appstripe.SubscriptionState(&sub)
	if !ok {
		return nil
	}
	if sub.Customer == nil {
		return fmt.Errorf("subscription %s has no customer", sub.ID)
	}

	var user models.User
	if err := c.db.Where("stripe_customer_id = ?", sub.Customer.ID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// The customer ID is saved before checkout starts, so this account
			// no longer exists (deleted since). A retry can't change that, so
			// acknowledge the event instead of making Stripe redeliver it for days.
			logger.Warn("Subscription event for an unknown customer, ignoring", "subscription", sub.ID, "type", string(event.Type))
			return nil
		}
		return fmt.Errorf("finding user for customer %s: %w", sub.Customer.ID, err)
	}

	// Lifetime and comped accounts don't change with subscription events, and
	// a late event for a subscription that was replaced shouldn't clobber the
	// current one.
	if user.SubscriptionStatus == models.SubscriptionLifetime || user.SubscriptionStatus == models.SubscriptionComped {
		return nil
	}
	if user.StripeSubscriptionID != "" && user.StripeSubscriptionID != sub.ID {
		return nil
	}

	return c.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"subscription_status":    status,
		"current_period_end":     periodEnd,
		"stripe_subscription_id": sub.ID,
		"updated_at":             time.Now().UTC(),
	}).Error
}

func (c *StripeController) handleCheckoutSessionCompleted(event *stripe.Event) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("parsing checkout session: %w", err)
	}

	userID, err := uuid.Parse(session.ClientReferenceID)
	if err != nil {
		return fmt.Errorf("checkout session %s has no valid client_reference_id", session.ID)
	}
	var user models.User
	if err := c.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("finding user %s: %w", userID, err)
	}

	switch session.Mode {
	case stripe.CheckoutSessionModeSubscription:
		// The subscription events carry the status and period; this just
		// links the subscription so they can be matched up.
		if session.Subscription == nil || user.SubscriptionStatus == models.SubscriptionLifetime || user.SubscriptionStatus == models.SubscriptionComped {
			return nil
		}
		return c.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"stripe_subscription_id": session.Subscription.ID,
			"updated_at":             time.Now().UTC(),
		}).Error

	case stripe.CheckoutSessionModePayment:
		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			return nil
		}
		// Someone going lifetime shouldn't keep paying monthly or yearly.
		if user.StripeSubscriptionID != "" && user.SubscriptionStatus != models.SubscriptionLifetime {
			if _, err := subscription.Cancel(user.StripeSubscriptionID, nil); err != nil {
				logger.Error("Error canceling subscription after lifetime purchase", "user", user.ID.String(), "error", err)
			}
		}
		return c.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"subscription_status":    models.SubscriptionLifetime,
			"current_period_end":     nil,
			"stripe_subscription_id": "",
			"updated_at":             time.Now().UTC(),
		}).Error
	}
	return nil
}
