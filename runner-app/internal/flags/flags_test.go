package flags

import (
	"testing"
)

func TestSetAndGet(t *testing.T) {
	// Capture original and restore after
	orig := Get()
	t.Cleanup(func() { Set(orig) })

	f := Flags{EnableCache: false, EnableWebSockets: true, ReadOnlyMode: true}
	Set(f)
	got := Get()
	if got != f {
		t.Fatalf("Get after Set mismatch: got %+v want %+v", got, f)
	}
}

func TestUpdateFromJSON_MergeKnownKeys(t *testing.T) {
	orig := Get()
	t.Cleanup(func() { Set(orig) })

	Set(Flags{EnableCache: false, EnableWebSockets: false, ReadOnlyMode: false})
	payload := []byte(`{"enable_cache": true, "enable_websockets": true, "read_only_mode": true, "unknown": 123}`)
	if err := UpdateFromJSON(payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := Get()
	if !got.EnableCache || !got.EnableWebSockets || !got.ReadOnlyMode {
		t.Fatalf("merge failed, got %+v", got)
	}
}

func TestUpdateFromJSON_BadJSON(t *testing.T) {
	orig := Get()
	t.Cleanup(func() { Set(orig) })

	err := UpdateFromJSON([]byte("{"))
	if err == nil {
		t.Fatalf("expected error for malformed JSON")
	}
}
