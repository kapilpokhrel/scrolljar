package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

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
	flag.IntVar(&cfg.DB.MaxIdleConns, "db_max-idle-conns", 50, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.DB.MaxIdleTime, "db_max-idle-time", time.Minute*10, "PostgreSQL max idle time")
	flag.Parse()

	return cfg
}

func main() {
	logger := setupLogger()
	err := godotenv.Load()
	if err != nil {
		logger.Error("Error loading .env file")
	}
	cfg := parseFlags()

	logger.Info(fmt.Sprintf("Connecting to database at %s", cfg.DB.URL))
	db, err := sql.Open("pgx", cfg.DB.URL)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}
	defer db.Close()

	app := api.NewApplication(cfg, logger, db)
	server := app.NewServer()

	logger.Info("Starting scrolljar API server", "addr", server.Addr, "env", cfg.Env)
	err = server.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(-1)
}
