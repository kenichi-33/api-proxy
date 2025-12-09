package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"api-proxy/internal/domain"
	infra_http "api-proxy/internal/infrastructure/http"
	"api-proxy/internal/usecase/auth"
	"api-proxy/internal/usecase/transform"
)

type ProxyHandler struct {
	configRepo    domain.ConfigRepository
	healthManager domain.HealthManager
	transport     http.RoundTripper
	authenticator auth.Authenticator
	transformer   transform.Transformer
	logger        *slog.Logger
}

type contextKey string

const backendStartTimeKey contextKey = "backend_start_time"

func NewProxyHandler(repo domain.ConfigRepository, healthManager domain.HealthManager, auth auth.Authenticator, trans transform.Transformer, logger *slog.Logger) *ProxyHandler {
	// Create a global recreating transport.
	// In a more complex scenario, we might want one per backend,
	// but for now, a global one handles the "connection pool" requirement generally.
	// If we need strict per-backend isolation, we would manage a map of transports.
	// Given the requirement "backend毎に...個別に設定できるように",
	// we might need to adjust this to be per-backend if they have different transport settings (like TLS).
	// For now, we assume standard transport settings.
	transport := infra_http.NewRecreatingTransport(5 * time.Minute)

	return &ProxyHandler{
		configRepo:    repo,
		healthManager: healthManager,
		transport:     transport,
		authenticator: auth,
		transformer:   trans,
		logger:        logger,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check Health State
	if !h.healthManager.GetStatus() {
		w.Header().Set("Connection", "close")
	}

	cfg := h.configRepo.GetCurrent()
	if cfg == nil {
		http.Error(w, "Proxy not configured", http.StatusServiceUnavailable)
		return
	}

	// 1. Routing Logic
	// Find matching backend and location
	h.logger.DebugContext(r.Context(), "Proxying request", "path", r.URL.Path)
	h.logger.DebugContext(r.Context(), "Config loaded", "backends_count", len(cfg.Backends))

	var matchedBackend *domain.Backend
	var matchedLocation *domain.Location

	// Simple linear search for now. Can be optimized with a trie or router.
	for _, backend := range cfg.Backends {
		for _, loc := range backend.Locations {
			if strings.HasPrefix(r.URL.Path, loc.Path) {
				if len(r.URL.Path) == len(loc.Path) || r.URL.Path[len(loc.Path)] == '/' {
					matchedBackend = &backend
					matchedLocation = &loc
					break
				}
			}
		}
		if matchedBackend != nil {
			break
		}
	}

	if matchedBackend == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// 2. Auth Check
	if ok, _ := h.authenticator.Validate(r, matchedBackend.Auth); !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	target, err := url.Parse(matchedLocation.BackendURL)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 3. Setup Reverse Proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path + req.URL.Path
			req.Host = target.Host
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}

			// 4. Transformations (Request)
			if err := h.transformer.TransformRequest(req, matchedBackend.Transform); err != nil {
				h.logger.ErrorContext(req.Context(), "Failed to transform request", "error", err)
				// We can't easily abort here in Director without a custom Transport or panic.
				// For now, we log it. The request might be malformed or missing headers.
			}

			// 5. Context for Timing
			ctx := context.WithValue(req.Context(), backendStartTimeKey, time.Now())
			*req = *req.WithContext(ctx)
		},
		Transport: h.transport,
		ModifyResponse: func(res *http.Response) error {
			// Calculate Backend Duration
			if start, ok := res.Request.Context().Value(backendStartTimeKey).(time.Time); ok {
				duration := time.Since(start)
				res.Header.Set("X-Backend-Duration", duration.String())
			}

			if err := h.transformer.TransformResponse(res, matchedBackend.Transform); err != nil {
				return err
			}
			return nil
		},
	}

	proxy.ServeHTTP(w, r)
}
