package monitoring

import (
	"context"
	"runtime"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

// ResourceMonitor tracks system resource usage for the runner app
type ResourceMonitor struct {
	interval time.Duration
}

// NewResourceMonitor creates a new resource monitor with the specified interval
func NewResourceMonitor(interval time.Duration) *ResourceMonitor {
	if interval <= 0 {
		interval = 30 * time.Second // Default to 30 second intervals
	}
	return &ResourceMonitor{interval: interval}
}

// Start begins monitoring system resources in a background goroutine
func (rm *ResourceMonitor) Start(ctx context.Context) {
	l := logging.FromContext(ctx)
	l.Info().Dur("interval", rm.interval).Msg("resource monitor started")
	
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			l.Info().Msg("resource monitor stopping")
			return
		case <-ticker.C:
			rm.collectMetrics(ctx)
		}
	}
}

// collectMetrics gathers and logs current resource usage
func (rm *ResourceMonitor) collectMetrics(ctx context.Context) {
	l := logging.FromContext(ctx)
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Memory metrics
	heapAllocMB := float64(m.HeapAlloc) / 1024 / 1024
	heapSysMB := float64(m.HeapSys) / 1024 / 1024
	stackInUseMB := float64(m.StackInuse) / 1024 / 1024
	
	// GC metrics
	gcPauseMs := float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000
	
	// Goroutine count
	numGoroutines := runtime.NumGoroutine()
	
	// Update Prometheus metrics
	metrics.MemoryHeapAllocBytes.Set(float64(m.HeapAlloc))
	metrics.MemoryHeapSysBytes.Set(float64(m.HeapSys))
	metrics.MemoryStackInUseBytes.Set(float64(m.StackInuse))
	metrics.GoroutineCount.Set(float64(numGoroutines))
	metrics.GCPauseDurationSeconds.Set(gcPauseMs / 1000)
	
	// Log detailed resource usage
	l.Debug().
		Float64("heap_alloc_mb", heapAllocMB).
		Float64("heap_sys_mb", heapSysMB).
		Float64("stack_inuse_mb", stackInUseMB).
		Int("goroutines", numGoroutines).
		Uint32("gc_cycles", m.NumGC).
		Float64("gc_pause_ms", gcPauseMs).
		Msg("resource usage snapshot")
	
	// Alert on high resource usage
	if heapAllocMB > 100 {
		l.Warn().Float64("heap_alloc_mb", heapAllocMB).Msg("high memory usage detected")
	}
	
	if numGoroutines > 100 {
		l.Warn().Int("goroutines", numGoroutines).Msg("high goroutine count detected")
	}
	
	if gcPauseMs > 10 {
		l.Warn().Float64("gc_pause_ms", gcPauseMs).Msg("long GC pause detected")
	}
}

// GetCurrentStats returns current resource usage statistics
func (rm *ResourceMonitor) GetCurrentStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"heap_alloc_mb":    float64(m.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":      float64(m.HeapSys) / 1024 / 1024,
		"stack_inuse_mb":   float64(m.StackInuse) / 1024 / 1024,
		"goroutines":       runtime.NumGoroutine(),
		"gc_cycles":        m.NumGC,
		"gc_pause_ms":      float64(m.PauseNs[(m.NumGC+255)%256]) / 1000000,
		"mallocs":          m.Mallocs,
		"frees":            m.Frees,
		"heap_objects":     m.HeapObjects,
	}
}
