package logger

import (
	"context"
	"log/slog"
)

type discardHandler struct{}

func newDiscardHandler() *discardHandler {
	return &discardHandler{}
}

func (h *discardHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (h *discardHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h *discardHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *discardHandler) WithGroup(_ string) slog.Handler {
	return h
}
