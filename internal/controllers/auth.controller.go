package controllers

import (
	"journal/internal/emails"
	"journal/internal/jobs"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/auth"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	c.views.Get("/confirm-email", c.confirmEmail, utils.RenderPage(auth.ConfirmEmailPage))
}

func (c *AuthController) RegisterApiRoutes() {
	// Limits are per client, in memory only (see middleware.RateLimit).
	c.api.Post("/register", middleware.RateLimit(10, time.Hour), c.register)
	c.api.Post("/login", middleware.RateLimit(10, 15*time.Minute), c.login)
	c.api.Post("/logout", c.logout)
	c.api.Post("/forgot", middleware.RateLimit(5, time.Hour), c.forgot)
	c.api.Put("/reset", middleware.RateLimit(10, 15*time.Minute), c.resetPassword)
}

type AuthFormData struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

const sessionCookieName = "x-token"
const sessionLifetime = time.Hour * 24 * 30

// setSessionCookie signs the person in. The token carries their TokenVersion,
// so a later password change or reset ends it.
func setSessionCookie(ctx *fiber.Ctx, user *models.User) error {
	token, err := utils.CreateAccessToken(user.ID, user.TokenVersion)
	if err != nil {
		return err
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		Expires:  time.Now().Add(sessionLifetime),
		MaxAge:   int(sessionLifetime.Seconds()),
	})
	return nil
}

// clearSessionCookie must repeat the Path the cookie was set with, or the
// browser treats it as a different cookie and the session survives.
func clearSessionCookie(ctx *fiber.Ctx) {
	ctx.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   os.Getenv("APP_ENV") == "production",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func (c *AuthController) register(ctx *fiber.Ctx) error {
	var body AuthFormData
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	body.Email = strings.TrimSpace(body.Email)
	if !strings.Contains(body.Email, "@") {
		return ctx.Status(http.StatusBadRequest).SendString("Enter a valid email address")
	}
	if msg := models.ValidatePassword(body.Password); msg != "" {
		return ctx.Status(http.StatusBadRequest).SendString(msg)
	}

	var user models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&user).Error
	if err == nil {
		return ctx.Status(http.StatusBadRequest).SendString("User already exists")
	}

	tx := c.db.Begin()
	if tx.Error != nil {
		utils.SetRequestError(ctx, tx.Error)
		return ctx.Status(http.StatusInternalServerError).SendString("Error starting a transaction")
	}

	user = models.User{
		Password: body.Password,
		Email:    body.Email,
		Base: &models.Base{
			CreatorID:     &uuid.Nil,
			LastUpdaterID: &uuid.Nil,
		},
	}
	err = tx.Create(&user).Error

	if err != nil {
		tx.Rollback()
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating user")
	}

	if err := tx.Commit().Error; err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error committing the transaction")
	}

	// Best effort: signing up never depends on the email going out.
	if err := sendEmailConfirmation(c.db, &user); err != nil {
		logger.Error("Error sending email confirmation", "error", err.Error())
	}

	if err := setSessionCookie(ctx, &user); err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token")
	}

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
		return ctx.Status(http.StatusBadRequest).SendString("Email or password incorrect")
	}

	if err := setSessionCookie(ctx, &user); err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error creating token")
	}

	return ctx.Redirect("/dashboard")
}

func (c *AuthController) logout(ctx *fiber.Ctx) error {
	clearSessionCookie(ctx)
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

	// The same answer whether or not the address has an account, so this
	// can't be used to find out who is registered.
	const accepted = "If that email has an account, a reset link is on its way."

	var user models.User
	err = c.db.Where("lower(email) = ?", strings.ToLower(body.Email)).First(&user).Error
	if err != nil {
		return ctx.Status(http.StatusAccepted).SendString(accepted)
	}

	resetToken, err := utils.CreateResetToken(user.ID, user.TokenVersion)
	if err != nil {
		utils.SetRequestError(ctx, err)
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
	_, err = jobs.ScheduleEmailJob(user.ID, data, time.Now())
	if err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).SendString("Error sending reset email")
	}

	return ctx.Status(http.StatusAccepted).SendString(accepted)
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

	userID, version, err := utils.ValidateResetToken(body.Token)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("This reset link isn't valid or has expired. Ask for a new one.")
	}

	if body.Password != body.ConfirmPassword {
		return ctx.Status(http.StatusBadRequest).SendString("Passwords do not match")
	}
	if msg := models.ValidatePassword(body.Password); msg != "" {
		return ctx.Status(http.StatusBadRequest).SendString(msg)
	}

	// The version in the WHERE makes the link work once, even for two
	// simultaneous requests; the bump also ends every existing session.
	res := c.db.Model(&models.User{}).
		Where("id = ? AND token_version = ?", userID, version).
		Updates(&models.User{Password: body.Password, TokenVersion: version + 1})
	if res.Error != nil {
		utils.SetRequestError(ctx, res.Error)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating password"})
	}
	if res.RowsAffected == 0 {
		return ctx.Status(http.StatusBadRequest).SendString("This reset link isn't valid or has expired. Ask for a new one.")
	}

	return ctx.Status(http.StatusOK).SendString("Password updated, <a href='/auth/login'>login</a> to continue.")
}
