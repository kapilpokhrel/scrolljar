// Package api defines the config, management, handlers and middleware for scrolljar api
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
		MinIdleConns int
		MaxIdleTime  time.Duration
	}
}

type Application struct {
	config Config
	logger *slog.Logger
	models database.Models
}

func NewApplication(cfg Config, logger *slog.Logger, dbPool *pgxpool.Pool) *Application {
	return &Application{
		config: cfg,
		logger: logger,
		models: database.NewModels(dbPool),
	}
}

func (app *Application) Serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutDownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		app.logger.Info("shutting down server", "signal", (<-quit).String())
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		shutDownError <- server.Shutdown(ctx)
	}()

	app.logger.Info("Starting scrolljar API server", "addr", server.Addr, "env", app.config.Env)
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutDownError
	if err != nil {
		return err
	}

	app.logger.Info("server stopped")
	return nil
}
