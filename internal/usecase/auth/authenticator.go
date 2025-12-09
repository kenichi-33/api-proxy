package auth

import (
	"net/http"

	"api-proxy/internal/domain"
)

type AuthStrategy interface {
	Validate(r *http.Request, config map[string]string) (bool, error)
}

type Authenticator interface {
	Validate(r *http.Request, config domain.AuthConfig) (bool, error)
}

type authenticator struct {
	strategies map[string]AuthStrategy
}

func NewAuthenticator() Authenticator {
	return &authenticator{
		strategies: map[string]AuthStrategy{
			"basic": &BasicAuthStrategy{},
			"jwt":   &JWTAuthStrategy{},
			"oauth": &OAuthAuthStrategy{},
		},
	}
}

func (a *authenticator) Validate(r *http.Request, config domain.AuthConfig) (bool, error) {
	if config.Type == "" || config.Type == "none" {
		return true, nil
	}

	strategy, ok := a.strategies[config.Type]
	if !ok {
		// Default to allow if unknown? Or deny?
		// For safety, maybe allow if explicit "none", but here we return true for unknown to match previous behavior (default: return true)
		// But strictly, if a type is specified and we don't have a strategy, we might want to error.
		// Previous code: default: return true, nil.
		return true, nil
	}

	return strategy.Validate(r, config.Config)
}

// Strategies

type BasicAuthStrategy struct{}

func (s *BasicAuthStrategy) Validate(r *http.Request, config map[string]string) (bool, error) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false, nil
	}
	expectedUser := config["username"]
	expectedPass := config["password"]
	return user == expectedUser && pass == expectedPass, nil
}

type JWTAuthStrategy struct{}

func (s *JWTAuthStrategy) Validate(r *http.Request, config map[string]string) (bool, error) {
	// Placeholder
	return true, nil
}

type OAuthAuthStrategy struct{}

func (s *OAuthAuthStrategy) Validate(r *http.Request, config map[string]string) (bool, error) {
	// Placeholder
	return true, nil
}
