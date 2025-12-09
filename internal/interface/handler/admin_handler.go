package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"api-proxy/internal/domain"
)

type AdminHandler struct {
	configRepo    domain.ConfigRepository
	healthManager domain.HealthManager
	logger        *slog.Logger
}

func NewAdminHandler(repo domain.ConfigRepository, healthManager domain.HealthManager, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		configRepo:    repo,
		healthManager: healthManager,
		logger:        logger,
	}
}

func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Health check called") // Replaced fmt.Println with slog.Info
	if !h.healthManager.GetStatus() {
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Service Unavailable"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *AdminHandler) SetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Healthy bool `json:"healthy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.healthManager.SetStatus(req.Healthy)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"healthy": req.Healthy})
}

func (h *AdminHandler) Reload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := h.configRepo.Load()
	if err != nil {
		http.Error(w, "Failed to reload config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "reloaded"})
}
