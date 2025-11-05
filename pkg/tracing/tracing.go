package tracing

import (
	"context"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Layer string

const (
	LayerController Layer = "controller"
	LayerService    Layer = "service"
	LayerRepository Layer = "repository"
	LayerMiddleware Layer = "middleware"
	LayerUnknown    Layer = "unknown"
)

type spanInfo struct {
	layer      Layer
	domain     string
	method     string
	tracer     trace.Tracer
	opName     string
	tracerName string
}

var (
	spanInfoCache = sync.Map{}
	layerAttrs    = map[Layer]attribute.KeyValue{
		LayerController: attribute.String("layer", string(LayerController)),
		LayerService:    attribute.String("layer", string(LayerService)),
		LayerRepository: attribute.String("layer", string(LayerRepository)),
		LayerMiddleware: attribute.String("layer", string(LayerMiddleware)),
		LayerUnknown:    attribute.String("layer", string(LayerUnknown)),
	}
	attrPool = sync.Pool{
		New: func() interface{} {
			s := make([]attribute.KeyValue, 0, 8)
			return &s
		},
	}
)

func getSpanInfo(pc uintptr) spanInfo {
	if cached, ok := spanInfoCache.Load(pc); ok {
		return cached.(spanInfo)
	}

	info := spanInfo{
		layer:  LayerUnknown,
		domain: "unknown",
		method: "unknown",
	}

	fn := runtime.FuncForPC(pc)
	if fn != nil {
		fullName := fn.Name()

		if idx := strings.Index(fullName, "/controller."); idx != -1 {
			info.layer = LayerController
		} else if idx := strings.Index(fullName, "/service."); idx != -1 {
			info.layer = LayerService
		} else if idx := strings.Index(fullName, "/repository."); idx != -1 {
			info.layer = LayerRepository
		} else if idx := strings.Index(fullName, "/middlewares."); idx != -1 {
			info.layer = LayerMiddleware
		}

		if idx := strings.Index(fullName, "/modules/"); idx != -1 {
			remaining := fullName[idx+9:]
			if slashIdx := strings.Index(remaining, "/"); slashIdx != -1 {
				info.domain = remaining[:slashIdx]
			}
		}

		if idx := strings.LastIndexByte(fullName, '.'); idx != -1 {
			info.method = fullName[idx+1:]
		}
	}

	var opNameBuilder, tracerNameBuilder strings.Builder

	opNameBuilder.Grow(len(info.layer) + 1 + len(info.method))
	opNameBuilder.WriteString(string(info.layer))
	opNameBuilder.WriteByte('.')
	opNameBuilder.WriteString(info.method)
	info.opName = opNameBuilder.String()

	tracerNameBuilder.Grow(len(info.domain) + 1 + len(info.layer))
	tracerNameBuilder.WriteString(info.domain)
	tracerNameBuilder.WriteByte('.')
	tracerNameBuilder.WriteString(string(info.layer))
	info.tracerName = tracerNameBuilder.String()

	info.tracer = otel.Tracer(info.tracerName)

	spanInfoCache.Store(pc, info)

	return info
}

func Auto(ctx context.Context, attributes ...attribute.KeyValue) (context.Context, *Span) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		tracer := otel.Tracer("unknown")
		otelCtx, otelSpan := tracer.Start(ctx, "unknown.unknown")
		return otelCtx, &Span{Span: otelSpan}
	}

	info := getSpanInfo(pc)

	layerAttr := layerAttrs[info.layer]
	domainAttr := attribute.String("domain", info.domain)

	attrsPtr := attrPool.Get().(*[]attribute.KeyValue)
	attrs := *attrsPtr
	attrs = attrs[:0]

	attrs = append(attrs, attributes...)
	attrs = append(attrs, layerAttr, domainAttr)

	otelCtx, otelSpan := info.tracer.Start(ctx, info.opName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)

	*attrsPtr = attrs
	attrPool.Put(attrsPtr)

	return otelCtx, &Span{Span: otelSpan}
}

type Span struct {
	trace.Span
	err error
}

func (s *Span) End() {
	if s.err != nil {
		s.RecordError(s.err)
		s.SetStatus(codes.Error, s.err.Error())
	}
	s.Span.End()
}

func (s *Span) SetError(err error) {
	s.err = err
}

func (s *Span) RecordError(err error) {
	if err != nil {
		s.err = err

		var opts []trace.EventOption

		if shouldCaptureStacktrace() {
			stacktrace := extractStacktrace(err)
			if stacktrace != "" {
				opts = append(opts, trace.WithAttributes(
					attribute.String("exception.stacktrace", stacktrace),
				))
			}
		}

		s.Span.RecordError(err, opts...)
		s.SetStatus(codes.Error, err.Error())
	}
}

func shouldCaptureStacktrace() bool {
	env := os.Getenv("APP_ENV")
	return env == "dev" || env == "development"
}

func extractStacktrace(err error) string {
	// Return current goroutine stack trace
	return string(debug.Stack())
}
