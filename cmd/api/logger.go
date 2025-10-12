package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/kapilpokhrel/scrolljar/pkg/handlermux"
)

func setupLogger() *slog.Logger {
	stdHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelWarn})

	homeDir, _ := os.UserHomeDir()
	rotFileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(homeDir, ".local/share/kdfs/logs/kdfs.log"),
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	}
	rotFileHandler := slog.NewTextHandler(rotFileWriter, &slog.HandlerOptions{Level: slog.LevelInfo})

	handlerMux := handlermux.New(stdHandler, rotFileHandler)
	logger := slog.New(handlerMux)
	return logger
}
