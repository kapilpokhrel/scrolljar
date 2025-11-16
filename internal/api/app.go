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
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kapilpokhrel/scrolljar/internal/database"
	"github.com/kapilpokhrel/scrolljar/internal/mailer"
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
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		Sender   string
	}
}

type Application struct {
	config    Config
	logger    *slog.Logger
	models    database.Models
	mailer    mailer.Mailer
	wg        sync.WaitGroup
	startTime time.Time
}

func NewApplication(cfg Config, logger *slog.Logger, dbPool *pgxpool.Pool) *Application {
	return &Application{
		config:    cfg,
		logger:    logger,
		models:    database.NewModels(dbPool),
		mailer:    mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender),
		startTime: time.Now(),
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

		app.wg.Wait()
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
