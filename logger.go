package main

import (
	"log/slog"
	"os"

	"github.com/default23/protofake/config"
)

// NewLogger constructs the logger with the provided log level and format.
func NewLogger(conf config.Logger) *slog.Logger {
	if conf.Level == "" {
		conf.Level = "info"
	}

	opts := &slog.HandlerOptions{Level: conf.Level}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if conf.JSONFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
