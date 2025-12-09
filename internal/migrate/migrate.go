package main

import (
	"go-starter/internal/database"
	"go-starter/internal/logger"
)

func main() {
	database.New()
	database.AutoMigrate()
	logger.Info("? AutoMigrated Successfully")
}
