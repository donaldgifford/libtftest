package sneakystack

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Sidecar wraps a Proxy as a harness.Sidecar implementation.
type Sidecar struct {
	cfg    Config
	server *http.Server
	addr   string
}

// NewSidecar creates a sneakystack Sidecar for use with harness.Run.
func NewSidecar(cfg Config) *Sidecar {
	return &Sidecar{cfg: cfg}
}

// Start launches the sneakystack proxy on an ephemeral port and returns the
// URL callers should use instead of the raw LocalStack URL.
func (s *Sidecar) Start(ctx context.Context, localstackURL string) (string, error) {
	store := NewMapStore()

	proxy, err := NewProxy(store, localstackURL)
	if err != nil {
		return "", fmt.Errorf("create proxy: %w", err)
	}

	// Register handlers based on configured services.
	// Service handlers are registered here as they are implemented.
	// For now, the proxy forwards everything to LocalStack.
	_ = proxy // Handlers will be registered in future commits.

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}

	s.addr = listener.Addr().String()
	s.server = &http.Server{
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// Log but don't panic — the test will fail with a connection error.
			fmt.Printf("sneakystack: serve error: %v\n", err)
		}
	}()

	return "http://" + s.addr, nil
}

// Stop shuts down the proxy server.
func (s *Sidecar) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}

// Healthy returns true when the sidecar is ready to accept traffic.
func (s *Sidecar) Healthy(ctx context.Context) bool {
	if s.addr == "" {
		return false
	}

	dialer := &net.Dialer{Timeout: time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return false
	}

	return conn.Close() == nil
}
