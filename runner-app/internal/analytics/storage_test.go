package analytics

import (
    "testing"
)

func TestGenerateRecommendations_Branches(t *testing.T) {
    sa := &StorageAnalytics{}

    // Case 1: triggers growth + efficiency + retention + size
    storage := &StorageMetrics{
        IPFSStorage:     0,
        TransparencyLog: 0,
        DatabaseStorage: 0,
        TotalStorage:    11 * 1024 * 1024 * 1024, // >10GB
        AverageBundle:   60 * 1024 * 1024,        // >50MB
    }
    growth := &GrowthAnalysis{DailyGrowthRate: float64(2*1024*1024*1024)} // >1GB/day
    efficiency := &EfficiencyAnalysis{StorageEfficiency: 0.60}
    retention := &RetentionMetrics{RetentionCompliance: 0.80}

    recs := sa.generateRecommendations(storage, growth, efficiency, retention)
    if len(recs) < 4 {
        t.Fatalf("expected multiple recommendations, got %d: %+v", len(recs), recs)
    }

    // Case 2: no triggers -> default recommendation only
    storage2 := &StorageMetrics{TotalStorage: 1, AverageBundle: 1}
    growth2 := &GrowthAnalysis{DailyGrowthRate: 0}
    efficiency2 := &EfficiencyAnalysis{StorageEfficiency: 0.95}
    retention2 := &RetentionMetrics{RetentionCompliance: 0.99}
    recs2 := sa.generateRecommendations(storage2, growth2, efficiency2, retention2)
    if len(recs2) != 1 {
        t.Fatalf("expected default recommendation only, got %d: %+v", len(recs2), recs2)
    }
}
