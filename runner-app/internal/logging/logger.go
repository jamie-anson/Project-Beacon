package logging

import (
	"context"
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
	
	// Add Sentry hook if SENTRY_DSN is set
	if os.Getenv("SENTRY_DSN") != "" {
		log.Logger = log.Logger.Hook(SentryHook{})
	}
	
	return log.Logger
}

// L returns the global logger
func L() zerolog.Logger { return log.Logger }

// FromContext returns a logger enriched with correlation fields from context
// Currently attaches request_id if present in context under key "request_id".
func FromContext(ctx context.Context) zerolog.Logger {
	l := log.Logger
	if ctx == nil {
		return l
	}
	if v := ctx.Value("request_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			l = l.With().Str("request_id", s).Logger()
		}
	}
	return l
}

// WithFields attaches arbitrary string fields to the logger and returns it
func WithFields(l zerolog.Logger, fields map[string]string) zerolog.Logger {
	if len(fields) == 0 {
		return l
	}
	ctx := l.With()
	for k, v := range fields {
		ctx = ctx.Str(k, v)
	}
	return ctx.Logger()
}

// DebugEnabled returns true when extra debug instrumentation should run.
// Enabled if either DEBUG=true or LOG_LEVEL=debug.
func DebugEnabled() bool {
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		return true
	}
	return strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug")
}
