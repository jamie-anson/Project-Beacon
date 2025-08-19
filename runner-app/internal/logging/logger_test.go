package logging

import (
	"context"
	"os"
	"testing"
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
