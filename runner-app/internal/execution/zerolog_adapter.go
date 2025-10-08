package execution

import (
	"github.com/rs/zerolog"
)

// ZerologAdapter adapts zerolog.Logger to execution.Logger interface
type ZerologAdapter struct {
	logger *zerolog.Logger
}

// NewZerologAdapter creates a new zerolog adapter
func NewZerologAdapter(logger *zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// Info logs an info message
func (z *ZerologAdapter) Info(msg string, keysAndValues ...interface{}) {
	event := z.logger.Info()
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keysAndValues[i+1])
	}
	event.Msg(msg)
}

// Warn logs a warning message
func (z *ZerologAdapter) Warn(msg string, keysAndValues ...interface{}) {
	event := z.logger.Warn()
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keysAndValues[i+1])
	}
	event.Msg(msg)
}

// Debug logs a debug message
func (z *ZerologAdapter) Debug(msg string, keysAndValues ...interface{}) {
	event := z.logger.Debug()
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keysAndValues[i+1])
	}
	event.Msg(msg)
}

// Error logs an error message
func (z *ZerologAdapter) Error(msg string, keysAndValues ...interface{}) {
	event := z.logger.Error()
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keysAndValues[i+1])
	}
	event.Msg(msg)
}
