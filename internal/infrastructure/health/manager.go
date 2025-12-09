package health

import (
	"sync/atomic"
)

type AtomicHealthManager struct {
	healthy atomic.Bool
}

func NewAtomicHealthManager() *AtomicHealthManager {
	m := &AtomicHealthManager{}
	m.healthy.Store(true) // Default to healthy
	return m
}

func (m *AtomicHealthManager) GetStatus() bool {
	return m.healthy.Load()
}

func (m *AtomicHealthManager) SetStatus(healthy bool) {
	m.healthy.Store(healthy)
}
