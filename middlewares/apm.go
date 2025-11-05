package middlewares

import (
	"context"
	"sync"
	"time"

	"github.com/elskow/go-microservice-template/pkg/apm"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const attributePoolCapacity = 8

var apmAttrPool = sync.Pool{
	New: func() interface{} {
		attrs := make([]attribute.KeyValue, 0, attributePoolCapacity)
		return &attrs
	},
}

func getAttributeSlice() *[]attribute.KeyValue {
	return apmAttrPool.Get().(*[]attribute.KeyValue)
}

func putAttributeSlice(attrs *[]attribute.KeyValue) {
	*attrs = (*attrs)[:0]
	apmAttrPool.Put(attrs)
}

const (
	statusThreshold3xx = 300
	statusThreshold4xx = 400
	statusThreshold5xx = 500
)

func getStatusCategory(statusCode int) string {
	switch {
	case statusCode < statusThreshold3xx:
		return "2xx"
	case statusCode < statusThreshold4xx:
		return "3xx"
	case statusCode < statusThreshold5xx:
		return "4xx"
	default:
		return "5xx"
	}
}

type pathNormalizer struct {
	mu    sync.RWMutex
	cache map[string]string
}

var pathCache = &pathNormalizer{
	cache: make(map[string]string, 256),
}

const (
	maxPathLength      = 100
	maxCacheSize       = 1024
	pathTruncationMark = "..."
)

func normalizePath(path string) string {
	if len(path) <= maxPathLength {
		return path
	}

	pathCache.mu.RLock()
	if normalized, exists := pathCache.cache[path]; exists {
		pathCache.mu.RUnlock()
		return normalized
	}
	pathCache.mu.RUnlock()

	normalized := path[:maxPathLength] + pathTruncationMark

	pathCache.mu.Lock()
	if len(pathCache.cache) < maxCacheSize {
		pathCache.cache[path] = normalized
	}
	pathCache.mu.Unlock()

	return normalized
}

var skipPathsSet = map[string]bool{
	"/health":      true,
	"/metrics":     true,
	"/ready":       true,
	"/ping":        true,
	"/favicon.ico": true,
}

const staticPathPrefix = "/static/"

func shouldSkipMetrics(path string) bool {
	if skipPathsSet[path] {
		return true
	}

	if len(path) > len(staticPathPrefix) && path[:len(staticPathPrefix)] == staticPathPrefix {
		return true
	}

	return false
}

func HTTPMetricsMiddleware(metricsCollector *apm.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldSkipMetrics(path) {
			c.Next()
			return
		}

		startTime := time.Now()

		ctx := c.Request.Context()

		const defaultStatusCode = 200
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			statusCode:     defaultStatusCode,
			size:           0,
		}
		c.Writer = writer

		c.Next()

		const notFoundStatus = 404
		statusCode := c.Writer.Status()
		if statusCode == notFoundStatus {
			return
		}

		duration := time.Since(startTime)
		responseSize := int64(c.Writer.Size())

		recordHTTPMetrics(ctx, metricsCollector, c.Request.Method, path, statusCode, duration, responseSize)
	}
}

type responseWriter struct {
	gin.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

func (w *responseWriter) WriteString(s string) (int, error) {
	size, err := w.ResponseWriter.WriteString(s)
	w.size += size
	return size, err
}

func recordHTTPMetrics(ctx context.Context, mc *apm.MetricsCollector, method string, path string, statusCode int, duration time.Duration, responseSize int64) {
	if !mc.IsEnabled() {
		return
	}

	durationMs := float64(duration.Milliseconds())
	statusCategory := getStatusCategory(statusCode)
	normalizedPath := normalizePath(path)

	attrs := getAttributeSlice()
	defer putAttributeSlice(attrs)

	*attrs = append(*attrs,
		attribute.String("http.method", method),
		attribute.String("http.path", normalizedPath),
		attribute.Int("http.status_code", statusCode),
		attribute.String("http.status_class", statusCategory),
	)

	mc.HttpDuration.Record(ctx, durationMs, metric.WithAttributes(*attrs...))

	mc.HttpResponseSize.Record(ctx, responseSize, metric.WithAttributes(*attrs...))

	*attrs = (*attrs)[:0]
	*attrs = append(*attrs,
		attribute.String("http.method", method),
		attribute.String("http.status_class", statusCategory),
	)
	mc.RequestThroughput.Add(ctx, 1, metric.WithAttributes(*attrs...))

	const clientErrorThreshold = 400
	if statusCode >= clientErrorThreshold {
		*attrs = (*attrs)[:0]
		*attrs = append(*attrs,
			attribute.String("http.method", method),
			attribute.String("http.path", normalizedPath),
			attribute.Int("http.status_code", statusCode),
		)
		mc.HttpErrorCount.Add(ctx, 1, metric.WithAttributes(*attrs...))
	}
}
