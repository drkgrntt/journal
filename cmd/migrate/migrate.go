package main

import (
	"journal/internal/database"
	"journal/internal/logger"
)

func main() {
	database.New()
	database.RunMigrations()
	logger.Info("Ran pending SQL migrations successfully")
}
