package http

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxRequestsPerTransport = 10000
)

// TransportFactory creates new http.Transports.
// We use a factory to easily recreate transports.
type TransportFactory func() *http.Transport

// RecreatingTransport is a RoundTripper that recreates the underlying http.Transport
// after a certain number of requests or idle time.
type RecreatingTransport struct {
	factory       TransportFactory
	current       *http.Transport
	requestCount  int64
	mu            sync.RWMutex
	idleTimeout   time.Duration
	lastActivity  time.Time
}

func NewRecreatingTransport(idleTimeout time.Duration) *RecreatingTransport {
	rt := &RecreatingTransport{
		factory: func() *http.Transport {
			return &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		},
		idleTimeout: idleTimeout,
		lastActivity: time.Now(),
	}
	rt.current = rt.factory()
	return rt
}

func (t *RecreatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.RLock()
	current := t.current
	t.mu.RUnlock()

	// Check if we need to rotate
	count := atomic.AddInt64(&t.requestCount, 1)
	
	t.mu.Lock()
	// Double check locking
	if t.current != current {
		// Already rotated by another goroutine
		current = t.current
		t.mu.Unlock()
	} else {
		// Check conditions
		shouldRotate := false
		if count >= MaxRequestsPerTransport {
			shouldRotate = true
		} else if time.Since(t.lastActivity) > t.idleTimeout && t.idleTimeout > 0 {
			shouldRotate = true
		}

		if shouldRotate {
			// Rotate
			old := t.current
			t.current = t.factory()
			atomic.StoreInt64(&t.requestCount, 0)
			t.mu.Unlock()

			// Close old transport in background to allow in-flight requests to finish
			// CloseIdleConnections doesn't interrupt active requests
			go func(tr *http.Transport) {
				tr.CloseIdleConnections()
			}(old)
			
			current = t.current
		} else {
			t.mu.Unlock()
		}
	}
	
	// Update activity
	t.mu.Lock()
	t.lastActivity = time.Now()
	t.mu.Unlock()

	return current.RoundTrip(req)
}
