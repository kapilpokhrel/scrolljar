// Package logger handles logging
package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
)

func SetupLogger(name string) *slog.Logger {
	stdHandler := tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelWarn})

	homeDir, _ := os.UserHomeDir()
	rotFileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(homeDir, fmt.Sprintf(".local/share/scrolljar/logs/%s.log", name)),
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	}
	rotFileHandler := slog.NewTextHandler(rotFileWriter, &slog.HandlerOptions{Level: slog.LevelInfo})

	handlerMux := NewHandlerMux(stdHandler, rotFileHandler)
	logger := slog.New(handlerMux)
	return logger
}
