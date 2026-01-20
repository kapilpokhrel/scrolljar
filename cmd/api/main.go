package main

import (
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/kapilpokhrel/scrolljar/internal/api"
)

func main() {
	logger := setupLogger()
	err := godotenv.Load()
	if err != nil {
		logger.Error("Error loading .env file")
	}

	app, err := api.NewApplication(logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}
	if err = app.Serve(); err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}
}
