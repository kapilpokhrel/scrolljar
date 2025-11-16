package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/kapilpokhrel/scrolljar/internal/api"
)

func parseFlags() api.Config {
	var cfg api.Config
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
	flag.Parse()

	return cfg
}

func setupDB(cfg api.Config) (*pgxpool.Pool, error) {
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

func main() {
	logger := setupLogger()
	err := godotenv.Load()
	if err != nil {
		logger.Error("Error loading .env file")
	}
	cfg := parseFlags()

	logger.Info(fmt.Sprintf("Connecting to database at %s", cfg.DB.URL))
	dbPool, err := setupDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}
	defer dbPool.Close()

	app := api.NewApplication(cfg, logger, dbPool)
	if err = app.Serve(); err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}
}
