package controllers

import (
	"go-starter/cmd/web/auth"
	"go-starter/internal/logger"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerController(&AuthController{})
}

type AuthController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *AuthController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("auth")
	c.api = app.Group("api/auth")
}

func (c *AuthController) RegisterViewRoutes() {
	c.views.Get("/register", utils.RenderPage(auth.RegisterPage))
	c.views.Get("/login", utils.RenderPage(auth.LoginPage))
}

func (c *AuthController) RegisterApiRoutes() {
	c.api.Post("/register", c.register)
	c.api.Post("/login", c.login)
	c.api.Post("/logout", c.logout)
	c.api.Post("/forgot", c.forgot)
	c.api.Put("/updatePassword", c.updatePassword)
	c.api.Put("/resetPassword", c.resetPassword)
}

type AuthFormData struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

func (c *AuthController) register(ctx *fiber.Ctx) error {
	var body AuthFormData
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	var user models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&user).Error
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("User already exists")
	}

	tx := c.db.Begin()
	if tx.Error != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error starting a transaction")
	}

	user = models.User{
		Password: body.Password,
		Email:    body.Email,
	}
	err = tx.Create(&user).Error

	if err != nil {
		tx.Rollback()
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating user")
	}

	if err := tx.Commit().Error; err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error committing the transaction")
	}

	// utils.SendVerificationEmail(body.Email, userStatus.ID.String())

	token, err := utils.CreateToken(user.ID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token" + err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "x-token",
		Value:    token,
		HTTPOnly: true,
	})

	return ctx.Redirect("/dashboard")
}

func (c *AuthController) login(ctx *fiber.Ctx) error {
	var body AuthFormData
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	var user models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&user).Error
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Email or password incorrect")
	}

	err = user.ComparePasswords(body.Password)
	if err != nil {
		logger.Warn("Error comparing passwords", err)
		logger.Warn("Body", body)
		return ctx.Status(http.StatusBadRequest).SendString("Email or password incorrect")
	}

	token, err := utils.CreateToken(user.ID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token" + err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "x-token",
		Value:    token,
		HTTPOnly: true,
	})

	return ctx.Redirect("/dashboard")
}

func (c *AuthController) logout(ctx *fiber.Ctx) error {
	ctx.Cookie(&fiber.Cookie{
		Name:     "x-token",
		HTTPOnly: true,
		MaxAge:   0,
	})
	return ctx.Status(http.StatusNoContent).Redirect("/auth/login")
}

func (c *AuthController) forgot(ctx *fiber.Ctx) error {
	type Forgot struct {
		Email string `json:"email"`
	}

	var body Forgot
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	var email models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&email).Error
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Email not found"})
	}

	// utils.SendResetEmail(body.Email, userStatus.ID.String())

	response := fiber.Map{
		"data": fiber.Map{
			"line1": "Verification email sent",
			"line2": "Please check your email at " + body.Email + " for a verification link",
		},
	}

	return ctx.Status(http.StatusCreated).JSON(response)
}

func (c *AuthController) updatePassword(ctx *fiber.Ctx) error {
	return ctx.Status(http.StatusOK).JSON(fiber.Map{"message": "Change password"})
}

func (c *AuthController) resetPassword(ctx *fiber.Ctx) error {
	type UpdatePassword struct {
		UserId   uuid.UUID `json:"userId"`
		Password string    `json:"password"`
	}

	var body UpdatePassword
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	err = c.db.Model(&models.User{}).Where("id = ?", body.UserId).Updates(models.User{Password: body.Password}).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating password"})
	}

	response := fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"line1": "Password updated",
			"line2": "Please login with your new password",
		},
	}

	return ctx.Status(http.StatusOK).JSON(response)
}
