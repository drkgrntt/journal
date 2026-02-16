package controllers

import (
	"encoding/json"
	"fmt"
	"journal/internal/logger"
	"journal/internal/models"
	_ "journal/internal/stripe"
	"journal/internal/utils"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/product"
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
	// payload, err := io.ReadAll(ctx.Request().BodyStream())
	// if err != nil {
	// 	fmt.Printf("Error reading request body: %v\n", err)
	// 	return ctx.Status(http.StatusServiceUnavailable).SendString(fmt.Sprintf("Error reading request body: %v\n", err))
	// }

	// Pass the request body and Stripe-Signature header to ConstructEvent, along
	// with the webhook signing key.
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	event, err := webhook.ConstructEvent(ctx.Body(), ctx.Get("Stripe-Signature"),
		endpointSecret)

	if err != nil {
		fmt.Printf("Error verifying webhook signature: %v\n", err)
		return ctx.Status(http.StatusBadRequest).SendString(fmt.Sprintf("Error verifying webhook signature: %v\n", err))
	}

	switch event.Type {
	case stripe.EventTypePaymentIntentSucceeded:
		err = c.handlePaymentIntentSucceeded(&event)
	}

	if err != nil {
		log.Printf("Error handling webhook: %v\n", err)
		return ctx.SendStatus(http.StatusBadRequest)
	}

	return ctx.SendStatus(http.StatusOK)
}

func (c *StripeController) getUser(stripeCustomerId string) (user *models.User) {
	c.db.Where("stripe_customer_id = ?", stripeCustomerId).First(&user)
	return
}

func (c *StripeController) handlePaymentIntentSucceeded(event *stripe.Event) error {
	var paymentIntent *stripe.PaymentIntent
	err := json.Unmarshal(event.Data.Raw, &paymentIntent)
	if err != nil {
		log.Printf("Error parsing webhook JSON: %v\n", err)
		return err
	}

	productId := paymentIntent.PaymentDetails.OrderReference
	foundProduct, err := product.Get(productId, &stripe.ProductParams{})
	if err != nil {
		logger.Error("Error getting product", "error", err)
		return err
	}

	var feature *models.Feature
	err = c.db.Where("stripe_price_id = ?", foundProduct.DefaultPrice.ID).First(&feature).Error
	if err != nil {
		logger.Error("Error getting feature", "error", err)
		return err
	}

	user := c.getUser(paymentIntent.Customer.ID)
	err = c.db.Save(&models.UserFeature{
		FeatureID: feature.ID,
		EnabledAt: utils.Pointer(time.Now().UTC()),
		Base: &models.Base{
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
			CreatorID:     &user.ID,
			LastUpdaterID: &user.ID,
		},
	}).Error
	if err != nil {
		logger.Error("Error saving user feature", "error", err)
		return err
	}

	return nil
}
