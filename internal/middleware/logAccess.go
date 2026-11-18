package middleware

import (
	"errors"
	"journal/internal/logger"
	"journal/internal/models"
	"journal/internal/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// LogAccess records an anonymous line per request: when, method, host, path,
// response status, and the random per-request reference (see RequestID). It never records who made the request (no IP address,
// user agent, query string, or account ID), so the log can help find errors
// without being a history of anyone's activity. Rows are deleted after
// models.RequestLogRetention.
func LogAccess(ctx *fiber.Ctx) error {
	err := ctx.Next()

	status := ctx.Response().StatusCode()
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		status = fiberErr.Code
	}

	requestID := ""
	if id := utils.GetLocal[string](ctx, utils.LocalRequestID); id != nil {
		requestID = *id
	}

	if status >= 500 {
		// The reference ties this line to what the person saw. The cause is a
		// sanitized description, never raw error text or anything they wrote.
		cause := ""
		if c := utils.GetLocal[string](ctx, utils.LocalRequestError); c != nil {
			cause = *c
		} else if err != nil {
			cause = utils.ErrorCause(err)
		}
		logger.Error("request failed",
			"request_id", requestID, "method", ctx.Method(), "path", ctx.Path(), "status", status, "cause", cause)
	}

	DB().Create(newAnalytic(ctx.Method(), ctx.Hostname(), ctx.Path(), status, requestID, time.Now().UTC()))
	return err
}

func newAnalytic(method, domain, path string, status int, requestID string, now time.Time) *models.Analytic {
	return &models.Analytic{
		Base: &models.Base{
			CreatedAt:     now,
			UpdatedAt:     now,
			CreatorID:     &uuid.Nil,
			LastUpdaterID: &uuid.Nil,
		},
		Method:    method,
		Domain:    domain,
		Page:      path,
		Status:    status,
		RequestID: requestID,
	}
}
