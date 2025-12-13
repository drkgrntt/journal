package journal

import (
	"go-starter/internal/models"
	"go-starter/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func getJournal(c *fiber.Ctx) *models.Journal {
	journal := utils.GetLocal[models.Journal](c, "journal")
	return journal
}
