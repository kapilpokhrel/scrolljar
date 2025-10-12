package main

import (
	"flag"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kapilpokhrel/scrolljar/internal/api"
)

func parseFlags() api.Config {
	var cfg api.Config
	flag.IntVar(&cfg.Port, "port", 8008, "API server port")
	flag.StringVar((*string)(&cfg.Env), "env", "dev", "Environment (dev|pord))")
	flag.StringVar(&cfg.DB.URL, "db_url", os.Getenv("SCROLLJAR_DB_URL"), "PostgreSQL URL")
	flag.IntVar(&cfg.DB.MaxOpenConns, "db_max-open-conns", 50, "PostgreSQL URL")
	flag.IntVar(&cfg.DB.MaxIdleConns, "db_max-idle-conns", 50, "PostgreSQL URL")
	flag.DurationVar(&cfg.DB.MaxIdleTime, "db_max-idle-time", time.Minute*10, "PostgreSQL URL")
	flag.Parse()

	return cfg
}

func main() {
	logger := setupLogger()
	cfg := parseFlags()

	app := api.NewApplication(cfg, logger)

	server := app.NewServer()
	err := app.OpenDB()
	if err != nil {
		logger.Error(err.Error())
		app.Close()
		os.Exit(-1)
	}

	logger.Info("Starting scrolljar API server", "addr", server.Addr, "env", cfg.Env)
	err = server.ListenAndServe()
	logger.Error(err.Error())
	app.Close()
	os.Exit(-1)
}
