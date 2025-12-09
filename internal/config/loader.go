package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"api-proxy/internal/domain"

	"gopkg.in/yaml.v3"
)

type FileConfigRepository struct {
	serverConfigPath  string
	backendConfigPath string
	mu                sync.RWMutex
	config            *domain.ProxyConfig
}

func NewFileConfigRepository(serverPath, backendPath string) *FileConfigRepository {
	return &FileConfigRepository{
		serverConfigPath:  serverPath,
		backendConfigPath: backendPath,
	}
}

func (r *FileConfigRepository) Load() (*domain.ProxyConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load Server Config
	serverData, err := os.ReadFile(r.serverConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server config file: %w", err)
	}
	var serverCfg domain.ServerConfig
	if err := yaml.Unmarshal(serverData, &serverCfg); err != nil {
		return nil, fmt.Errorf("failed to parse server config file: %w", err)
	}

	// Load Backend Config
	backendData, err := os.ReadFile(r.backendConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backend config file: %w", err)
	}
	var backendCfg struct {
		Backends []domain.Backend `yaml:"backends"`
	}
	if err := yaml.Unmarshal(backendData, &backendCfg); err != nil {
		return nil, fmt.Errorf("failed to parse backend config file: %w", err)
	}

	// Merge
	cfg := domain.ProxyConfig{
		Server:   serverCfg,
		Backends: backendCfg.Backends,
	}

	r.config = &cfg
	return &cfg, nil
}

func (r *FileConfigRepository) Watch(interval time.Duration, onChange func(*domain.ProxyConfig)) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// In a real implementation, we might check modtime to avoid unnecessary parsing
			cfg, err := r.Load()
			if err != nil {
				slog.Error("Error reloading config", "error", err)
				continue
			}
			if onChange != nil {
				onChange(cfg)
			}
		}
	}()
}

func (r *FileConfigRepository) GetCurrent() *domain.ProxyConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}
