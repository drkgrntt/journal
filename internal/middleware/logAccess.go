package middleware

import (
	"journal/internal/models"
	"journal/internal/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

func LogAccess(ctx *fiber.Ctx) error {
	ip := ctx.IP()
	userAgent := ctx.Context().UserAgent()

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	now := time.Now().UTC()

	base := models.Base{
		CreatedAt: now,
		UpdatedAt: now,
	}

	if currentUser != nil {
		base.CreatorID = &currentUser.ID
		base.LastUpdaterID = &currentUser.ID
	}

	analytic := models.Analytic{
		Base:      &base,
		Page:      string(ctx.Request().URI().Path()),
		Query:     string(ctx.Request().URI().QueryString()),
		Useragent: string(userAgent),
		IP:        ip,
		Domain:    ctx.Hostname(),
	}

	db.Create(&analytic)

	return ctx.Next()
}
