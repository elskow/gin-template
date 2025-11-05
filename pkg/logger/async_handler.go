package logger

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type logRecord struct {
	ctx    context.Context
	record slog.Record
}

type asyncHandler struct {
	handler      slog.Handler
	logChan      chan logRecord
	wg           sync.WaitGroup
	stopOnce     sync.Once
	closed       atomic.Bool
	droppedCount atomic.Int64
	dropOnFull   bool
}

func newAsyncHandler(handler slog.Handler, bufferSize int, dropOnFull bool) *asyncHandler {
	if bufferSize <= 0 {
		bufferSize = 5000
	}

	ah := &asyncHandler{
		handler:    handler,
		logChan:    make(chan logRecord, bufferSize),
		dropOnFull: dropOnFull,
	}

	ah.wg.Add(1)
	go ah.processLogs()

	return ah
}

func (h *asyncHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *asyncHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.closed.Load() {
		return nil
	}

	lr := logRecord{
		ctx:    ctx,
		record: record.Clone(),
	}

	if h.dropOnFull {
		select {
		case h.logChan <- lr:
		default:
			h.droppedCount.Add(1)
		}
	} else {
		select {
		case h.logChan <- lr:
		case <-time.After(100 * time.Millisecond):
			h.droppedCount.Add(1)
		}
	}

	return nil
}

func (h *asyncHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return newAsyncHandler(
		h.handler.WithAttrs(attrs),
		cap(h.logChan),
		h.dropOnFull,
	)
}

func (h *asyncHandler) WithGroup(name string) slog.Handler {
	return newAsyncHandler(
		h.handler.WithGroup(name),
		cap(h.logChan),
		h.dropOnFull,
	)
}

func (h *asyncHandler) processLogs() {
	defer h.wg.Done()

	for lr := range h.logChan {
		_ = h.handler.Handle(lr.ctx, lr.record)
	}
}

func (h *asyncHandler) Shutdown(ctx context.Context) error {
	h.stopOnce.Do(func() {
		h.closed.Store(true)
		close(h.logChan)
	})

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *asyncHandler) DroppedCount() int64 {
	return h.droppedCount.Load()
}
