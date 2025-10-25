package performance

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Monitor provides performance monitoring and metrics collection for RAG operations
type Monitor struct {
	metrics    map[string]*OperationMetrics
	mutex      sync.RWMutex
	startTime  time.Time
	collectors []MetricsCollector
}

// OperationMetrics contains performance metrics for a specific operation
type OperationMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalDuration      time.Duration `json:"total_duration"`
	MinDuration        time.Duration `json:"min_duration"`
	MaxDuration        time.Duration `json:"max_duration"`
	AvgDuration        time.Duration `json:"avg_duration"`
	P50Duration        time.Duration `json:"p50_duration"`
	P95Duration        time.Duration `json:"p95_duration"`
	P99Duration        time.Duration `json:"p99_duration"`
	LastUpdated        time.Time     `json:"last_updated"`
	durations          []time.Duration // For percentile calculations
	mutex              sync.RWMutex
}

// MetricsCollector interface for custom metrics collection
type MetricsCollector interface {
	Collect(ctx context.Context, operation string, metrics *OperationMetrics) error
}

// NewMonitor creates a new performance monitor
func NewMonitor() *Monitor {
	return &Monitor{
		metrics:   make(map[string]*OperationMetrics),
		startTime: time.Now(),
	}
}

// AddCollector adds a metrics collector
func (m *Monitor) AddCollector(collector MetricsCollector) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.collectors = append(m.collectors, collector)
}

// StartOperation begins monitoring an operation
func (m *Monitor) StartOperation(operation string) *OperationTracker {
	return &OperationTracker{
		monitor:   m,
		operation: operation,
		startTime: time.Now(),
	}
}

// RecordOperation records a completed operation
func (m *Monitor) RecordOperation(operation string, duration time.Duration, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.metrics[operation] == nil {
		m.metrics[operation] = &OperationMetrics{
			MinDuration: duration,
			MaxDuration: duration,
			durations:   make([]time.Duration, 0, 1000), // Keep last 1000 measurements
		}
	}
	
	metrics := m.metrics[operation]
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	
	atomic.AddInt64(&metrics.TotalRequests, 1)
	if success {
		atomic.AddInt64(&metrics.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&metrics.FailedRequests, 1)
	}
	
	metrics.TotalDuration += duration
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
	
	// Update average
	metrics.AvgDuration = time.Duration(int64(metrics.TotalDuration) / metrics.TotalRequests)
	
	// Keep sliding window of durations for percentile calculations
	if len(metrics.durations) >= 1000 {
		// Remove oldest half
		copy(metrics.durations, metrics.durations[500:])
		metrics.durations = metrics.durations[:500]
	}
	metrics.durations = append(metrics.durations, duration)
	
	// Update percentiles
	m.updatePercentiles(metrics)
	
	metrics.LastUpdated = time.Now()
	
	// Notify collectors
	go m.notifyCollectors(operation, metrics)
}

// GetMetrics returns metrics for a specific operation
func (m *Monitor) GetMetrics(operation string) *OperationMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if metrics, exists := m.metrics[operation]; exists {
		// Return a copy to avoid race conditions
		metrics.mutex.RLock()
		defer metrics.mutex.RUnlock()
		
		return &OperationMetrics{
			TotalRequests:      atomic.LoadInt64(&metrics.TotalRequests),
			SuccessfulRequests: atomic.LoadInt64(&metrics.SuccessfulRequests),
			FailedRequests:     atomic.LoadInt64(&metrics.FailedRequests),
			TotalDuration:      metrics.TotalDuration,
			MinDuration:        metrics.MinDuration,
			MaxDuration:        metrics.MaxDuration,
			AvgDuration:        metrics.AvgDuration,
			P50Duration:        metrics.P50Duration,
			P95Duration:        metrics.P95Duration,
			P99Duration:        metrics.P99Duration,
			LastUpdated:        metrics.LastUpdated,
		}
	}
	
	return nil
}

