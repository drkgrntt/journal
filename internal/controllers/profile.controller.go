package controllers

import (
	"go-starter/cmd/web/profile"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
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
	c.views.Get("/", middleware.SetRatings, middleware.SetJournalTypes, utils.RenderPage(profile.ProfilePage))
}

func (c *ProfileController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Put("/", c.updateUser)
	c.api.Put("/password", c.updatePassword)
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
