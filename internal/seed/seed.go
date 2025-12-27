package main

import (
	"journal/internal/database"
	"journal/internal/logger"
	"journal/internal/seed/seeders"
)

func init() {
	database.New()
}

func main() {
	database.DropTables()
	logger.Info("? Dropped Tables Successfully")

	database.AutoMigrate()
	logger.Info("? AutoMigrated Successfully")

	seeders.Seed()
	logger.Info("? Seeded Successfully")
}
