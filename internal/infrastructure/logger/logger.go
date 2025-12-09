package logger

import (
	"context"
	"log/slog"
	"os"

	"api-proxy/internal/domain"
)

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := domain.GetRequestID(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	if correlationID := domain.GetCorrelationID(ctx); correlationID != "" {
		r.AddAttrs(slog.String("correlation_id", correlationID))
	}
	return h.Handler.Handle(ctx, r)
}

func NewLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Changed to Debug to see debug logs
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(&ContextHandler{Handler: handler})
}
