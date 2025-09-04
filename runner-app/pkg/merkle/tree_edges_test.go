package merkle

import (
	"encoding/json"
	"testing"
)

func TestNewTree_Empty(t *testing.T) {
	tr := NewTree([]string{})
	if tr.Root != nil {
		t.Fatalf("expected nil Root for empty tree, got non-nil")
	}
	if tr.Leaves != nil && len(tr.Leaves) != 0 {
		t.Fatalf("expected no leaves, got %d", len(tr.Leaves))
	}
	if tr.Depth != 0 {
		t.Fatalf("expected depth=0, got %d", tr.Depth)
	}
}

func TestNewTree_OddLeaves_DuplicatesLast(t *testing.T) {
	tr := NewTree([]string{"a", "b", "c"})
	if tr.Root == nil || tr.Root.Left == nil || tr.Root.Right == nil {
		t.Fatalf("unexpected nil nodes in tree")
	}
	// For 3 leaves, the last leaf is duplicated when pairing, so the right child of root
	// should have Left == Right (same pointer) at the level where duplication happened.
	right := tr.Root.Right
	if right.Left != right.Right {
		t.Fatalf("expected duplicated last leaf (right.Left == right.Right), got different pointers")
	}

	// Also ensure proof for last leaf (index 2) verifies
	p, err := tr.GetProof(2)
	if err != nil {
		t.Fatalf("GetProof err: %v", err)
	}
	if !VerifyProof(p) {
		t.Fatalf("proof should verify for last leaf with duplication")
	}
}

func TestGetProof_OutOfRange(t *testing.T) {
	tr := NewTree([]string{"x", "y"})
	if _, err := tr.GetProof(-1); err == nil {
		t.Fatalf("expected error for negative index")
	}
	if _, err := tr.GetProof(2); err == nil {
		t.Fatalf("expected error for out-of-range index")
	}
}

func TestVerifyProof_Tampered(t *testing.T) {
	tr := NewTree([]string{"p", "q", "r", "s"})
	p, err := tr.GetProof(1)
	if err != nil {
		t.Fatalf("GetProof err: %v", err)
	}
	// Tamper with a sibling hash
	if len(p.Siblings) == 0 {
		t.Fatalf("expected siblings in proof")
	}
	p.Siblings[0] = "deadbeef"
	if VerifyProof(p) {
		t.Fatalf("expected tampered proof to fail verification")
	}
	// Tamper with leaf hash
	p2, err := tr.GetProof(1)
	if err != nil { t.Fatal(err) }
	p2.LeafHash = "00"
	if VerifyProof(p2) {
		t.Fatalf("expected tampered leaf hash to fail verification")
	}
}

func TestDeserializeProof_InvalidJSON(t *testing.T) {
	if _, err := DeserializeProof([]byte("not-json")); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
	// Also ensure corrupt but valid JSON shape fails verification
	tr := NewTree([]string{"a", "b"})
	pf, err := tr.GetProof(0)
	if err != nil { t.Fatal(err) }
	b, _ := json.Marshal(pf)
	b[0] = '{' // still valid JSON; we will instead corrupt a required field semantically
	var parsed Proof
	_ = json.Unmarshal(b, &parsed)
	parsed.RootHash = "bad"
	if VerifyProof(&parsed) {
		t.Fatalf("expected corrupted proof not to verify")
	}
}