// GetAllMetrics returns all operation metrics
func (m *Monitor) GetAllMetrics() map[string]*OperationMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	result := make(map[string]*OperationMetrics)
	for operation := range m.metrics {
		result[operation] = m.GetMetrics(operation)
	}
	
	return result
}

// GetSystemStats returns overall system statistics
func (m *Monitor) GetSystemStats() *SystemStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := &SystemStats{
		Uptime:           time.Since(m.startTime),
		TotalOperations:  len(m.metrics),
		LastUpdated:      time.Now(),
	}
	
	for _, metrics := range m.metrics {
		stats.TotalRequests += atomic.LoadInt64(&metrics.TotalRequests)
		stats.TotalSuccessful += atomic.LoadInt64(&metrics.SuccessfulRequests)
		stats.TotalFailed += atomic.LoadInt64(&metrics.FailedRequests)
	}
	
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.TotalSuccessful) / float64(stats.TotalRequests) * 100
	}
	
	return stats
}

// Reset clears all metrics
func (m *Monitor) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.metrics = make(map[string]*OperationMetrics)
	m.startTime = time.Now()
}

// updatePercentiles calculates percentile durations
func (m *Monitor) updatePercentiles(metrics *OperationMetrics) {
	if len(metrics.durations) == 0 {
		return
	}
	
	// Simple percentile calculation (for production, use a more efficient algorithm)
	sorted := make([]time.Duration, len(metrics.durations))
	copy(sorted, metrics.durations)
	
	// Basic bubble sort (fine for small datasets)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	p50Index := int(float64(len(sorted)) * 0.5)
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)
	
	if p50Index < len(sorted) {
		metrics.P50Duration = sorted[p50Index]
	}
	if p95Index < len(sorted) {
		metrics.P95Duration = sorted[p95Index]
	}
	if p99Index < len(sorted) {
		metrics.P99Duration = sorted[p99Index]
	}
}

// notifyCollectors notifies all registered metrics collectors
func (m *Monitor) notifyCollectors(operation string, metrics *OperationMetrics) {
	ctx := context.Background()
	for _, collector := range m.collectors {
		if err := collector.Collect(ctx, operation, metrics); err != nil {
			// Log error but don't fail the operation
			// In production, you might want to use a proper logger
			continue
		}
	}
}

// OperationTracker tracks a single operation
type OperationTracker struct {
	monitor   *Monitor
	operation string
	startTime time.Time
}

// Finish completes the operation tracking
func (t *OperationTracker) Finish(success bool) {
	duration := time.Since(t.startTime)
	t.monitor.RecordOperation(t.operation, duration, success)
}

// FinishWithError completes the operation tracking with an error
func (t *OperationTracker) FinishWithError(err error) {
	duration := time.Since(t.startTime)
	t.monitor.RecordOperation(t.operation, duration, err == nil)
}

// SystemStats contains overall system performance statistics
type SystemStats struct {
	Uptime          time.Duration `json:"uptime"`
	TotalOperations int           `json:"total_operations"`
	TotalRequests   int64         `json:"total_requests"`
	TotalSuccessful int64         `json:"total_successful"`
	TotalFailed     int64         `json:"total_failed"`
	SuccessRate     float64       `json:"success_rate"`
	LastUpdated     time.Time     `json:"last_updated"`
}

// LoggingCollector logs metrics to standard output
type LoggingCollector struct{}

func (l *LoggingCollector) Collect(ctx context.Context, operation string, metrics *OperationMetrics) error {
	// In production, use proper logging
	// fmt.Printf("Metrics for %s: requests=%d, avg_duration=%v, success_rate=%.2f%%\n",
	//     operation, metrics.TotalRequests, metrics.AvgDuration,
	//     float64(metrics.SuccessfulRequests)/float64(metrics.TotalRequests)*100)
	return nil
}

// PrometheusCollector can export metrics to Prometheus (placeholder)
type PrometheusCollector struct {
	// Prometheus metrics would be defined here
}

func (p *PrometheusCollector) Collect(ctx context.Context, operation string, metrics *OperationMetrics) error {
	// Implementation would update Prometheus metrics
	return nil
}