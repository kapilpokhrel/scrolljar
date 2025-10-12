// Package api defines the config, management, handlers and middleware for scrolljar api
package api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kapilpokhrel/scrolljar/internal/database"
)

type Environment string

const (
	DEV  Environment = "dev"
	PROD Environment = "prod"
)

type Config struct {
	Port int
	Env  Environment
	DB   struct {
		URL          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  time.Duration
	}
}

type Application struct {
	config Config
	logger *slog.Logger
	models database.Models
}

func NewApplication(cfg Config, logger *slog.Logger, db *sql.DB) *Application {
	return &Application{
		config: cfg,
		logger: logger,
		models: database.NewModels(db),
	}
}

func (app *Application) NewServer() *http.Server {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}
	return server
}
