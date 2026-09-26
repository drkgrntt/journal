package controllers

import (
	"fmt"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/profile"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/stripe/stripe-go/v84"
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
		middleware.SetFeatures,
		utils.RenderPage(profile.ProfilePage),
	)
}

func (c *ProfileController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Put("/", c.updateUser)
	c.api.Put("/password", c.updatePassword)
	c.api.Put("/features", c.updateFeatures)
	c.api.Put("/features/:id", c.buyFeature)
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
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

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

func (c *ProfileController) buyFeature(ctx *fiber.Ctx) error {
	var feature *models.Feature
	err := c.db.Where("enabled_at < ?", time.Now().UTC()).Where("id = ?", ctx.Params("id")).First(&feature).Error
	if err != nil {
		return ctx.Status(http.StatusNotFound).SendString("Feature not found")
	}

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	if feature.StripePriceID == "" {
		err := c.db.Save(&models.UserFeature{
			FeatureID: feature.ID,
			EnabledAt: utils.Pointer(time.Now().UTC()),
			Base: &models.Base{
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
				CreatorID:     &currentUser.ID,
				LastUpdaterID: &currentUser.ID,
			},
		}).Error
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).SendString("Error adding feature")
		}
		return ctx.Status(http.StatusOK).JSON(feature)
	}

	// Create the customer in stripe if applicable
	if currentUser.StripeCustomerID == "" {
		params := &stripe.CustomerParams{
			Name:  stripe.String(fmt.Sprintf("%s %s", currentUser.FirstName, currentUser.LastName)),
			Email: stripe.String(currentUser.Email),
		}
		result, err := customer.New(params)
		if err != nil {
			logger.Error("Error creating stripe customer", "error", err)
			return ctx.Status(http.StatusInternalServerError).SendString("Error creating stripe customer")
		}
		currentUser.StripeCustomerID = result.ID
		c.db.Save(&currentUser)
	}

	// Send the user to stripe checkout
	params := &stripe.CheckoutSessionParams{
		AllowPromotionCodes: stripe.Bool(true),
		Customer:            stripe.String(currentUser.StripeCustomerID),
		PaymentMethodTypes:  stripe.StringSlice([]string{string(stripe.PaymentMethodTypeCard)}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(feature.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(fmt.Sprintf("%s/profile", os.Getenv("ROOT_URL"))),
		CancelURL:  stripe.String(fmt.Sprintf("%s/profile", os.Getenv("ROOT_URL"))),
	}
	result, err := session.New(params)
	if err != nil {
		logger.Error("Error creating stripe checkout session", "error", err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating stripe checkout session")
	}

	ctx.Set("HX-Redirect", result.URL)
	return ctx.SendStatus(http.StatusOK)
}

type UpdateFeaturesBody struct {
	Features []int `form:"features"`
}

func (c *ProfileController) updateFeatures(ctx *fiber.Ctx) error {
	var body UpdateFeaturesBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	userFeaturesToUpdate := []*models.UserFeature{}

disableFeatureLoop:
	for _, userFeature := range currentUser.UserFeatures {
		if !userFeature.IsEnabled() {
			continue disableFeatureLoop
		}
		for _, bodyFeatureId := range body.Features {
			if bodyFeatureId == userFeature.FeatureID {
				continue disableFeatureLoop
			}
		}
		userFeature.EnabledAt = nil
		userFeature.UpdatedAt = time.Now().UTC()
		userFeaturesToUpdate = append(userFeaturesToUpdate, userFeature)
	}

enableFeatureLoop:
	for _, bodyFeatureId := range body.Features {
		for _, userFeatureToUpdate := range userFeaturesToUpdate {
			if bodyFeatureId == userFeatureToUpdate.FeatureID {
				continue enableFeatureLoop
			}
		}

	userFeatureLoop:
		for _, userFeature := range currentUser.UserFeatures {
			if userFeature.FeatureID != bodyFeatureId {
				continue userFeatureLoop
			}
			if userFeature.IsEnabled() {
				continue userFeatureLoop
			}

			userFeature.EnabledAt = utils.Pointer(time.Now().UTC())
			userFeature.UpdatedAt = time.Now().UTC()
			userFeaturesToUpdate = append(userFeaturesToUpdate, userFeature)
		}
	}

	var newFeatureIds []int
bodyFeatureLoop:
	for _, bodyFeatureId := range body.Features {
		for _, userFeature := range currentUser.UserFeatures {
			if bodyFeatureId == userFeature.FeatureID {
				continue bodyFeatureLoop
			}
		}
		newFeatureIds = append(newFeatureIds, bodyFeatureId)
	}

	if len(userFeaturesToUpdate) > 0 {
		err = c.db.Save(&userFeaturesToUpdate).Error
		if err != nil {
			logger.Warn(err.Error())
			return ctx.Status(http.StatusInternalServerError).SendString("Error updating features")
		}
	}

	if len(newFeatureIds) > 0 {
		userFeatures := []*models.UserFeature{}
		for _, featureId := range newFeatureIds {
			uf := &models.UserFeature{
				FeatureID: featureId,
				EnabledAt: utils.Pointer(time.Now().UTC()),
				Base: &models.Base{
					CreatorID:     &currentUser.ID,
					LastUpdaterID: &currentUser.ID,
					CreatedAt:     time.Now().UTC(),
					UpdatedAt:     time.Now().UTC(),
				},
			}
			userFeatures = append(userFeatures, uf)
		}
		err = c.db.Create(&userFeatures).Error
		if err != nil {
			logger.Warn(err.Error())
			return ctx.Status(http.StatusInternalServerError).SendString("Error adding features")
		}
	}

	return ctx.Status(http.StatusOK).JSON(currentUser)
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

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

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

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
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
