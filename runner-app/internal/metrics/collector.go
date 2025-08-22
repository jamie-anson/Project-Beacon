package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// Small interfaces to allow testing without a real DB
type ipfsRepoIface interface {
    ListBundles(limit, offset int) ([]store.IPFSBundle, error)
}

type transparencyRepoIface interface {
    GetLogSize() (int64, error)
    VerifyLogIntegrity() (bool, error)
}

// Collector manages all Project Beacon metrics
type Collector struct {
	// IPFS Metrics
	ipfsBundlesCreated    prometheus.Counter
	ipfsBundleSize        prometheus.Histogram
	ipfsPinOperations     prometheus.Counter
	ipfsStorageTotal      prometheus.Gauge
	ipfsGatewayLatency    prometheus.Histogram

	// Transparency Log Metrics
	transparencyLogEntries    prometheus.Gauge
	transparencyLogIntegrity  prometheus.Gauge
	transparencyVerifyFails   prometheus.Counter
	transparencyAnchorSuccess prometheus.Counter
	transparencyAnchorAttempt prometheus.Counter
	anchorDuration           prometheus.HistogramVec

	// Storage Analytics
	storageEfficiency    prometheus.Gauge
	bundleCompressionRatio prometheus.Histogram
	retentionCompliance   prometheus.Gauge

	// Repository dependencies (interfaces for testability)
	ipfsRepo        ipfsRepoIface
	transparencyRepo transparencyRepoIface
}

// NewCollector creates a new metrics collector
func NewCollector(ipfsRepo *store.IPFSRepo, transparencyRepo *store.TransparencyRepo) *Collector {
	c := &Collector{
		// IPFS Metrics
		ipfsBundlesCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ipfs_bundles_created_total",
			Help: "Total number of IPFS bundles created",
		}),
		ipfsBundleSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "ipfs_bundle_size_bytes",
			Help: "Size distribution of IPFS bundles in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to ~1GB
		}),
		ipfsPinOperations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ipfs_pin_operations_total",
			Help: "Total number of IPFS pin operations",
		}),
		ipfsStorageTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ipfs_storage_bytes_total",
			Help: "Total storage used by IPFS bundles in bytes",
		}),
		ipfsGatewayLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "ipfs_gateway_latency_seconds",
			Help: "Latency of IPFS gateway requests",
			Buckets: prometheus.DefBuckets,
		}),

		// Transparency Log Metrics
		transparencyLogEntries: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "transparency_log_entries_total",
			Help: "Total number of entries in transparency log",
		}),
		transparencyLogIntegrity: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "transparency_log_integrity_valid",
			Help: "Whether transparency log integrity is valid (1=valid, 0=invalid)",
		}),
		transparencyVerifyFails: promauto.NewCounter(prometheus.CounterOpts{
			Name: "transparency_log_verification_failures_total",
			Help: "Total number of transparency log verification failures",
		}),
		transparencyAnchorSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "transparency_anchors_success_total",
			Help: "Total number of successful anchor operations",
		}),
		transparencyAnchorAttempt: promauto.NewCounter(prometheus.CounterOpts{
			Name: "transparency_anchors_attempted_total",
			Help: "Total number of attempted anchor operations",
		}),
		anchorDuration: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "anchor_duration_seconds",
			Help: "Duration of anchor operations by strategy",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to ~100s
		}, []string{"strategy"}),

		// Storage Analytics
		storageEfficiency: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "storage_efficiency_ratio",
			Help: "Storage efficiency ratio (compressed/uncompressed)",
		}),
		bundleCompressionRatio: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "bundle_compression_ratio",
			Help: "Compression ratio distribution for bundles",
			Buckets: prometheus.LinearBuckets(0.1, 0.1, 10), // 0.1 to 1.0
		}),
		retentionCompliance: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "retention_compliance_ratio",
			Help: "Ratio of data meeting retention requirements",
		}),

	}
	// Only assign interface fields when concrete repos are non-nil to avoid typed-nil interface values
	if ipfsRepo != nil {
		c.ipfsRepo = ipfsRepo
	}
	if transparencyRepo != nil {
		c.transparencyRepo = transparencyRepo
	}
	return c
}

