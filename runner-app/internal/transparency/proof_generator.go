package transparency

import (
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/merkle"
)

// ComputeEntryLeafHash computes the Merkle leaf hash for a given log entry.
func ComputeEntryLeafHash(e LogEntry) string {
	// Ensure deterministic timestamp format
	ts := e.Timestamp.UTC().Format(time.RFC3339Nano)
	return merkle.ComputeLeafHash(
		e.LogIndex,
		e.ExecutionID,
		e.JobID,
		e.Region,
		e.ProviderID,
		e.Status,
		e.OutputHash,
		e.ReceiptHash,
		e.IPFSCID,
		e.PrevHash,
		ts,
	)
}

// GenerateProof returns the Merkle proof for the entry index from the writer.
func GenerateProof(w *Writer, index int) (*merkle.Proof, bool) {
	return w.GetProof(index)
}
