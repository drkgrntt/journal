package journal

import (
	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func getJournal(c *fiber.Ctx) *models.Journal {
	journal := utils.GetLocal[models.Journal](c, "journal")
	return journal
}