// RecordIPFSBundleCreated records a new IPFS bundle creation
func (c *Collector) RecordIPFSBundleCreated(sizeBytes int64) {
	c.ipfsBundlesCreated.Inc()
	c.ipfsBundleSize.Observe(float64(sizeBytes))
}

// RecordIPFSPinOperation records an IPFS pin operation
func (c *Collector) RecordIPFSPinOperation() {
	c.ipfsPinOperations.Inc()
}

// RecordIPFSGatewayLatency records IPFS gateway request latency
func (c *Collector) RecordIPFSGatewayLatency(duration time.Duration) {
	c.ipfsGatewayLatency.Observe(duration.Seconds())
}

// RecordTransparencyLogEntry records a new transparency log entry
func (c *Collector) RecordTransparencyLogEntry() {
	c.transparencyLogEntries.Inc()
}

// RecordTransparencyVerificationFailure records a verification failure
func (c *Collector) RecordTransparencyVerificationFailure() {
	c.transparencyVerifyFails.Inc()
}

// RecordAnchorAttempt records an anchor attempt
func (c *Collector) RecordAnchorAttempt(strategy string, duration time.Duration, success bool) {
	c.transparencyAnchorAttempt.Inc()
	c.anchorDuration.WithLabelValues(strategy).Observe(duration.Seconds())
	
	if success {
		c.transparencyAnchorSuccess.Inc()
	}
}

// RecordCompressionRatio records bundle compression ratio
func (c *Collector) RecordCompressionRatio(ratio float64) {
	c.bundleCompressionRatio.Observe(ratio)
}

// UpdateStorageMetrics updates storage-related metrics from database
func (c *Collector) UpdateStorageMetrics(ctx context.Context) error {
	if c.ipfsRepo == nil {
		return nil
	}

	// Get all bundles to calculate storage metrics
	bundles, err := c.ipfsRepo.ListBundles(0, 10000) // Large limit for metrics
	if err != nil {
		return err
	}

	var totalStorage int64
	for _, bundle := range bundles {
		if bundle.BundleSize != nil {
			totalStorage += *bundle.BundleSize
		}
	}

	c.ipfsStorageTotal.Set(float64(totalStorage))

	// Calculate storage efficiency (placeholder - would need actual compression data)
	c.storageEfficiency.Set(0.75) // Assume 75% efficiency for now

	return nil
}

// UpdateTransparencyMetrics updates transparency log metrics from database
func (c *Collector) UpdateTransparencyMetrics(ctx context.Context) error {
	if c.transparencyRepo == nil {
		return nil
	}

	// Get log size
	logSize, err := c.transparencyRepo.GetLogSize()
	if err != nil {
		return err
	}
	c.transparencyLogEntries.Set(float64(logSize))

	// Check integrity
	isValid, err := c.transparencyRepo.VerifyLogIntegrity()
	if err != nil {
		return err
	}
	
	if isValid {
		c.transparencyLogIntegrity.Set(1)
	} else {
		c.transparencyLogIntegrity.Set(0)
	}

	return nil
}

// UpdateRetentionMetrics updates data retention compliance metrics
func (c *Collector) UpdateRetentionMetrics(ctx context.Context) error {
	// Calculate retention compliance based on data age and policies
	// This would integrate with actual retention policies
	c.retentionCompliance.Set(0.95) // Assume 95% compliance for now
	
	return nil
}

// StartPeriodicUpdates starts background metric updates
func (c *Collector) StartPeriodicUpdates(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update all metrics from database
			c.UpdateStorageMetrics(ctx)
			c.UpdateTransparencyMetrics(ctx)
			c.UpdateRetentionMetrics(ctx)
		}
	}
}
