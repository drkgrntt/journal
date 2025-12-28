package controllers

import (
	"journal/cmd/web/auth"
	"journal/internal/emails"
	"journal/internal/jobs"
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
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
	c.views.Get("/forgot", utils.RenderPage(auth.ForgotPage))
	c.views.Get("/reset", utils.RenderPage(auth.ResetPage))
}

func (c *AuthController) RegisterApiRoutes() {
	c.api.Post("/register", c.register)
	c.api.Post("/login", c.login)
	c.api.Post("/logout", c.logout)
	c.api.Post("/forgot", c.forgot)
	c.api.Put("/reset", c.resetPassword)
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
	if err == nil {
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

	token, err := utils.CreateAccessToken(user.ID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token" + err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "x-token",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 24 * 30),
		MaxAge:   60 * 60 * 24 * 30,
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

	token, err := utils.CreateAccessToken(user.ID)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token" + err.Error())
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "x-token",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
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
		Email string `form:"email"`
	}

	var body Forgot
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	var user models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&user).Error
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Email not found")
	}

	claims := map[string]any{
		"sub": user.ID,
	}
	resetToken, err := utils.CreateToken(time.Minute*5, claims)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating reset token")
	}

	data := &jobs.EmailData{
		Name: emails.FORGOT_PASSWORD,
		Recipients: []*emails.EmailRecipient{
			{
				Email: user.Email,
				Name:  user.FullName(),
			},
		},
		Variables: &emails.ForgotPasswordVariables{
			ResetToken: resetToken,
			RootUrl:    os.Getenv("ROOT_URL"),
		},
	}
	jobs.ScheduleEmailJob(user.ID, data, time.Now())

	return ctx.Status(http.StatusAccepted).SendString("Check your email to reset your password.")
}

func (c *AuthController) resetPassword(ctx *fiber.Ctx) error {
	type UpdatePassword struct {
		Token           string `form:"token"`
		Password        string `form:"password"`
		ConfirmPassword string `form:"confirmPassword"`
	}

	var body UpdatePassword
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	tokenData, err := utils.ValidateToken(body.Token)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}

	if body.Password != body.ConfirmPassword {
		return ctx.Status(http.StatusBadRequest).SendString("Passwords do not match")
	}

	err = c.db.Model(&models.User{}).Where("id = ?", tokenData.UserID).Updates(&models.User{Password: body.Password}).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating password"})
	}

	return ctx.Status(http.StatusOK).SendString("Password updated, <a href='/auth/login'>login</a> to continue.")
}
