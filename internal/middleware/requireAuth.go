package middleware

import (
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// SessionEndedMessage is shown to the user when a background request arrives
// without a session.
const SessionEndedMessage = "Your session has ended. Reload the page to log in again."

// RequireAuth sends a signed-out page load to the login page. Background
// requests (htmx, or any write) get a 401 instead: the browser follows a
// redirect on its own and ends on a 200 login page, which reads as success and
// makes the client delete its on-device draft.
func RequireAuth(ctx *fiber.Ctx) error {
	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if user != nil {
		return ctx.Next()
	}
	logger.Warn("No current user found")

	method := ctx.Method()
	isRead := method == fiber.MethodGet || method == fiber.MethodHead
	if ctx.Get("HX-Request") == "true" || !isRead {
		return ctx.Status(http.StatusUnauthorized).SendString(SessionEndedMessage)
	}
	return ctx.Redirect("/auth/login")
}
