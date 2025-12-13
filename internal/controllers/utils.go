package controllers

import (
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"sync"

	"github.com/gofiber/fiber/v2"
)

var (
	controllersMux sync.Mutex
	controllers    []Controller
)

func registerController(controller Controller) {
	controllersMux.Lock()
	defer controllersMux.Unlock()
	controllers = append(controllers, controller)
}

func GetControllers() []Controller {
	return controllers
}

func GetJournalTypes(ctx *fiber.Ctx) []*models.JournalType {
	types := utils.GetLocal[[]*models.JournalType](ctx, "journalTypes")
	return *types
}

func GetRatings(ctx *fiber.Ctx) []*models.Rating {
	ratings := utils.GetLocal[[]*models.Rating](ctx, "ratings")
	return *ratings
}
