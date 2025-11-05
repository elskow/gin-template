package logger

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	loggerProvider     *sdklog.LoggerProvider
	globalAsyncHandler *asyncHandler
)

func NewLogger(serviceName, serviceVersion string) *slog.Logger {
	config := LoadConfig(serviceName, serviceVersion)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	level := slog.LevelInfo
	if config.Environment == "development" || config.Environment == "dev" {
		level = slog.LevelDebug
	}

	var handlers []slog.Handler

	if config.EnableStdout {
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
		handlers = append(handlers, stdoutHandler)
	}

	if config.EnableOTLP && config.OTLPEndpoint != "" {
		otelHandler := createOTLPHandler(config, hostname)
		if otelHandler != nil {
			globalAsyncHandler = newAsyncHandler(otelHandler, config.BufferSize, config.DropOnFull)
			handlers = append(handlers, globalAsyncHandler)
		}
	}

	var handler slog.Handler
	switch len(handlers) {
	case 0:
		handler = newDiscardHandler()
	case 1:
		handler = handlers[0]
	default:
		handler = newMultiHandler(handlers...)
	}

	return slog.New(handler)
}

func createOTLPHandler(config Config, hostname string) slog.Handler {
	logEndpoint := config.OTLPEndpoint

	const httpPrefix = "http://"
	if len(logEndpoint) > len(httpPrefix) && logEndpoint[:len(httpPrefix)] == httpPrefix {
		logEndpoint = logEndpoint[len(httpPrefix):]
	}

	const metricsPort = "4318"
	const logsPort = "4319"
	if len(logEndpoint) > len(metricsPort) && logEndpoint[len(logEndpoint)-len(metricsPort):] == metricsPort {
		logEndpoint = logEndpoint[:len(logEndpoint)-len(metricsPort)] + logsPort
	}

	ctx := context.Background()
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(logEndpoint),
		otlploghttp.WithURLPath("/v1/logs"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.ServiceInstanceID(hostname),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
	)

	loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	return otelslog.NewHandler(config.ServiceName, otelslog.WithLoggerProvider(loggerProvider))
}

func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}

func Shutdown(ctx context.Context) error {
	var err error

	if globalAsyncHandler != nil {
		if shutdownErr := globalAsyncHandler.Shutdown(ctx); shutdownErr != nil {
			err = shutdownErr
		}
	}

	if loggerProvider != nil {
		if shutdownErr := loggerProvider.Shutdown(ctx); shutdownErr != nil {
			err = shutdownErr
		}
	}

	return err
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return newMultiHandler(handlers...)
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(handlers...)
}
