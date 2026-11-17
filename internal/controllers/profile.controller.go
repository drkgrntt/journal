package controllers

import (
	"fmt"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	appstripe "journal/internal/stripe"
	"journal/internal/utils"
	"journal/internal/web/profile"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v84"
	portal "github.com/stripe/stripe-go/v84/billingportal/session"
	"github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/customer"
	"gorm.io/gorm"
)

func init() {
	registerController(&ProfileController{})
}

type ProfileController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *ProfileController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("profile")
	c.api = app.Group("api/profile")
}

func (c *ProfileController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)
	c.views.Get("/",
		middleware.SetRatings,
		middleware.SetJournalTypes,
		utils.RenderPage(profile.ProfilePage),
	)
}

func (c *ProfileController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Put("/", c.updateUser)
	c.api.Put("/password", c.updatePassword)
	c.api.Put("/appearance", middleware.RequireSubscription, c.updateAppearance)
	c.api.Post("/billing/checkout/:plan", c.startCheckout)
	c.api.Post("/billing/portal", c.openPortal)
	c.api.Get("/export", c.exportJournals)
}

type exportedActionItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

type exportedJournal struct {
	Date         string               `json:"date"`
	Topic        string               `json:"topic"`
	Rating       string               `json:"rating,omitempty"`
	IsBookmarked bool                 `json:"isBookmarked"`
	Entry        string               `json:"entry"`
	ActionItems  []exportedActionItem `json:"actionItems,omitempty"`
	Thankfuls    []string             `json:"thankfuls,omitempty"`
}

func (c *ProfileController) exportJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var journals []*models.Journal
	err := c.db.
		Where("creator_id = ?", currentUser.ID).
		Preload("JournalType").
		Preload("CustomJournalType").
		Preload("Rating").
		Preload("ActionItems", func(db *gorm.DB) *gorm.DB { return db.Order("created_at asc") }).
		Preload("Thankfuls", func(db *gorm.DB) *gorm.DB { return db.Order("created_at asc") }).
		Order("date asc").
		Find(&journals).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error exporting journals")
	}

	exports := make([]exportedJournal, 0, len(journals))
	for _, j := range journals {
		topic := ""
		if j.JournalType != nil && j.JournalType.IsCustom() && j.CustomJournalType != nil {
			topic = j.CustomJournalType.Name
		} else if j.JournalType != nil {
			topic = j.JournalType.Name
		}

		rating := ""
		if j.Rating != nil {
			rating = j.Rating.Name
		}

		actionItems := make([]exportedActionItem, 0, len(j.ActionItems))
		for _, a := range j.ActionItems {
			actionItems = append(actionItems, exportedActionItem{Text: a.Text, Completed: a.IsComplete()})
		}

		thankfuls := make([]string, 0, len(j.Thankfuls))
		for _, t := range j.Thankfuls {
			thankfuls = append(thankfuls, t.Text)
		}

		date := ""
		if j.Date != nil {
			date = j.Date.UTC().Format("2006-01-02")
		}

		exports = append(exports, exportedJournal{
			Date:         date,
			Topic:        topic,
			Rating:       rating,
			IsBookmarked: j.BookmarkedAt != nil,
			Entry:        j.Entry,
			ActionItems:  actionItems,
			Thankfuls:    thankfuls,
		})
	}

	filename := fmt.Sprintf("contour-journal-export-%s", time.Now().UTC().Format("2006-01-02"))

	if ctx.Query("format") == "json" {
		ctx.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, filename))
		return ctx.JSON(exports)
	}

	var sb strings.Builder
	sb.WriteString("# Contour Journal Export\n\n")
	for _, e := range exports {
		sb.WriteString(fmt.Sprintf("## %s — %s", e.Date, e.Topic))
		if e.Rating != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", e.Rating))
		}
		if e.IsBookmarked {
			sb.WriteString(" ★")
		}
		sb.WriteString("\n\n")
		sb.WriteString(e.Entry)
		sb.WriteString("\n")

		if len(e.ActionItems) > 0 {
			sb.WriteString("\n**Action Items:**\n")
			for _, a := range e.ActionItems {
				mark := " "
				if a.Completed {
					mark = "x"
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", mark, a.Text))
			}
		}

		if len(e.Thankfuls) > 0 {
			sb.WriteString("\n**Thankful for:**\n")
			for _, t := range e.Thankfuls {
				sb.WriteString(fmt.Sprintf("- %s\n", t))
			}
		}

		sb.WriteString("\n---\n\n")
	}

	ctx.Set("Content-Type", "text/markdown; charset=utf-8")
	ctx.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.md"`, filename))
	return ctx.SendString(sb.String())
}

