// Package api defines the config, management, handlers and middleware for scrolljar api
package api

import (
	"context"
	"errors"
	"flag"
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
	DB   database.DBCFG
	SMTP mailer.MailerCFG
	Rate struct {
		GlobalRps float64
		GlobalBps int
		IPRps     float64
		IPBps     int
	}
	S3 database.S3CFG
}

type Application struct {
	config    Config
	dbPool    *pgxpool.Pool
	logger    *slog.Logger
	models    database.Models
	mailer    mailer.Mailer
	wg        sync.WaitGroup
	startTime time.Time
	ipLimiter routeIPLimiter
	s3Bucket  *database.S3Bucket
}

func parseFlags() Config {
	var cfg Config
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&cfg.Port, "port", 8008, "API server port")
	fs.StringVar((*string)(&cfg.Env), "env", "dev", "Environment (dev|pord))")

	cfg.DB.RegisterFlags(fs)
	cfg.SMTP.RegisterFlags(fs)

	fs.Float64Var(&cfg.Rate.GlobalRps, "global-rate-limit", 200.0, "Global rate limit (per second)")
	fs.IntVar(&cfg.Rate.GlobalBps, "global-burst", 250, "Global limit burst (per second)")
	fs.Float64Var(&cfg.Rate.IPRps, "ip-rate-limit", 10.0, "IP rate limit (per second)")
	fs.IntVar(&cfg.Rate.IPBps, "ip-burst", 15, "IP limit burst (per second)")

	fs.StringVar(&cfg.S3.BucketName, "s3-bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	fs.Parse(os.Args[1:])

	return cfg
}

func NewApplication(logger *slog.Logger) (*Application, error) {
	cfg := parseFlags()
	logger.Info(fmt.Sprintf("Connecting to database at %s", cfg.DB.URL))
	dbPool, err := database.SetupDB(cfg.DB)
	if err != nil {
		return nil, err
	}

	s3Bucket, err := database.NewS3Bucket(cfg.S3)
	if err != nil {
		return nil, err
	}

	app := &Application{
		config:    cfg,
		dbPool:    dbPool,
		logger:    logger,
		models:    database.NewModels(dbPool),
		mailer:    mailer.New(cfg.SMTP),
		startTime: time.Now(),
		s3Bucket:  s3Bucket,
	}
	app.ipLimiter = NewRouteIPLimiter(app.ipRateLimiter)
	return app, nil
}

func (app *Application) Serve() error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.Port),
		Handler:           app.GetRouter(),
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      3 * time.Minute, // Large timeout is needed for upload/. We can have more granular control with TimeoutHandler
		ErrorLog:          slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
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

	if err := <-shutDownError; err != nil {
		return err
	}

	app.logger.Info("server stopped")
	app.dbPool.Close()
	return nil
}
