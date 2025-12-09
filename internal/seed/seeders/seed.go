package seeders

import (
	"go-starter/internal/database"
	"go-starter/internal/models"

	"gorm.io/gorm"
)

var (
	db    *gorm.DB
	admin *models.User
)

func Seed() {
	db = database.New().DB
	seedUsers()
}