// ensureStripeCustomer returns the user's Stripe customer ID, creating the
// customer on first use.
func (c *ProfileController) ensureStripeCustomer(user *models.User) (string, error) {
	if user.StripeCustomerID != "" {
		return user.StripeCustomerID, nil
	}
	result, err := customer.New(&stripe.CustomerParams{
		Name:  stripe.String(user.FullName()),
		Email: stripe.String(user.Email),
		Metadata: map[string]string{
			"user_id": user.ID.String(),
		},
	})
	if err != nil {
		return "", err
	}
	err = c.db.Model(&models.User{}).Where("id = ?", user.ID).Update("stripe_customer_id", result.ID).Error
	if err != nil {
		return "", err
	}
	user.StripeCustomerID = result.ID
	return result.ID, nil
}

// startCheckout sends the user to Stripe Checkout for the plan in :plan.
func (c *ProfileController) startCheckout(ctx *fiber.Ctx) error {
	plan, ok := appstripe.ParsePlan(ctx.Params("plan"))
	if !ok {
		return ctx.Status(http.StatusNotFound).SendString("Plan not found")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	switch currentUser.SubscriptionStatus {
	case models.SubscriptionLifetime, models.SubscriptionComped:
		return ctx.Status(http.StatusBadRequest).SendString("You don't need a plan")
	case models.SubscriptionActive, models.SubscriptionPastDue:
		return ctx.Status(http.StatusBadRequest).SendString("You already have a plan; manage it from your profile")
	}

	price, err := appstripe.GetPlanPrice(plan)
	if err != nil {
		logger.Error("Error getting plan price", "plan", string(plan), "error", err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error finding that plan")
	}

	customerID, err := c.ensureStripeCustomer(currentUser)
	if err != nil {
		logger.Error("Error creating stripe customer", "error", err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating stripe customer")
	}

	params := &stripe.CheckoutSessionParams{
		AllowPromotionCodes: stripe.Bool(true),
		Customer:            stripe.String(customerID),
		ClientReferenceID:   stripe.String(currentUser.ID.String()),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s/profile?checkout=success", os.Getenv("ROOT_URL"))),
		CancelURL:  stripe.String(fmt.Sprintf("%s/profile", os.Getenv("ROOT_URL"))),
	}
	if plan.IsRecurring() {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModeSubscription))
	} else {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
	}
	result, err := session.New(params)
	if err != nil {
		logger.Error("Error creating stripe checkout session", "error", err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating stripe checkout session")
	}

	ctx.Set("HX-Redirect", result.URL)
	return ctx.SendStatus(http.StatusOK)
}

// openPortal sends the user to the Stripe customer portal to manage or cancel.
func (c *ProfileController) openPortal(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if currentUser.StripeCustomerID == "" {
		return ctx.Status(http.StatusBadRequest).SendString("There's nothing to manage yet")
	}

	result, err := portal.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(currentUser.StripeCustomerID),
		ReturnURL: stripe.String(fmt.Sprintf("%s/profile", os.Getenv("ROOT_URL"))),
	})
	if err != nil {
		logger.Error("Error creating stripe portal session", "error", err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error opening billing")
	}

	ctx.Set("HX-Redirect", result.URL)
	return ctx.SendStatus(http.StatusOK)
}

type UpdateUserBody struct {
	FirstName string `form:"firstName"`
	LastName  string `form:"lastName"`
}

func (c *ProfileController) updateUser(ctx *fiber.Ctx) error {
	var body UpdateUserBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	currentUser.FirstName = body.FirstName
	currentUser.LastName = body.LastName

	err = c.db.Save(currentUser).Error
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error updating user")
	}

	return ctx.Status(http.StatusOK).JSON(currentUser)
}

type UpdatePasswordBody struct {
	CurrentPassword    string `form:"currentPassword"`
	NewPassword        string `form:"newPassword"`
	ConfirmNewPassword string `form:"confirmNewPassword"`
}

func (c *ProfileController) updatePassword(ctx *fiber.Ctx) error {
	var body UpdatePasswordBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	err = currentUser.ComparePasswords(body.CurrentPassword)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).SendString("Current password is incorrect")
	}

	if body.NewPassword != body.ConfirmNewPassword {
		return ctx.Status(http.StatusBadRequest).SendString("Passwords do not match")
	}

	err = c.db.Model(currentUser).
		Updates(&models.User{Password: body.NewPassword}).Error
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error updating user")
	}

	return ctx.SendStatus(http.StatusAccepted)
}

type UpdateAppearanceBody struct {
	Theme      string `form:"theme"`
	FontFamily string `form:"fontFamily"`
}

// updateAppearance saves the palette and font choice. Both are validated
// against an allowlist; the route is behind RequireSubscription.
func (c *ProfileController) updateAppearance(ctx *fiber.Ctx) error {
	var body UpdateAppearanceBody
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}
	if !models.IsValidTheme(body.Theme) || !models.IsValidFontFamily(body.FontFamily) {
		return ctx.Status(http.StatusBadRequest).SendString("Unknown appearance option")
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	err := c.db.Model(currentUser).Updates(map[string]any{
		"theme":       body.Theme,
		"font_family": body.FontFamily,
	}).Error
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusInternalServerError).SendString("Error saving appearance")
	}
	return ctx.SendStatus(http.StatusOK)
}
