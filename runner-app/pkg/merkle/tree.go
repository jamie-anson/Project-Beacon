package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

// Node represents a node in the Merkle tree
type Node struct {
	Hash     string `json:"hash"`
	Left     *Node  `json:"left,omitempty"`
	Right    *Node  `json:"right,omitempty"`
	IsLeaf   bool   `json:"is_leaf"`
	Index    int    `json:"index,omitempty"` // For leaf nodes
	Data     string `json:"data,omitempty"`  // Original data for leaf nodes
}

// Tree represents a Merkle tree
type Tree struct {
	Root   *Node    `json:"root"`
	Leaves []*Node  `json:"leaves"`
	Depth  int      `json:"depth"`
}

// Proof represents a Merkle proof for verification
type Proof struct {
	LeafHash   string   `json:"leaf_hash"`
	LeafIndex  int      `json:"leaf_index"`
	Siblings   []string `json:"siblings"`
	Directions []bool   `json:"directions"` // true = right, false = left
	RootHash   string   `json:"root_hash"`
}

// NewTree creates a new Merkle tree from the given data
func NewTree(data []string) *Tree {
	if len(data) == 0 {
		return &Tree{}
	}

	// Create leaf nodes
	leaves := make([]*Node, len(data))
	for i, d := range data {
		hash := computeHash(d)
		leaves[i] = &Node{
			Hash:   hash,
			IsLeaf: true,
			Index:  i,
			Data:   d,
		}
	}

	// Build tree bottom-up
	currentLevel := leaves
	depth := 0

	for len(currentLevel) > 1 {
		nextLevel := make([]*Node, 0)
		
		// Process pairs
		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			var right *Node
			
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			} else {
				// Odd number of nodes - duplicate the last one
				right = currentLevel[i]
			}
			
			// Create parent node
			parentHash := computeHash(left.Hash + right.Hash)
			parent := &Node{
				Hash:   parentHash,
				Left:   left,
				Right:  right,
				IsLeaf: false,
			}
			
			nextLevel = append(nextLevel, parent)
		}
		
		currentLevel = nextLevel
		depth++
	}

	return &Tree{
		Root:   currentLevel[0],
		Leaves: leaves,
		Depth:  depth,
	}
}

// AddLeaf adds a new leaf to the tree and rebuilds it
func (t *Tree) AddLeaf(data string) {
	// Extract existing data
	existingData := make([]string, len(t.Leaves))
	for i, leaf := range t.Leaves {
		existingData[i] = leaf.Data
	}
	
	// Add new data and rebuild tree
	existingData = append(existingData, data)
	newTree := NewTree(existingData)
	
	// Update current tree
	t.Root = newTree.Root
	t.Leaves = newTree.Leaves
	t.Depth = newTree.Depth
}

// GetProof generates a Merkle proof for the leaf at the given index
func (t *Tree) GetProof(leafIndex int) (*Proof, error) {
	if leafIndex < 0 || leafIndex >= len(t.Leaves) {
		return nil, fmt.Errorf("leaf index %d out of range", leafIndex)
	}

	leaf := t.Leaves[leafIndex]
	proof := &Proof{
		LeafHash:   leaf.Hash,
		LeafIndex:  leafIndex,
		Siblings:   make([]string, 0),
		Directions: make([]bool, 0),
		RootHash:   t.Root.Hash,
	}

	// We need to traverse up the tree to collect proof
	t.collectProofPath(t.Root, leafIndex, 0, int(math.Pow(2, float64(t.Depth))), proof)

	return proof, nil
}

// collectProofPath recursively collects the proof path
func (t *Tree) collectProofPath(node *Node, targetIndex, nodeStart, nodeEnd int, proof *Proof) bool {
	if node.IsLeaf {
		return node.Index == targetIndex
	}

	mid := (nodeStart + nodeEnd) / 2
	
	// Check left subtree
	if targetIndex < mid {
		if t.collectProofPath(node.Left, targetIndex, nodeStart, mid, proof) {
			// Target is in left subtree, add right sibling to proof
			if node.Right != nil {
				proof.Siblings = append(proof.Siblings, node.Right.Hash)
				proof.Directions = append(proof.Directions, true) // right sibling
			}
			return true
		}
	} else {
		// Check right subtree
		if node.Right != nil && t.collectProofPath(node.Right, targetIndex, mid, nodeEnd, proof) {
			// Target is in right subtree, add left sibling to proof
			proof.Siblings = append(proof.Siblings, node.Left.Hash)
			proof.Directions = append(proof.Directions, false) // left sibling
			return true
		}
	}
	
	return false
}

// VerifyProof verifies a Merkle proof
func VerifyProof(proof *Proof) bool {
	if proof == nil {
		return false
	}

	currentHash := proof.LeafHash
	
	// Traverse up the tree using the proof
	for i := 0; i < len(proof.Siblings); i++ {
		sibling := proof.Siblings[i]
		isRight := proof.Directions[i]
		
		if isRight {
			// Sibling is on the right
			currentHash = computeHash(currentHash + sibling)
		} else {
			// Sibling is on the left
			currentHash = computeHash(sibling + currentHash)
		}
	}
	
	return currentHash == proof.RootHash
}

// GetRootHash returns the root hash of the tree
func (t *Tree) GetRootHash() string {
	if t.Root == nil {
		return ""
	}
	return t.Root.Hash
}

// SerializeProof converts a proof to JSON
func (p *Proof) SerializeProof() ([]byte, error) {
	return json.Marshal(p)
}

// DeserializeProof converts JSON to a proof
func DeserializeProof(data []byte) (*Proof, error) {
	var proof Proof
	err := json.Unmarshal(data, &proof)
	return &proof, err
}

// computeHash computes SHA-256 hash of the input
func computeHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeLeafHash computes the hash for a transparency log entry
func ComputeLeafHash(logIndex int64, executionID int, jobID, region, providerID, status string, 
	outputHash, receiptHash, ipfsCID, previousHash string, timestamp string) string {
	
	// Create canonical representation
	data := fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s|%s|%s|%s|%s",
		logIndex, executionID, jobID, region, providerID, status,
		outputHash, receiptHash, ipfsCID, previousHash, timestamp)
	
	return computeHash(data)
}
