package merkle

import (
    "testing"
)

func TestGetRootHash_EmptyAndNonEmpty(t *testing.T) {
    // Empty tree
    empty := NewTree([]string{})
    if got := empty.GetRootHash(); got != "" {
        t.Fatalf("empty root hash: want '' got %q", got)
    }

    // Non-empty tree
    tr := NewTree([]string{"a", "b", "c"})
    if got := tr.GetRootHash(); got == "" {
        t.Fatalf("non-empty root hash should not be empty")
    }
}

func TestGetRootHash_ChangesOnAddLeaf(t *testing.T) {
    tr := NewTree([]string{"a", "b"})
    before := tr.GetRootHash()
    tr.AddLeaf("c")
    after := tr.GetRootHash()
    if after == "" || before == after {
        t.Fatalf("expected root hash to change after AddLeaf; before=%q after=%q", before, after)
    }
}
