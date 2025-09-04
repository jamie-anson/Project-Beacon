package transparency

import (
    "crypto/sha256"
    "encoding/hex"
    "testing"
    "time"

    "github.com/jamie-anson/project-beacon-runner/pkg/merkle"
)

func TestComputeEntryLeafHash_Deterministic(t *testing.T) {
    ts := time.Date(2025, 1, 2, 3, 4, 5, 6, time.UTC)
    e1 := LogEntry{
        LogIndex:    0,
        ExecutionID: 123,
        JobID:       "job-A",
        Region:      "us-east-1",
        ProviderID:  "prov-1",
        Status:      "ok",
        OutputHash:  "out-abc",
        ReceiptHash: "rcpt-xyz",
        IPFSCID:     "bafy...",
        PrevHash:    "",
        Timestamp:   ts,
    }
    h1 := ComputeEntryLeafHash(e1)
    h2 := ComputeEntryLeafHash(e1)
    if h1 != h2 {
        t.Fatalf("expected deterministic hash, got %s vs %s", h1, h2)
    }

    // Change one field -> hash should change
    e2 := e1
    e2.Status = "fail"
    h3 := ComputeEntryLeafHash(e2)
    if h1 == h3 {
        t.Fatalf("expected different hash when entry changes")
    }
}

func TestVerifyProof_FalseForTamperedProof(t *testing.T) {
    w := NewWriter()
    fixed := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

    // Build a small tree
    _, _ = w.Append(LogEntry{JobID: "a", Region: "r1", ProviderID: "p", Status: "ok", Timestamp: fixed})
    p, _ := w.Append(LogEntry{JobID: "b", Region: "r1", ProviderID: "p", Status: "ok", Timestamp: fixed.Add(time.Second)})

    if !merkle.VerifyProof(p) {
        t.Fatalf("baseline proof should verify")
    }

    // Tamper leaf hash
    bad := *p
    bad.LeafHash = doubleHash(p.LeafHash) // change it deterministically
    if merkle.VerifyProof(&bad) {
        t.Fatalf("tampered leaf hash should fail verification")
    }

    // Tamper siblings
    bad2 := *p
    if len(bad2.Siblings) > 0 {
        bad2.Siblings[0] = doubleHash(bad2.Siblings[0])
        if merkle.VerifyProof(&bad2) {
            t.Fatalf("tampered sibling should fail verification")
        }
    }
}

func TestProofSerialization_RoundTrip(t *testing.T) {
    w := NewWriter()
    fixed := time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC)

    p, _ := w.Append(LogEntry{JobID: "x", Region: "r", ProviderID: "p", Status: "ok", Timestamp: fixed})
    if p == nil { t.Fatalf("expected proof") }

    // Serialize
    data, err := p.SerializeProof()
    if err != nil { t.Fatalf("serialize: %v", err) }

    // Deserialize
    p2, err := merkle.DeserializeProof(data)
    if err != nil { t.Fatalf("deserialize: %v", err) }

    // Compare core fields
    if p2.LeafHash != p.LeafHash || p2.RootHash != p.RootHash || p2.LeafIndex != p.LeafIndex {
        t.Fatalf("round-trip mismatch")
    }
    if len(p2.Siblings) != len(p.Siblings) || len(p2.Directions) != len(p.Directions) {
        t.Fatalf("round-trip arrays mismatch")
    }
    for i := range p.Siblings {
        if p.Siblings[i] != p2.Siblings[i] || p.Directions[i] != p2.Directions[i] {
            t.Fatalf("round-trip element mismatch at %d", i)
        }
    }

    if !merkle.VerifyProof(p2) {
        t.Fatalf("round-tripped proof should still verify")
    }
}

func TestGenerateProof_WithWriterAndVerify(t *testing.T) {
    w := NewWriter()
    fixed := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

    // Append first entry
    e1 := LogEntry{
        ExecutionID: 1,
        JobID:       "job-1",
        Region:      "us-west-1",
        ProviderID:  "p1",
        Status:      "ok",
        OutputHash:  "out1",
        ReceiptHash: "rcpt1",
        IPFSCID:     "cid1",
        Timestamp:   fixed,
    }
    p1, root1 := w.Append(e1)
    if p1 == nil || root1 == "" {
        t.Fatalf("expected non-nil proof and non-empty root after first append")
    }
    // Verify proof 0
    if p1.LeafIndex != 0 {
        t.Fatalf("expected leaf index 0, got %d", p1.LeafIndex)
    }
    if !merkle.VerifyProof(p1) {
        t.Fatalf("proof 1 failed verification")
    }
    // Ensure ComputeEntryLeafHash matches proof leaf hash using writer's stored entry (with PrevHash filled in)
    got := w.Entries()
    if len(got) != 1 {
        t.Fatalf("expected 1 entry stored, got %d", len(got))
    }
    h1 := ComputeEntryLeafHash(got[0])
    // Merkle tree stores leaves as hash(data), where data is the canonical string we already hashed once.
    // Writer adds leaf as the single-hash string, so the tree's leaf hash = sha256(ComputeEntryLeafHash(...)).
    h1Node := doubleHash(h1)
    if h1Node != p1.LeafHash {
        t.Fatalf("leaf hash mismatch: node=%s proof=%s", h1Node, p1.LeafHash)
    }

    // Append second entry; PrevHash should chain to root1
    e2 := LogEntry{
        ExecutionID: 2,
        JobID:       "job-2",
        Region:      "us-west-2",
        ProviderID:  "p2",
        Status:      "ok",
        OutputHash:  "out2",
        ReceiptHash: "rcpt2",
        IPFSCID:     "cid2",
        Timestamp:   fixed.Add(time.Second),
    }
    p2, root2 := w.Append(e2)
    if p2 == nil || root2 == "" {
        t.Fatalf("expected non-nil proof and root after second append")
    }
    if p2.LeafIndex != 1 {
        t.Fatalf("expected leaf index 1, got %d", p2.LeafIndex)
    }
    if !merkle.VerifyProof(p2) {
        t.Fatalf("proof 2 failed verification")
    }
    got = w.Entries()
    if len(got) != 2 {
        t.Fatalf("expected 2 entries stored, got %d", len(got))
    }
    // Check chaining: entry[1].PrevHash equals root after first append
    if got[1].PrevHash != root1 {
        t.Fatalf("expected prev hash to equal first root; got prev=%s root1=%s", got[1].PrevHash, root1)
    }
    h2 := ComputeEntryLeafHash(got[1])
    h2Node := doubleHash(h2)
    if h2Node != p2.LeafHash {
        t.Fatalf("leaf hash mismatch for second entry: node=%s proof=%s", h2Node, p2.LeafHash)
    }

    // Validate wrapper GenerateProof returns same proof as GetProof
    gp, ok := GenerateProof(w, 1)
    if !ok || gp == nil {
        t.Fatalf("expected proof from GenerateProof")
    }
    if gp.LeafHash != p2.LeafHash || gp.RootHash != p2.RootHash || gp.LeafIndex != p2.LeafIndex {
        t.Fatalf("GenerateProof did not match expected proof")
    }
}

// doubleHash computes sha256 over the hex string input.
func doubleHash(hexString string) string {
    // Tree's computeHash takes raw string input; we emulate here using stdlib.
    // It hashes the ASCII representation of the previous hex string.
    h := sha256.Sum256([]byte(hexString))
    return hex.EncodeToString(h[:])
}
