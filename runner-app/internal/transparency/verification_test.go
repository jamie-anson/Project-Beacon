package transparency

import (
	"context"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/merkle"
)

func TestVerifyProof_TrueForValidProof(t *testing.T) {
	data := []string{"a", "b", "c"}
	tree := merkle.NewTree(data)
	proof, err := tree.GetProof(1) // proof for "b"
	if err != nil {
		t.Fatalf("failed to build proof: %v", err)
	}
	if !VerifyProof(proof) {
		t.Fatalf("expected proof to verify")
	}
}

func TestVerifyBundleCID_Passthrough(t *testing.T) {
	// We cannot hit IPFS in unit tests; ensure method compiles and returns error with nil storage.
	ctx := context.Background()
	storage := NewIPFSStorage(nil)
	// Expect error because no local IPFS; just ensure it doesn't panic.
	_, _ = VerifyBundleCID(ctx, storage, "bafy...fake")
}
