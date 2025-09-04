package negotiation

import (
	"testing"
)

func TestDemandBuilderBuild(t *testing.T) {
	b := NewDemandBuilder()
	req := RegionRequest{
		Region:            "EU",
		MinVCPU:           2,
		MinMemGiB:         4,
		NetworkEgress:     true,
		PricePerMinuteMax: 0.05,
		TotalPriceCap:     2.5,
	}
	got, err := b.Build(req)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if got.Runtime != "docker" {
		t.Fatalf("runtime = %q want docker", got.Runtime)
	}
	if got.MinVCPU != req.MinVCPU || got.MinMemGiB != req.MinMemGiB {
		t.Fatalf("resources mismatch: got vcpu=%d mem=%d", got.MinVCPU, got.MinMemGiB)
	}
	if got.NetworkEgress != req.NetworkEgress {
		t.Fatalf("egress = %v want %v", got.NetworkEgress, req.NetworkEgress)
	}
	if got.PricePerMinuteMax != req.PricePerMinuteMax || got.TotalPriceCap != req.TotalPriceCap {
		t.Fatalf("pricing mismatch: got %v/%v", got.PricePerMinuteMax, got.TotalPriceCap)
	}
	if got.Region != req.Region {
		t.Fatalf("region = %q want %q", got.Region, req.Region)
	}
}
