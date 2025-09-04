package merkle

import (
	"testing"
)

func TestNewTree(t *testing.T) {
	data := []string{"tx1", "tx2", "tx3", "tx4"}
	tree := NewTree(data)
	
	if tree.Root == nil {
		t.Fatal("Root should not be nil")
	}
	
	if len(tree.Leaves) != 4 {
		t.Fatalf("Expected 4 leaves, got %d", len(tree.Leaves))
	}
	
	// Verify leaf hashes
	for i, leaf := range tree.Leaves {
		expectedHash := computeHash(data[i])
		if leaf.Hash != expectedHash {
			t.Errorf("Leaf %d hash mismatch. Expected %s, got %s", i, expectedHash, leaf.Hash)
		}
	}
}

func TestGetProof(t *testing.T) {
	data := []string{"tx1", "tx2", "tx3", "tx4"}
	tree := NewTree(data)
	
	// Test proof for first leaf
	proof, err := tree.GetProof(0)
	if err != nil {
		t.Fatalf("Failed to get proof: %v", err)
	}
	
	if proof.LeafIndex != 0 {
		t.Errorf("Expected leaf index 0, got %d", proof.LeafIndex)
	}
	
	if proof.LeafHash != tree.Leaves[0].Hash {
		t.Errorf("Proof leaf hash doesn't match actual leaf hash")
	}
	
	if proof.RootHash != tree.Root.Hash {
		t.Errorf("Proof root hash doesn't match actual root hash")
	}
}

func TestVerifyProof(t *testing.T) {
	data := []string{"tx1", "tx2", "tx3", "tx4"}
	tree := NewTree(data)
	
	// Test proof verification for each leaf
	for i := 0; i < len(data); i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Fatalf("Failed to get proof for leaf %d: %v", i, err)
		}
		
		if !VerifyProof(proof) {
			t.Errorf("Proof verification failed for leaf %d", i)
		}
	}
}

func TestAddLeaf(t *testing.T) {
	data := []string{"tx1", "tx2"}
	tree := NewTree(data)
	originalRoot := tree.Root.Hash
	
	// Add new leaf
	tree.AddLeaf("tx3")
	
	if len(tree.Leaves) != 3 {
		t.Fatalf("Expected 3 leaves after adding, got %d", len(tree.Leaves))
	}
	
	// Root should change after adding new leaf
	if tree.Root.Hash == originalRoot {
		t.Error("Root hash should change after adding new leaf")
	}
	
	// Verify all proofs still work
	for i := 0; i < 3; i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Fatalf("Failed to get proof for leaf %d after adding: %v", i, err)
		}
		
		if !VerifyProof(proof) {
			t.Errorf("Proof verification failed for leaf %d after adding", i)
		}
	}
}

func TestComputeLeafHash(t *testing.T) {
	hash := ComputeLeafHash(
		1, 123, "job-001", "us-east-1", "provider-123", "completed",
		"output-hash", "receipt-hash", "ipfs-cid", "prev-hash", "2023-01-01T00:00:00Z",
	)
	
	if hash == "" {
		t.Error("Hash should not be empty")
	}
	
	// Same inputs should produce same hash
	hash2 := ComputeLeafHash(
		1, 123, "job-001", "us-east-1", "provider-123", "completed",
		"output-hash", "receipt-hash", "ipfs-cid", "prev-hash", "2023-01-01T00:00:00Z",
	)
	
	if hash != hash2 {
		t.Error("Same inputs should produce same hash")
	}
	
	// Different inputs should produce different hash
	hash3 := ComputeLeafHash(
		2, 123, "job-001", "us-east-1", "provider-123", "completed",
		"output-hash", "receipt-hash", "ipfs-cid", "prev-hash", "2023-01-01T00:00:00Z",
	)
	
	if hash == hash3 {
		t.Error("Different inputs should produce different hash")
	}
}

func TestProofSerialization(t *testing.T) {
	data := []string{"tx1", "tx2", "tx3", "tx4"}
	tree := NewTree(data)
	
	proof, err := tree.GetProof(0)
	if err != nil {
		t.Fatalf("Failed to get proof: %v", err)
	}
	
	// Serialize
	serialized, err := proof.SerializeProof()
	if err != nil {
		t.Fatalf("Failed to serialize proof: %v", err)
	}
	
	// Deserialize
	deserialized, err := DeserializeProof(serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize proof: %v", err)
	}
	
	// Verify deserialized proof
	if !VerifyProof(deserialized) {
		t.Error("Deserialized proof verification failed")
	}
	
	// Check all fields match
	if proof.LeafHash != deserialized.LeafHash {
		t.Error("LeafHash mismatch after serialization")
	}
	if proof.RootHash != deserialized.RootHash {
		t.Error("RootHash mismatch after serialization")
	}
	if proof.LeafIndex != deserialized.LeafIndex {
		t.Error("LeafIndex mismatch after serialization")
	}
}
