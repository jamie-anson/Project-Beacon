package logging

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures zerolog for the app
func Init() zerolog.Logger {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info", "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Human-friendly console output in dev; JSON otherwise
	if os.Getenv("LOG_FORMAT") == "console" || os.Getenv("GIN_MODE") != "release" {
		console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log.Logger = zerolog.New(console).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
	return log.Logger
}

// L returns the global logger
func L() zerolog.Logger { return log.Logger }
