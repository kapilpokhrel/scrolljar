package main

import (
	"flag"
	"os"

	"github.com/kapilpokhrel/scrolljar/internal/api"
)

func parseFlags() api.Config {
	var cfg api.Config
	flag.IntVar(&cfg.Port, "port", 8008, "API server port")
	flag.StringVar((*string)(&cfg.Env), "env", "dev", "Environment (dev|pord))")
	flag.Parse()

	return cfg
}

func main() {
	logger := setupLogger()
	cfg := parseFlags()

	app := api.NewApplication(cfg, logger)
	
	server := app.NewServer()

	logger.Info("Starting scrolljar API server", "addr", server.Addr, "env", cfg.Env)
	err := server.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(-1)
}
