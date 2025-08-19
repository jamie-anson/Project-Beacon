package analytics

import (
	"context"
	"database/sql"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// StorageAnalytics provides long-term storage analysis and insights
type StorageAnalytics struct {
	db               *sql.DB
	ipfsRepo         *store.IPFSRepo
	transparencyRepo *store.TransparencyRepo
}

// NewStorageAnalytics creates a new storage analytics service
func NewStorageAnalytics(db *sql.DB, ipfsRepo *store.IPFSRepo, transparencyRepo *store.TransparencyRepo) *StorageAnalytics {
	return &StorageAnalytics{
		db:               db,
		ipfsRepo:         ipfsRepo,
		transparencyRepo: transparencyRepo,
	}
}

// StorageReport represents a comprehensive storage analysis report
type StorageReport struct {
	GeneratedAt      time.Time            `json:"generated_at"`
	TotalStorage     StorageMetrics       `json:"total_storage"`
	GrowthTrends     GrowthAnalysis       `json:"growth_trends"`
	EfficiencyMetrics EfficiencyAnalysis  `json:"efficiency_metrics"`
	RetentionAnalysis RetentionMetrics    `json:"retention_analysis"`
	CostProjections  CostAnalysis         `json:"cost_projections"`
	Recommendations  []string             `json:"recommendations"`
}

// StorageMetrics contains current storage utilization data
type StorageMetrics struct {
	IPFSStorage       int64   `json:"ipfs_storage_bytes"`
	TransparencyLog   int64   `json:"transparency_log_bytes"`
	DatabaseStorage   int64   `json:"database_storage_bytes"`
	TotalStorage      int64   `json:"total_storage_bytes"`
	BundleCount       int64   `json:"bundle_count"`
	LogEntryCount     int64   `json:"log_entry_count"`
	AverageBundle     float64 `json:"average_bundle_size_bytes"`
}

// GrowthAnalysis tracks storage growth patterns
type GrowthAnalysis struct {
	DailyGrowthRate   float64 `json:"daily_growth_rate_bytes"`
	WeeklyGrowthRate  float64 `json:"weekly_growth_rate_bytes"`
	MonthlyGrowthRate float64 `json:"monthly_growth_rate_bytes"`
	ProjectedSize30d  int64   `json:"projected_size_30d_bytes"`
	ProjectedSize90d  int64   `json:"projected_size_90d_bytes"`
	ProjectedSize1y   int64   `json:"projected_size_1y_bytes"`
}

// EfficiencyAnalysis measures storage efficiency
type EfficiencyAnalysis struct {
	CompressionRatio    float64 `json:"compression_ratio"`
	DeduplicationSavings int64  `json:"deduplication_savings_bytes"`
	StorageEfficiency   float64 `json:"storage_efficiency_ratio"`
	WastedSpace         int64   `json:"wasted_space_bytes"`
}

// RetentionMetrics tracks data retention compliance
type RetentionMetrics struct {
	RetentionCompliance float64           `json:"retention_compliance_ratio"`
	ExpiredData         int64             `json:"expired_data_bytes"`
	RetentionPolicies   map[string]int64  `json:"retention_policies_bytes"`
	CleanupOpportunities int64            `json:"cleanup_opportunities_bytes"`
}

// CostAnalysis provides cost projections and optimization insights
type CostAnalysis struct {
	CurrentMonthlyCost  float64 `json:"current_monthly_cost_usd"`
	ProjectedCost30d    float64 `json:"projected_cost_30d_usd"`
	ProjectedCost90d    float64 `json:"projected_cost_90d_usd"`
	ProjectedCost1y     float64 `json:"projected_cost_1y_usd"`
	OptimizationSavings float64 `json:"optimization_savings_usd"`
}

// GenerateStorageReport creates a comprehensive storage analysis report
func (sa *StorageAnalytics) GenerateStorageReport(ctx context.Context) (*StorageReport, error) {
	report := &StorageReport{
		GeneratedAt: time.Now(),
	}

	// Gather current storage metrics
	storageMetrics, err := sa.calculateStorageMetrics(ctx)
	if err != nil {
		return nil, err
	}
	report.TotalStorage = *storageMetrics

	// Analyze growth trends
	growthAnalysis, err := sa.analyzeGrowthTrends(ctx)
	if err != nil {
		return nil, err
	}
	report.GrowthTrends = *growthAnalysis

	// Calculate efficiency metrics
	efficiencyAnalysis, err := sa.analyzeEfficiency(ctx)
	if err != nil {
		return nil, err
	}
	report.EfficiencyMetrics = *efficiencyAnalysis

	// Analyze retention compliance
	retentionMetrics, err := sa.analyzeRetention(ctx)
	if err != nil {
		return nil, err
	}
	report.RetentionAnalysis = *retentionMetrics

	// Project costs
	costAnalysis, err := sa.analyzeCosts(ctx, storageMetrics, growthAnalysis)
	if err != nil {
		return nil, err
	}
	report.CostProjections = *costAnalysis

	// Generate recommendations
	report.Recommendations = sa.generateRecommendations(storageMetrics, growthAnalysis, efficiencyAnalysis, retentionMetrics)

	return report, nil
}

// calculateStorageMetrics computes current storage utilization
func (sa *StorageAnalytics) calculateStorageMetrics(ctx context.Context) (*StorageMetrics, error) {
	metrics := &StorageMetrics{}

	// Get IPFS bundle metrics
	bundles, err := sa.ipfsRepo.ListBundles(0, 10000) // Large limit for analytics
	if err != nil {
		return nil, err
	}

	var totalIPFSStorage int64
	for _, bundle := range bundles {
		if bundle.BundleSize != nil {
			totalIPFSStorage += *bundle.BundleSize
		}
	}

	metrics.IPFSStorage = totalIPFSStorage
	metrics.BundleCount = int64(len(bundles))
	if len(bundles) > 0 {
		metrics.AverageBundle = float64(totalIPFSStorage) / float64(len(bundles))
	}

	// Get transparency log metrics
	logSize, err := sa.transparencyRepo.GetLogSize()
	if err != nil {
		return nil, err
	}
	metrics.LogEntryCount = logSize

	// Estimate transparency log storage (approximate)
	metrics.TransparencyLog = logSize * 512 // Assume ~512 bytes per entry

	// Get database storage size (approximate)
	var dbSize sql.NullInt64
	err = sa.db.QueryRowContext(ctx, `
		SELECT pg_total_relation_size('jobs') + 
		       pg_total_relation_size('executions') + 
		       pg_total_relation_size('ipfs_bundles') + 
		       pg_total_relation_size('transparency_log')
	`).Scan(&dbSize)
	if err == nil && dbSize.Valid {
		metrics.DatabaseStorage = dbSize.Int64
	}

	metrics.TotalStorage = metrics.IPFSStorage + metrics.TransparencyLog + metrics.DatabaseStorage

	return metrics, nil
}

// analyzeGrowthTrends calculates storage growth patterns
func (sa *StorageAnalytics) analyzeGrowthTrends(ctx context.Context) (*GrowthAnalysis, error) {
	analysis := &GrowthAnalysis{}

	// Query historical data for growth analysis
	rows, err := sa.db.QueryContext(ctx, `
		SELECT 
			DATE_TRUNC('day', created_at) as day,
			SUM(COALESCE(bundle_size, 0)) as daily_storage
		FROM ipfs_bundles 
		WHERE created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE_TRUNC('day', created_at)
		ORDER BY day
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyGrowth []int64
	for rows.Next() {
		var day time.Time
		var storage int64
		if err := rows.Scan(&day, &storage); err != nil {
			continue
		}
		dailyGrowth = append(dailyGrowth, storage)
	}

	// Calculate growth rates
	if len(dailyGrowth) > 1 {
		totalGrowth := dailyGrowth[len(dailyGrowth)-1] - dailyGrowth[0]
		days := len(dailyGrowth)
		analysis.DailyGrowthRate = float64(totalGrowth) / float64(days)
		analysis.WeeklyGrowthRate = analysis.DailyGrowthRate * 7
		analysis.MonthlyGrowthRate = analysis.DailyGrowthRate * 30

		// Project future storage needs
		currentStorage := dailyGrowth[len(dailyGrowth)-1]
		analysis.ProjectedSize30d = currentStorage + int64(analysis.DailyGrowthRate*30)
		analysis.ProjectedSize90d = currentStorage + int64(analysis.DailyGrowthRate*90)
		analysis.ProjectedSize1y = currentStorage + int64(analysis.DailyGrowthRate*365)
	}

	return analysis, nil
}

// analyzeEfficiency calculates storage efficiency metrics
func (sa *StorageAnalytics) analyzeEfficiency(ctx context.Context) (*EfficiencyAnalysis, error) {
	analysis := &EfficiencyAnalysis{}

	// For MVP, use estimated values
	// In production, these would be calculated from actual compression and deduplication data
	analysis.CompressionRatio = 0.75        // Assume 75% compression
	analysis.DeduplicationSavings = 0       // No deduplication implemented yet
	analysis.StorageEfficiency = 0.80       // 80% efficiency
	analysis.WastedSpace = 0                // No wasted space calculation yet

	return analysis, nil
}

// analyzeRetention calculates retention compliance metrics
func (sa *StorageAnalytics) analyzeRetention(ctx context.Context) (*RetentionMetrics, error) {
	metrics := &RetentionMetrics{}

	// For MVP, use estimated values
	// In production, these would be calculated based on actual retention policies
	metrics.RetentionCompliance = 0.95      // 95% compliance
	metrics.ExpiredData = 0                 // No expired data calculation yet
	metrics.RetentionPolicies = map[string]int64{
		"7_days":  1024 * 1024 * 100,      // 100MB
		"30_days": 1024 * 1024 * 500,      // 500MB
		"1_year":  1024 * 1024 * 1024 * 5, // 5GB
	}
	metrics.CleanupOpportunities = 0

	return metrics, nil
}

// analyzeCosts projects storage costs
func (sa *StorageAnalytics) analyzeCosts(ctx context.Context, storage *StorageMetrics, growth *GrowthAnalysis) (*CostAnalysis, error) {
	analysis := &CostAnalysis{}

	// Cost assumptions (USD per GB per month)
	ipfsCostPerGB := 0.10    // $0.10 per GB for IPFS storage
	dbCostPerGB := 0.25      // $0.25 per GB for database storage

	// Calculate current monthly costs
	ipfsCostGB := float64(storage.IPFSStorage) / (1024 * 1024 * 1024)
	dbCostGB := float64(storage.DatabaseStorage) / (1024 * 1024 * 1024)
	
	analysis.CurrentMonthlyCost = (ipfsCostGB * ipfsCostPerGB) + (dbCostGB * dbCostPerGB)

	// Project future costs based on growth
	if growth.DailyGrowthRate > 0 {
		growth30dGB := float64(growth.ProjectedSize30d) / (1024 * 1024 * 1024)
		growth90dGB := float64(growth.ProjectedSize90d) / (1024 * 1024 * 1024)
		growth1yGB := float64(growth.ProjectedSize1y) / (1024 * 1024 * 1024)

		analysis.ProjectedCost30d = growth30dGB * ipfsCostPerGB
		analysis.ProjectedCost90d = growth90dGB * ipfsCostPerGB
		analysis.ProjectedCost1y = growth1yGB * ipfsCostPerGB
	}

	// Estimate optimization savings (10% through compression/cleanup)
	analysis.OptimizationSavings = analysis.CurrentMonthlyCost * 0.10

	return analysis, nil
}

// generateRecommendations creates actionable recommendations
func (sa *StorageAnalytics) generateRecommendations(storage *StorageMetrics, growth *GrowthAnalysis, efficiency *EfficiencyAnalysis, retention *RetentionMetrics) []string {
	var recommendations []string

	// Storage growth recommendations
	if growth.DailyGrowthRate > 1024*1024*1024 { // > 1GB/day
		recommendations = append(recommendations, "Consider implementing data compression to reduce storage growth rate")
	}

	// Efficiency recommendations
	if efficiency.StorageEfficiency < 0.70 {
		recommendations = append(recommendations, "Storage efficiency is below 70% - review bundle optimization strategies")
	}

	// Retention recommendations
	if retention.RetentionCompliance < 0.90 {
		recommendations = append(recommendations, "Retention compliance is below 90% - implement automated cleanup policies")
	}

	// Cost optimization recommendations
	if storage.TotalStorage > 10*1024*1024*1024 { // > 10GB
		recommendations = append(recommendations, "Consider tiered storage strategy for cost optimization")
	}

	// Bundle size recommendations
	if storage.AverageBundle > 50*1024*1024 { // > 50MB average
		recommendations = append(recommendations, "Average bundle size is large - consider splitting or compression")
	}

	// Default recommendation if none triggered
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Storage metrics are within acceptable ranges - continue monitoring")
	}

	return recommendations
}

// GetStorageMetricsSummary returns a quick summary of storage metrics
func (sa *StorageAnalytics) GetStorageMetricsSummary(ctx context.Context) (map[string]interface{}, error) {
	metrics, err := sa.calculateStorageMetrics(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_storage_gb":    float64(metrics.TotalStorage) / (1024 * 1024 * 1024),
		"ipfs_storage_gb":     float64(metrics.IPFSStorage) / (1024 * 1024 * 1024),
		"bundle_count":        metrics.BundleCount,
		"log_entry_count":     metrics.LogEntryCount,
		"average_bundle_mb":   metrics.AverageBundle / (1024 * 1024),
		"generated_at":        time.Now(),
	}, nil
}
