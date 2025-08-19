package transparency

import (
	"context"

	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/pkg/merkle"
)

// VerifyProof validates a Merkle proof against its root.
func VerifyProof(proof *merkle.Proof) bool {
	return merkle.VerifyProof(proof)
}

// VerifyBundleCID fetches a bundle by CID and returns it for external checks.
// Kept minimal for now; callers can validate hashes/signatures as needed.
func VerifyBundleCID(ctx context.Context, storage *IPFSStorage, cid string) (*ipfs.Bundle, error) {
	return storage.FetchBundle(ctx, cid)
}
