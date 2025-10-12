// Package api defines the config, management, handlers and middleware for scrolljar api
package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Environment string

const (
	DEV  Environment = "dev"
	PROD Environment = "prod"
)

type Config struct {
	Port int
	Env  Environment
}

type Application struct {
	config Config
	logger *slog.Logger
}

func NewApplication(cfg Config, logger *slog.Logger) *Application {
	return &Application{
		config: cfg,
		logger: logger,
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
