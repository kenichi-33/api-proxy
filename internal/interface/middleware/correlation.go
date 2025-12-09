package middleware

import (
	"net/http"

	"api-proxy/internal/domain"

	"github.com/google/uuid"
)

const (
	HeaderXRequestID    = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
)

func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderXRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		correlationID := r.Header.Get(HeaderCorrelationID)
		if correlationID == "" {
			correlationID = reqID
		}

		// Set headers for downstream
		r.Header.Set(HeaderXRequestID, reqID)
		r.Header.Set(HeaderCorrelationID, correlationID)

		// Set headers for response
		w.Header().Set(HeaderXRequestID, reqID)
		w.Header().Set(HeaderCorrelationID, correlationID)

		ctx := domain.WithRequestID(r.Context(), reqID)
		ctx = domain.WithCorrelationID(ctx, correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
