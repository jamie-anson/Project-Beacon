package transparency

import (
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/merkle"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// LogEntry is a canonical entry for the transparency log.
// Keep minimal fields for now; extend as needed.
type LogEntry struct {
	LogIndex    int64     `json:"log_index"`
	ExecutionID int       `json:"execution_id"`
	JobID       string    `json:"job_id"`
	Region      string    `json:"region"`
	ProviderID  string    `json:"provider_id"`
	Status      string    `json:"status"`
	OutputHash  string    `json:"output_hash"`
	ReceiptHash string    `json:"receipt_hash"`
	IPFSCID     string    `json:"ipfs_cid"`
	PrevHash    string    `json:"prev_hash"`
	Timestamp   time.Time `json:"timestamp"`
}

// Writer maintains a Merkle tree over appended log entries.
// In-memory implementation; can be backed by storage later.
type Writer struct {
	mu      sync.RWMutex
	entries []LogEntry
	tree    *merkle.Tree
}

// NewWriter creates a new empty transparency log writer.
func NewWriter() *Writer {
	return &Writer{tree: merkle.NewTree(nil)}
}

// DefaultWriter is a package-level singleton for quick integration.
var DefaultWriter = NewWriter()

// Optional sinks: persistence and broadcast
var repoSink *store.TransparencyRepo
var broadcaster func(msgType string, data interface{})

// SetRepo configures a repository sink for appends.
func SetRepo(r *store.TransparencyRepo) { repoSink = r }

// RegisterBroadcaster configures a broadcaster callback (e.g., ws hub).
func RegisterBroadcaster(fn func(string, interface{})) { broadcaster = fn }

// Append adds a new entry, updates the Merkle tree, and returns the proof for the new leaf.
func (w *Writer) Append(entry LogEntry) (*merkle.Proof, string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	index := int64(len(w.entries))
	entry.LogIndex = index
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	prevHash := ""
	if len(w.entries) > 0 {
		prevHash = w.tree.GetRootHash()
	}
	entry.PrevHash = prevHash

	leafHash := merkle.ComputeLeafHash(
		entry.LogIndex,
		entry.ExecutionID,
		entry.JobID,
		entry.Region,
		entry.ProviderID,
		entry.Status,
		entry.OutputHash,
		entry.ReceiptHash,
		entry.IPFSCID,
		entry.PrevHash,
		entry.Timestamp.Format(time.RFC3339Nano),
	)

	// Add as leaf (store the canonical data string)
	w.tree.AddLeaf(leafHash)
	w.entries = append(w.entries, entry)

	proof, _ := w.tree.GetProof(len(w.tree.Leaves) - 1)
	root := w.tree.GetRootHash()

	// Persist to repo if configured
	if repoSink != nil {
		var (
			outHash *string
			rcptHash *string
			cid      *string
			rootPtr  *string
		)
		if entry.OutputHash != "" { outHash = &entry.OutputHash }
		if entry.ReceiptHash != "" { rcptHash = &entry.ReceiptHash }
		if entry.IPFSCID != "" { cid = &entry.IPFSCID }
		if root != "" { rootPtr = &root }

		re := &store.TransparencyLogEntry{
			LogIndex:       entry.LogIndex,
			ExecutionID:    entry.ExecutionID,
			JobID:          entry.JobID,
			Region:         entry.Region,
			ProviderID:     entry.ProviderID,
			Status:         entry.Status,
			OutputHash:     outHash,
			ReceiptHash:    rcptHash,
			IPFSCid:        cid,
			MerkleTreeRoot: rootPtr,
			MerkleProof:    proof,
			LoggedAt:       entry.Timestamp,
			PreviousLogHash: entry.PrevHash,
		}
		_ = repoSink.AppendEntry(re)
	}

	// Broadcast event if configured
	if broadcaster != nil {
		broadcaster("transparency.entry_appended", map[string]interface{}{
			"log_index": entry.LogIndex,
			"root":      root,
			"job_id":    entry.JobID,
			"region":    entry.Region,
			"provider_id": entry.ProviderID,
			"status":    entry.Status,
		})
	}
	return proof, root
}

// Root returns the current Merkle root.
func (w *Writer) Root() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.tree.GetRootHash()
}

// Entries returns a copy of entries for inspection.
func (w *Writer) Entries() []LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]LogEntry, len(w.entries))
	copy(out, w.entries)
	return out
}

// GetProof returns the proof for the given index, if present.
func (w *Writer) GetProof(idx int) (*merkle.Proof, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if idx < 0 || idx >= len(w.tree.Leaves) {
		return nil, false
	}
	p, err := w.tree.GetProof(idx)
	if err != nil {
		return nil, false
	}
	return p, true
}
