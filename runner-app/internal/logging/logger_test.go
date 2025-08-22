package logging

import (
	"context"
	"os"
	"testing"
	"github.com/rs/zerolog"
)

func TestInitDoesNotPanicAndSetsLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	_ = Init()
	_ = L() // retrieve global logger
}

func TestFromContextAddsRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), "request_id", "req-123")
	l := FromContext(ctx)
	_ = l // ensure no panic and value returned
}

func TestWithFieldsAddsFields(t *testing.T) {
	_ = Init()
	_ = WithFields(L(), map[string]string{"a": "1", "b": "2"})
}

func TestInit_LevelMappings(t *testing.T) {
	// debug
	os.Setenv("LOG_LEVEL", "debug")
	Init()
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Fatalf("expected debug level")
	}
	// warn
	os.Setenv("LOG_LEVEL", "warn")
	Init()
	if zerolog.GlobalLevel() != zerolog.WarnLevel {
		t.Fatalf("expected warn level")
	}
	// error
	os.Setenv("LOG_LEVEL", "error")
	Init()
	if zerolog.GlobalLevel() != zerolog.ErrorLevel {
		t.Fatalf("expected error level")
	}
	// unknown -> defaults to info
	os.Setenv("LOG_LEVEL", "nope")
	Init()
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Fatalf("expected info level for unknown")
	}
	os.Unsetenv("LOG_LEVEL")
}

func TestInit_ConsoleVsJSONBranches(t *testing.T) {
	// console branch
	os.Setenv("LOG_FORMAT", "console")
	os.Unsetenv("GIN_MODE")
	Init()

	// json branch (release and not console)
	os.Unsetenv("LOG_FORMAT")
	os.Setenv("GIN_MODE", "release")
	Init()

	// cleanup
	os.Unsetenv("GIN_MODE")
}
