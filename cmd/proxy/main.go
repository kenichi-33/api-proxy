package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-proxy/internal/config"
	"api-proxy/internal/domain"
	infra_health "api-proxy/internal/infrastructure/health"
	infra_logger "api-proxy/internal/infrastructure/logger"
	"api-proxy/internal/interface/handler"
	"api-proxy/internal/interface/middleware"
	"api-proxy/internal/usecase/auth"
	"api-proxy/internal/usecase/transform"
)

func main() {
	serverConfigPath := flag.String("server-config", "server.yaml", "Path to server configuration file")
	backendConfigPath := flag.String("backend-config", "backends.yaml", "Path to backend configuration file")
	flag.Parse()

	// 1. Initialize Logger
	logger := infra_logger.NewLogger()
	slog.SetDefault(logger)

	// 2. Load Configuration
	configRepo := config.NewFileConfigRepository(*serverConfigPath, *backendConfigPath)
	cfg, err := configRepo.Load()
	if err != nil {
		logger.Error("Failed to load initial config", "error", err)
		os.Exit(1)
	}

	// 3. Start Config Watcher
	configRepo.Watch(5*time.Minute, func(newCfg *domain.ProxyConfig) {
		logger.Info("Configuration reloaded")
	})

	// Initialize Infrastructure
	healthManager := infra_health.NewAtomicHealthManager()

	// Initialize UseCases
	authenticator := auth.NewAuthenticator()
	transformer := transform.NewTransformer()

	// Initialize Handlers
	proxyHandler := handler.NewProxyHandler(configRepo, healthManager, authenticator, transformer, logger)
	adminHandler := handler.NewAdminHandler(configRepo, healthManager, logger)

	// 6. Setup Middleware Chain
	// We wrap the proxy handler with middleware
	// Order: CorrelationID -> Logging -> Proxy
	// Auth is handled inside ProxyHandler or could be here if global.
	// Since Auth is per-route, we handle it in ProxyHandler logic or per-route middleware.
	// Here we apply global middleware.

	var h http.Handler = proxyHandler
	h = middleware.Logging(logger)(h)
	h = middleware.CorrelationID(h)

	// 7. Start Servers
	// Proxy Server
	proxyServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Admin Server
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/health", adminHandler.Health)
	adminMux.HandleFunc("/health/status", adminHandler.SetHealth)
	adminMux.HandleFunc("/reload", adminHandler.Reload)

	adminPort := cfg.Server.AdminPort
	if adminPort == 0 {
		adminPort = 9090 // Default if not specified
	}

	adminServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", adminPort),
		Handler: adminMux,
	}

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Starting Proxy Server", "addr", proxyServer.Addr)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Proxy Server failed", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		logger.Info("Starting Admin Server", "addr", adminServer.Addr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Admin Server failed", "error", err)
		}
	}()

	<-stop
	logger.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := proxyServer.Shutdown(ctx); err != nil {
		logger.Error("Proxy Server forced to shutdown", "error", err)
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		logger.Error("Admin Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
