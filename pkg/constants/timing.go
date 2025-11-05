package constants

import "time"

// Cache configuration defaults
const (
	// DefaultCacheTTL is the default time-to-live for cached permissions
	DefaultCacheTTL = 5 * time.Minute

	// DefaultCacheCleanupInterval is the default interval for cleaning expired cache entries
	DefaultCacheCleanupInterval = 10 * time.Minute
)

// Server timing defaults
const (
	// DefaultShutdownTimeout is the default timeout for graceful server shutdown
	DefaultShutdownTimeout = 5 * time.Second
)

// Metrics collection timing
const (
	// DefaultMetricsCollectionInterval is the default interval for collecting runtime metrics
	DefaultMetricsCollectionInterval = 15 * time.Second
)

// Buffer configuration defaults
const (
	// DefaultLogBufferSize is the default size of the async log buffer
	DefaultLogBufferSize = 5000

	// DevLogBufferSize is a smaller buffer size suitable for development
	DevLogBufferSize = 2000
)

// Environment variable keys for timing configuration
const (
	EnvCacheTTL             = "CACHE_TTL_MINUTES"
	EnvCacheCleanupInterval = "CACHE_CLEANUP_INTERVAL_MINUTES"
	EnvLogBufferSize        = "LOG_BUFFER_SIZE"
	EnvMetricsInterval      = "METRICS_COLLECTION_INTERVAL_SECONDS"
)
