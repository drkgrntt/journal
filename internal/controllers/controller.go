package controllers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Controller interface {
	Init(db *gorm.DB, app *fiber.App)
	RegisterViewRoutes()
	RegisterApiRoutes()
}
