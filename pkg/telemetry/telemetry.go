package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/elskow/go-microservice-template/config"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Telemetry struct {
	TracerProvider *trace.TracerProvider
	MeterProvider  *metric.MeterProvider
	logger         *slog.Logger
}

func InitTelemetry(ctx context.Context, serviceName, serviceVersion string, logger *slog.Logger) (*Telemetry, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.ServiceInstanceID(hostname),
		),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider, err := initTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	meterProvider, err := initMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tracerProvider))

	otel.SetMeterProvider(meterProvider)

	cfg := config.Get()
	samplingStrategy := cfg.OTELSamplingStrategy
	samplingRate := cfg.OTELSamplingRate

	logger.Info("telemetry initialized",
		"service", serviceName,
		"version", serviceVersion,
		"instance", hostname,
		"sampling_strategy", samplingStrategy,
		"sampling_rate", samplingRate,
	)

	return &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		logger:         logger,
	}, nil
}

func initTracerProvider(ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	cfg := config.Get()
	otlpEndpoint := cfg.OTELExporterEndpoint

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	sampler := getSampler()

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(time.Second),
			trace.WithMaxExportBatchSize(512),
			trace.WithMaxQueueSize(2048),
		),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	return tracerProvider, nil
}

func initMeterProvider(ctx context.Context, res *resource.Resource) (*metric.MeterProvider, error) {
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	cfg := config.Get()
	otlpEndpoint := cfg.OTELExporterEndpoint

	otlpExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(otlpEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(promExporter),
		metric.WithReader(metric.NewPeriodicReader(otlpExporter)),
		metric.WithResource(res),
	)

	return meterProvider, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	t.logger.Info("shutting down telemetry")

	var hasError bool

	if err := t.TracerProvider.Shutdown(ctx); err != nil {
		t.logger.Warn("tracer provider shutdown issue", "error", err)
		hasError = true
	}

	if err := t.MeterProvider.Shutdown(ctx); err != nil {
		t.logger.Warn("meter provider shutdown issue", "error", err)
		hasError = true
	}

	if !hasError {
		t.logger.Info("telemetry shutdown complete")
	}

	return nil
}

func getSampler() trace.Sampler {
	cfg := config.Get()
	samplingStrategy := cfg.OTELSamplingStrategy
	samplingRate := cfg.OTELSamplingRate

	switch samplingStrategy {
	case "always":
		return trace.AlwaysSample()

	case "never":
		return trace.NeverSample()

	case "parentbased":
		return trace.ParentBased(
			trace.TraceIDRatioBased(samplingRate),
			trace.WithRemoteParentSampled(trace.AlwaysSample()),
			trace.WithRemoteParentNotSampled(trace.NeverSample()),
			trace.WithLocalParentSampled(trace.AlwaysSample()),
			trace.WithLocalParentNotSampled(trace.NeverSample()),
		)

	case "ratio", "":
		return trace.TraceIDRatioBased(samplingRate)

	default:
		return trace.TraceIDRatioBased(samplingRate)
	}
}

func getSamplingRate() float64 {
	cfg := config.Get()
	return cfg.OTELSamplingRate
}
