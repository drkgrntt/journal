package utils

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func RenderPage(component func(*fiber.Ctx) templ.Component) func(*fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		return adaptor.HTTPHandler(templ.Handler(component(ctx)))(ctx)
	}
}

func RenderComponent(component templ.Component, ctx *fiber.Ctx) (err error) {
	buf := new(bytes.Buffer)
	err = component.Render(ctx.Context(), buf)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}

	err = ctx.Status(http.StatusOK).SendString(buf.String())
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}

	return
}

func RenderComponents(components []templ.Component, ctx *fiber.Ctx) (err error) {
	buf := new(bytes.Buffer)
	for _, component := range components {
		err = component.Render(ctx.Context(), buf)
		if err != nil {
			return ctx.Status(http.StatusBadRequest).SendString(err.Error())
		}
	}

	err = ctx.Status(http.StatusOK).SendString(buf.String())
	if err != nil {
		return ctx.Status(http.StatusBadRequest).SendString(err.Error())
	}

	return
}
