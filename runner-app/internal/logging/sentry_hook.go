package logging

import (
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// SentryHook is a zerolog hook that sends ALL logs to Sentry
type SentryHook struct{}

// Run implements zerolog.Hook interface
func (h SentryHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Send all log levels to Sentry (Info, Warn, Error, Fatal, Panic)
	// Skip Debug and Trace to avoid noise
	if level < zerolog.InfoLevel {
		return
	}

	// Convert zerolog level to Sentry level
	var sentryLevel sentry.Level
	switch level {
	case zerolog.InfoLevel:
		sentryLevel = sentry.LevelInfo
	case zerolog.WarnLevel:
		sentryLevel = sentry.LevelWarning
	case zerolog.ErrorLevel:
		sentryLevel = sentry.LevelError
	case zerolog.FatalLevel:
		sentryLevel = sentry.LevelFatal
	case zerolog.PanicLevel:
		sentryLevel = sentry.LevelFatal
	default:
		sentryLevel = sentry.LevelInfo
	}

	// Capture the message in Sentry
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentryLevel)
		
		// Add context tags
		scope.SetTag("log_level", level.String())
		
		sentry.CaptureMessage(msg)
	})
}
