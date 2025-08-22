package transparency

import (
    "testing"
    "time"
)

func TestWriter_Append_SetsTimestampAndPrevHash(t *testing.T) {
    w := NewWriter()

    // First entry with zero timestamp should be filled and prev hash empty
    e1 := LogEntry{JobID: "job1"}
    proof1, root1 := w.Append(e1)
    if proof1 == nil || root1 == "" {
        t.Fatalf("expected proof and non-empty root for first append")
    }
    entries := w.Entries()
    if len(entries) != 1 {
        t.Fatalf("expected 1 entry, got %d", len(entries))
    }
    if entries[0].Timestamp.IsZero() {
        t.Fatalf("expected timestamp to be set")
    }
    if entries[0].PrevHash != "" {
        t.Fatalf("expected empty PrevHash on first entry, got %q", entries[0].PrevHash)
    }

    // Second entry should have prev hash equal to first root
    e2 := LogEntry{JobID: "job2", Timestamp: time.Unix(0, 0).UTC()} // even if provided, keep as is
    proof2, root2 := w.Append(e2)
    if proof2 == nil || root2 == "" {
        t.Fatalf("expected proof and non-empty root for second append")
    }
    entries = w.Entries()
    if len(entries) != 2 {
        t.Fatalf("expected 2 entries, got %d", len(entries))
    }
    if entries[1].PrevHash != root1 {
        t.Fatalf("expected PrevHash to equal first root, got %q want %q", entries[1].PrevHash, root1)
    }
}

func TestWriter_Broadcaster_Called(t *testing.T) {
    w := NewWriter()
    called := false
    RegisterBroadcaster(func(msg string, data interface{}) { called = true })
    defer RegisterBroadcaster(nil)

    w.Append(LogEntry{JobID: "job-br"})
    if !called {
        t.Fatalf("expected broadcaster to be called")
    }
}

func TestWriter_GetProof_InvalidIndex(t *testing.T) {
    w := NewWriter()
    w.Append(LogEntry{JobID: "a"})
    if p, ok := w.GetProof(-1); ok || p != nil {
        t.Fatalf("expected false and nil proof for negative index")
    }
    if p, ok := w.GetProof(10); ok || p != nil {
        t.Fatalf("expected false and nil proof for out-of-range index")
    }
}
