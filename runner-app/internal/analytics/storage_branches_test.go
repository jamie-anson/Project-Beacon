package analytics

import (
    "context"
    "testing"
)

func TestGenerateRecommendations_AllTriggers(t *testing.T) {
    sa := &StorageAnalytics{}

    storage := &StorageMetrics{
        TotalStorage:  11 * 1024 * 1024 * 1024, // > 10GB
        AverageBundle: 60 * 1024 * 1024,        // > 50MB
    }
    growth := &GrowthAnalysis{
        DailyGrowthRate: 2 * 1024 * 1024 * 1024, // > 1GB/day
    }
    efficiency := &EfficiencyAnalysis{
        StorageEfficiency: 0.65, // < 0.70
    }
    retention := &RetentionMetrics{
        RetentionCompliance: 0.85, // < 0.90
    }

    recs := sa.generateRecommendations(storage, growth, efficiency, retention)
    if len(recs) < 5 {
        t.Fatalf("expected multiple recommendations, got %d: %+v", len(recs), recs)
    }
}

func TestGenerateRecommendations_Default(t *testing.T) {
    sa := &StorageAnalytics{}

    storage := &StorageMetrics{
        TotalStorage:  1 * 1024 * 1024, // tiny
        AverageBundle: 0,
    }
    growth := &GrowthAnalysis{
        DailyGrowthRate: 0,
    }
    efficiency := &EfficiencyAnalysis{
        StorageEfficiency: 0.9,
    }
    retention := &RetentionMetrics{
        RetentionCompliance: 0.99,
    }

    recs := sa.generateRecommendations(storage, growth, efficiency, retention)
    if len(recs) != 1 {
        t.Fatalf("expected default recommendation only, got %d: %+v", len(recs), recs)
    }
}

func TestAnalyzeCosts_WithAndWithoutGrowth(t *testing.T) {
    sa := &StorageAnalytics{}

    storage := &StorageMetrics{
        IPFSStorage:     5 * 1024 * 1024 * 1024, // 5GB
        DatabaseStorage: 2 * 1024 * 1024 * 1024, // 2GB
    }

    // With growth
    growth := &GrowthAnalysis{
        DailyGrowthRate: 10, // any positive
        ProjectedSize30d: 100 * 1024 * 1024, // bytes
        ProjectedSize90d: 200 * 1024 * 1024,
        ProjectedSize1y:  300 * 1024 * 1024,
    }

    ca, err := sa.analyzeCosts(context.Background(), storage, growth)
    if err != nil {
        t.Fatalf("analyzeCosts error: %v", err)
    }
    if ca.CurrentMonthlyCost <= 0 {
        t.Fatalf("expected positive current monthly cost")
    }
    if ca.ProjectedCost30d == 0 || ca.ProjectedCost90d == 0 || ca.ProjectedCost1y == 0 {
        t.Fatalf("expected non-zero projected costs when growth > 0")
    }
    if ca.OptimizationSavings <= 0 {
        t.Fatalf("expected positive optimization savings")
    }

    // No growth
    ca2, err := sa.analyzeCosts(context.Background(), storage, &GrowthAnalysis{})
    if err != nil {
        t.Fatalf("analyzeCosts error: %v", err)
    }
    if ca2.ProjectedCost30d != 0 || ca2.ProjectedCost90d != 0 || ca2.ProjectedCost1y != 0 {
        t.Fatalf("expected zero projected costs when growth == 0")
    }
}

func TestAnalyzeEfficiency_Defaults(t *testing.T) {
    sa := &StorageAnalytics{}
    ea, err := sa.analyzeEfficiency(context.Background())
    if err != nil {
        t.Fatalf("analyzeEfficiency error: %v", err)
    }
    if ea.CompressionRatio == 0 || ea.StorageEfficiency == 0 {
        t.Fatalf("expected default non-zero efficiency metrics: %+v", ea)
    }
}
