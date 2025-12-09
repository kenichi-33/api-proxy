package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			ww := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			totalDurationMs := duration.Milliseconds()

			reqID := r.Header.Get(HeaderXRequestID)
			backendDurationStr := w.Header().Get("X-Backend-Duration")
			var backendDurationMs int64
			if backendDurationStr != "" {
				if d, err := time.ParseDuration(backendDurationStr); err == nil {
					backendDurationMs = d.Milliseconds()
				}
			}

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.statusCode),
				slog.Int64("total_duration_ms", totalDurationMs),
				slog.Int64("backend_duration_ms", backendDurationMs),
				slog.String("request_id", reqID),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}
			logger.Info("request completed", attrs...)
		})
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
