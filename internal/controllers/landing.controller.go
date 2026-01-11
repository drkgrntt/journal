package controllers

import (
	"journal/internal/web/landings"
	"journal/internal/emails"
	"journal/internal/jobs"
	"journal/internal/utils"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"
)

func init() {
	registerController(&LandingController{})
}

type LandingController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *LandingController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("/")
	c.api = app.Group("api")
}

func (c *LandingController) RegisterViewRoutes() {
	c.views.Get("/about", utils.RenderPage(landings.AboutPage))
	c.views.Get("/privacy", utils.RenderPage(landings.PrivacyPage))
	c.views.Get("/feedback", utils.RenderPage(landings.FeedbackPage))
}

func (c *LandingController) RegisterApiRoutes() {
	c.api.Post("/feedback", c.sendFeedback)
}

type Feedback struct {
	Email   string `form:"email"`
	Message string `form:"message"`
}

func (c *LandingController) sendFeedback(ctx *fiber.Ctx) error {
	var body Feedback
	err := ctx.BodyParser(&body)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString("Bad request reading body")
	}

	data := &jobs.EmailData{
		Name: emails.FEEDBACK,
		Recipients: []*emails.EmailRecipient{
			{
				Email: os.Getenv("OWNER_EMAIL"),
				Name:  "Journal Feedback",
			},
		},
		Variables: &emails.FeedbackVariables{
			Email:   body.Email,
			Message: body.Message,
		},
	}
	jobs.ScheduleEmailJob(uuid.Nil, data, time.Now())

	return ctx.Status(http.StatusAccepted).SendString("Thank you!")
}
