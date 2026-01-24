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
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	Rate struct {
		GlobalRps float64
		GlobalBps int
		IPRps     float64
		IPBps     int
	}
	S3 struct {
		BucketName string
	}
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
	s3Client  *s3.Client
}

func parseFlags() Config {
	var cfg Config
	flag.IntVar(&cfg.Port, "port", 8008, "API server port")
	flag.StringVar((*string)(&cfg.Env), "env", "dev", "Environment (dev|pord))")
	flag.StringVar(&cfg.DB.URL, "db_url", os.Getenv("SCROLLJAR_DB_URL"), "PostgreSQL URL")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db_max-open-conns", 50, "PostgreSQL max open connections")
	flag.IntVar(&cfg.DB.MinIdleConns, "db_min-idle-conns", 50, "PostgreSQL min idle connections")
	flag.DurationVar(&cfg.DB.MaxIdleTime, "db_max-idle-time", time.Minute*10, "PostgreSQL max idle time")

	flag.StringVar(&cfg.SMTP.Host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")

	smtpPort, _ := strconv.ParseInt(os.Getenv("SMTP_PORT"), 10, 64)
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", int(smtpPort), "SMTP port")
	flag.StringVar(&cfg.SMTP.Username, "smt-username", os.Getenv("SMTP_USERNAME"), "SMPT Username")
	flag.StringVar(&cfg.SMTP.Password, "smt-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP sender")

	flag.Float64Var(&cfg.Rate.GlobalRps, "global-rate-limit", 200.0, "Global rate limit (per second)")
	flag.IntVar(&cfg.Rate.GlobalBps, "global-burst", 250, "Global limit burst (per second)")
	flag.Float64Var(&cfg.Rate.IPRps, "ip-rate-limit", 10.0, "IP rate limit (per second)")
	flag.IntVar(&cfg.Rate.IPBps, "ip-burst", 15, "IP limit burst (per second)")

	flag.StringVar(&cfg.S3.BucketName, "s3-bucket", os.Getenv("S3_BUCKET"), "s3 bucket")
	flag.Parse()

	return cfg
}

func setupDB(cfg Config) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		return nil, err
	}
	config.MaxConnIdleTime = cfg.DB.MaxIdleTime
	config.MaxConns = int32(cfg.DB.MaxOpenConns)
	config.MinIdleConns = int32(cfg.DB.MinIdleConns)
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewApplication(logger *slog.Logger) (*Application, error) {
	cfg := parseFlags()
	logger.Info(fmt.Sprintf("Connecting to database at %s", cfg.DB.URL))
	dbPool, err := setupDB(cfg)
	if err != nil {
		return nil, err
	}

	// S3 client
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsCfg)

	app := &Application{
		config:    cfg,
		dbPool:    dbPool,
		logger:    logger,
		models:    database.NewModels(dbPool),
		mailer:    mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender),
		startTime: time.Now(),
		s3Client:  s3Client,
	}
	app.ipLimiter = NewRouteIPLimiter(app.ipRateLimiter)
	return app, nil
}

func (app *Application) Serve() error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.GetRouter(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Minute,
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
	app.dbPool.Close()
	return nil
}
