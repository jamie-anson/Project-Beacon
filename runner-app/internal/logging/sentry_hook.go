package logging

import (
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

// SentryHook is a zerolog hook that sends Error and Fatal logs to Sentry
type SentryHook struct{}

// Run implements zerolog.Hook interface
func (h SentryHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// Only send Error, Fatal, and Panic level logs to Sentry
	if level < zerolog.ErrorLevel {
		return
	}

	// Convert zerolog level to Sentry level
	var sentryLevel sentry.Level
	switch level {
	case zerolog.ErrorLevel:
		sentryLevel = sentry.LevelError
	case zerolog.FatalLevel:
		sentryLevel = sentry.LevelFatal
	case zerolog.PanicLevel:
		sentryLevel = sentry.LevelFatal
	default:
		sentryLevel = sentry.LevelError
	}

	// Capture the message in Sentry
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentryLevel)
		
		// Add any context from the log event
		// Note: zerolog doesn't expose fields easily, so we just send the message
		// For more context, use sentry.CaptureException() directly in code
		
		sentry.CaptureMessage(msg)
	})
}
