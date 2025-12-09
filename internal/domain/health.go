package domain

// HealthManager manages the health state of the application.
type HealthManager interface {
	// GetStatus returns true if healthy, false otherwise.
	GetStatus() bool
	// SetStatus sets the health status.
	SetStatus(healthy bool)
}
