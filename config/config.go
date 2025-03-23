package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v6"
)

// LogLevel is the type, needed to implement the slog.Leveler interface.
type LogLevel string

// Config is the application configuration.
type Config struct {
	// WatchMappingsChanges is the option to watch for changes in the mappings directory.
	// If true, the application will watch for changes in the mappings directory and reload the mappings.
	WatchMappingsChanges bool     `env:"WATCH_MAPPINGS_CHANGES" envDefault:"false"`
	DataDir              string   `env:"DATA_DIR" envDefault:"/data"`
	DescriptorExtensions []string `env:"DESCRIPTOR_EXTENSIONS" envDefault:".pb"`

	GRPC   GRPC   `envPrefix:"GRPC_"`
	Logger Logger `envPrefix:"LOG_"`
}

// Logger is the logger configuration.
type Logger struct {
	Level      LogLevel `env:"LEVEL" envDefault:"info"`
	JSONFormat bool     `env:"JSON_FORMAT" envDefault:"true"`
}

// GRPC is the gRPC server configuration.
type GRPC struct {
	Host                   string `env:"HOST" envDefault:"0.0.0.0"`
	Port                   string `env:"PORT" envDefault:"5675"`
	ServerReflection       bool   `env:"SERVER_REFLECTION" envDefault:"false"`
	IgnoreDuplicateService bool   `env:"IGNORE_DUPLICATE_SERVICE" envDefault:"false"`
	DiscardUnknownFields   bool   `env:"DISCARD_UNKNOWN_FIELDS" envDefault:"false"`
}

// Parse returns configuration, parsed from Environment variables.
func Parse() (*Config, error) {
	conf := new(Config)
	if err := env.Parse(conf); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	exts := []string{}
	for _, ext := range conf.DescriptorExtensions {
		exts = append(exts, strings.TrimSpace(ext))
	}
	if len(exts) == 0 {
		return nil, fmt.Errorf("parse config: no descriptor extensions provided")
	}

	conf.DescriptorExtensions = exts
	return conf, nil
}

// Level is the implementation of slog.Leveler.
func (l LogLevel) Level() slog.Level {
	switch l {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
