package domain

import "time"

// ProxyConfig represents the root configuration for the proxy.
type ProxyConfig struct {
	Server   ServerConfig `yaml:"server"`
	Backends []Backend    `yaml:"backends"`
}

// ServerConfig holds server-level settings.
type ServerConfig struct {
	Port         int           `yaml:"port"`
	AdminPort    int           `yaml:"admin_port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// Backend represents a backend service configuration.
type Backend struct {
	Name      string           `yaml:"name"`
	Locations []Location       `yaml:"locations"`
	Auth      AuthConfig       `yaml:"auth"`
	Transform []Transformation `yaml:"transform"`
}

// Location defines routing rules for a backend.
type Location struct {
	Path       string        `yaml:"path"`
	BackendURL string        `yaml:"backend_url"`
	Timeout    time.Duration `yaml:"timeout"`
	Methods    []string      `yaml:"methods"`
}

// AuthConfig defines authentication strategy.
type AuthConfig struct {
	Type   string            `yaml:"type"` // "basic", "jwt", "oauth", "none"
	Config map[string]string `yaml:"config"`
}

// Transformation defines a single transformation step.
type Transformation struct {
	Type string                 `yaml:"type"` // "header", "casis", etc.
	Req  map[string]interface{} `yaml:"req"`
	Res  map[string]interface{} `yaml:"res"`
}

// ConfigRepository defines the interface for loading configuration.
type ConfigRepository interface {
	Load() (*ProxyConfig, error)
	Watch(interval time.Duration, onChange func(*ProxyConfig))
	GetCurrent() *ProxyConfig
}
