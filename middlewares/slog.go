package middlewares

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/elskow/go-microservice-template/config"
	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

var (
	blacklistPaths     []string
	blacklistPathsOnce sync.Once
)

const maxAttributesCapacity = 12

var slogAttrPool = sync.Pool{
	New: func() interface{} {
		attrs := make([]any, 0, maxAttributesCapacity)
		return &attrs
	},
}

func loadBlacklistPaths() {
	blacklistPathsOnce.Do(func() {
		cfg := config.Get()
		if cfg.LogBlacklistPaths != "" {
			blacklistPaths = strings.Split(cfg.LogBlacklistPaths, ",")
			for i := range blacklistPaths {
				blacklistPaths[i] = strings.TrimSpace(blacklistPaths[i])
			}
		}
	})
}

func isBlacklisted(path string) bool {
	loadBlacklistPaths()
	for _, blacklisted := range blacklistPaths {
		if blacklisted == path {
			return true
		}
	}
	return false
}

func normalizeRequestPath(c *gin.Context, actualPath string) string {
	routePattern := c.FullPath()

	if routePattern != "" {
		return routePattern
	}

	return actualPath
}

func SlogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		if isBlacklisted(path) {
			return
		}

		status := c.Writer.Status()
		latency := time.Since(start)

		spanCtx := trace.SpanContextFromContext(c.Request.Context())

		logLevel := getLogLevel(status, c.Errors)

		attrs := slogAttrPool.Get().(*[]any)
		defer func() {
			*attrs = (*attrs)[:0]
			slogAttrPool.Put(attrs)
		}()

		*attrs = append(*attrs,
			"method", c.Request.Method,
			"path", normalizeRequestPath(c, path),
			"status", status,
			"latency_ms", latency.Milliseconds(),
		)

		if spanCtx.IsValid() {
			*attrs = append(*attrs,
				constants.AttrKeyTraceID, spanCtx.TraceID().String(),
				constants.AttrKeySpanID, spanCtx.SpanID().String(),
			)
		}

		if len(c.Errors) > 0 {
			*attrs = append(*attrs, "error", c.Errors[0].Error())
		}

		switch logLevel {
		case "error":
			logger.Error("http", *attrs...)
		case "warn":
			logger.Warn("http", *attrs...)
		default:
			logger.Info("http", *attrs...)
		}
	}
}

const (
	statusServerError = 500
	statusClientError = 400
)

func getLogLevel(status int, errors []*gin.Error) string {
	if len(errors) > 0 || status >= statusServerError {
		return "error"
	}
	if status >= statusClientError {
		return "warn"
	}
	return "info"
}
