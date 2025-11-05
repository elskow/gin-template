package apm

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/elskow/go-microservice-template/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type MetricsCollector struct {
	meter             metric.Meter
	logger            *slog.Logger
	HttpDuration      metric.Float64Histogram
	HttpResponseSize  metric.Int64Histogram
	HttpErrorCount    metric.Int64Counter
	dbQueryDuration   metric.Float64Histogram
	dbErrorCount      metric.Int64Counter
	dbConnectionCount metric.Int64Gauge
	runtimeGoroutines metric.Int64Gauge
	runtimeMemory     metric.Int64Gauge
	runtimeGCCount    metric.Int64Counter
	RequestThroughput metric.Int64Counter
	mu                sync.RWMutex
	startTime         time.Time
	metricsEnabled    bool
	lastNumGC         uint32
	queryCache        map[string]string // Cache normalized queries
	queryCacheMu      sync.RWMutex
	stopChan          chan struct{}
}

var memStatsPool = sync.Pool{
	New: func() interface{} {
		return &runtime.MemStats{}
	},
}

func getMemStats() *runtime.MemStats {
	return memStatsPool.Get().(*runtime.MemStats)
}

func putMemStats(m *runtime.MemStats) {
	memStatsPool.Put(m)
}

func (mc *MetricsCollector) normalizeQuery(query string) string {
	if len(query) <= 100 {
		return query
	}

	mc.queryCacheMu.RLock()
	if cached, exists := mc.queryCache[query]; exists {
		mc.queryCacheMu.RUnlock()
		return cached
	}
	mc.queryCacheMu.RUnlock()

	normalized := query[:100] + "..."

	mc.queryCacheMu.Lock()
	if len(mc.queryCache) < 512 {
		mc.queryCache[query] = normalized
	}
	mc.queryCacheMu.Unlock()

	return normalized
}

func NewMetricsCollector(logger *slog.Logger) (*MetricsCollector, error) {
	meter := otel.Meter("go-gin-observability/apm")

	mc := &MetricsCollector{
		meter:          meter,
		logger:         logger,
		startTime:      time.Now(),
		metricsEnabled: true,
		queryCache:     make(map[string]string, 64),
		stopChan:       make(chan struct{}),
	}

	var err error

	mc.HttpDuration, err = meter.Float64Histogram(
		"http_request_duration_ms",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	mc.HttpResponseSize, err = meter.Int64Histogram(
		"http_response_size_bytes",
		metric.WithDescription("HTTP response size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	mc.HttpErrorCount, err = meter.Int64Counter(
		"http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
	)
	if err != nil {
		return nil, err
	}

	mc.dbQueryDuration, err = meter.Float64Histogram(
		"db_query_duration_ms",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	mc.dbErrorCount, err = meter.Int64Counter(
		"db_errors_total",
		metric.WithDescription("Total number of database errors"),
	)
	if err != nil {
		return nil, err
	}

	mc.dbConnectionCount, err = meter.Int64Gauge(
		"db_connections_active",
		metric.WithDescription("Number of active database connections"),
	)
	if err != nil {
		return nil, err
	}

	mc.runtimeGoroutines, err = meter.Int64Gauge(
		"runtime_goroutines",
		metric.WithDescription("Number of active goroutines"),
	)
	if err != nil {
		return nil, err
	}

	mc.runtimeMemory, err = meter.Int64Gauge(
		"runtime_memory_heap_bytes",
		metric.WithDescription("Heap memory usage in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	mc.runtimeGCCount, err = meter.Int64Counter(
		"runtime_gc_total",
		metric.WithDescription("Total number of garbage collection cycles"),
	)
	if err != nil {
		return nil, err
	}

	mc.RequestThroughput, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests processed"),
	)
	if err != nil {
		return nil, err
	}

	logger.Info("APM metrics collector initialized")

	go mc.collectRuntimeMetrics()

	return mc, nil
}

func (mc *MetricsCollector) RecordDatabaseQuery(ctx context.Context, query string, duration time.Duration, success bool) {
	if !mc.metricsEnabled {
		return
	}

	durationMs := float64(duration.Milliseconds())

	normalizedQuery := mc.normalizeQuery(query)
	_ = normalizedQuery

	mc.dbQueryDuration.Record(ctx, durationMs)

	if !success {
		mc.dbErrorCount.Add(ctx, 1)
	}
}

func getMetricsCollectionInterval() time.Duration {
	cfg := config.Get()
	return cfg.MetricsCollectionInterval()
}

func (mc *MetricsCollector) collectRuntimeMetrics() {
	interval := getMetricsCollectionInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	mc.lastNumGC = 0

	for {
		select {
		case <-mc.stopChan:
			return
		case <-ticker.C:
			mc.recordRuntimeStats()
		}
	}
}

func (mc *MetricsCollector) recordRuntimeStats() {
	ctx := context.Background()

	goroutines := int64(runtime.NumGoroutine())
	mc.runtimeGoroutines.Record(ctx, goroutines)

	m := getMemStats()
	runtime.ReadMemStats(m)

	mc.runtimeMemory.Record(ctx, int64(m.HeapAlloc))

	if m.NumGC > mc.lastNumGC {
		gcDiff := int64(m.NumGC - mc.lastNumGC)
		mc.runtimeGCCount.Add(ctx, gcDiff)
		mc.lastNumGC = m.NumGC
	}

	putMemStats(m)
}

func (mc *MetricsCollector) GetUptime() time.Duration {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return time.Since(mc.startTime)
}

func (mc *MetricsCollector) Enable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metricsEnabled = true
	mc.logger.Info("APM metrics collection enabled")
}

func (mc *MetricsCollector) Disable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metricsEnabled = false
	mc.logger.Info("APM metrics collection disabled")
}

func (mc *MetricsCollector) IsEnabled() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.metricsEnabled
}

func (mc *MetricsCollector) Shutdown() error {
	mc.logger.Info("shutting down APM metrics collector")
	close(mc.stopChan)
	return nil
}

func (mc *MetricsCollector) ClearQueryCache() {
	mc.queryCacheMu.Lock()
	defer mc.queryCacheMu.Unlock()
	mc.queryCache = make(map[string]string, 64)
}
