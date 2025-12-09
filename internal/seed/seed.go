package main

import (
	"go-starter/internal/database"
	"go-starter/internal/logger"
	"go-starter/internal/seed/seeders"
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
