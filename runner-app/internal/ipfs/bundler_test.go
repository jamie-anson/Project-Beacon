package ipfs

import (
	"testing"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

func TestHashOutput_Deterministic(t *testing.T) {
	b := &Bundler{}
	a := b.hashOutput("hello")
	b2 := &Bundler{}
	c := b2.hashOutput("hello")
	if a != c || a == "" {
		t.Fatalf("expected deterministic non-empty hash: %q vs %q", a, c)
	}
	if a == b.hashOutput("world") {
		t.Fatalf("hash should differ for different inputs")
	}
}

func TestGetRegions_UniqueAndOrderAgnostic(t *testing.T) {
	b := &Bundler{}
	execs := []store.Execution{
		{Region: "US"}, {Region: "EU"}, {Region: "US"}, {Region: "APAC"},
	}
	regions := b.getRegions(execs)
	// Expect exactly 3 unique regions
	if len(regions) != 3 {
		t.Fatalf("expected 3 unique regions, got %d: %#v", len(regions), regions)
	}
	// Validate set semantics
	seen := map[string]bool{}
	for _, r := range regions {
		seen[r] = true
	}
	for _, want := range []string{"US", "EU", "APAC"} {
		if !seen[want] {
			t.Fatalf("missing region %q in %v", want, regions)
		}
	}
}

func TestGetRegionsFromBundle_Unique(t *testing.T) {
	b := &Bundler{}
	bun := &Bundle{Receipts: []Receipt{{Region: "US"}, {Region: "EU"}, {Region: "US"}}}
	got := b.getRegionsFromBundle(bun)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique regions, got %d: %#v", len(got), got)
	}
}

func TestNewBundler_AssignsFields(t *testing.T) {
	client := &Client{}
	repo := &store.IPFSRepo{}
	b := NewBundler(client, repo)
	if b.client != client || b.ipfsRepo != repo {
		t.Fatalf("NewBundler did not assign fields correctly")
	}
}
