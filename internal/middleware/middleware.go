package middleware

import (
	"journal/internal/database"

	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func init() {
	db = database.New().DB
}
