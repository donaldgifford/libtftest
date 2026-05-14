package dockerx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ErrDockerUnavailable indicates the Docker daemon is not reachable.
var ErrDockerUnavailable = errors.New("docker daemon unavailable")

// Ping verifies the Docker daemon is reachable by sending an HTTP request
// to /_ping on the Docker socket. Returns nil if the daemon responds, or a
// classified error with remediation hints if not.
func Ping(ctx context.Context) error {
	socketPath := dockerSocket()

	dialer := &net.Dialer{Timeout: 2 * time.Second}

	// Quick check: can we connect to the socket at all?
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return classifyError(err, socketPath)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close docker socket probe: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/_ping", http.NoBody)
	if err != nil {
		return fmt.Errorf("build docker ping request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return classifyError(err, socketPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: /_ping returned %d", ErrDockerUnavailable, resp.StatusCode)
	}

	return nil
}

// dockerSocket returns the Docker socket path, honoring DOCKER_HOST if set.
func dockerSocket() string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		// Strip "unix://" prefix if present.
		if len(host) > 7 && host[:7] == "unix://" {
			return host[7:]
		}
		return host
	}

	if runtime.GOOS == "linux" {
		return "/var/run/docker.sock"
	}

	// macOS: check common socket locations.
	home := os.Getenv("HOME")
	candidates := []string{
		filepath.Join(home, ".colima", "default", "docker.sock"),
		filepath.Join(home, ".rd", "docker.sock"),
		"/var/run/docker.sock",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil { //nolint:gosec // Docker socket paths are under the user's home by design.
			return c
		}
	}

	return "/var/run/docker.sock"
}

// classifyError wraps a connection error with actionable remediation hints.
func classifyError(err error, socketPath string) error {
	if os.IsNotExist(err) || isSocketNotFound(err) {
		return fmt.Errorf(
			"%w: socket not found at %s\n\nRemediation:\n"+
				"  - macOS: run 'colima start' or start Rancher Desktop\n"+
				"  - Linux: run 'sudo systemctl start docker'\n"+
				"  - Custom socket: set DOCKER_HOST=unix:///path/to/docker.sock\n"+
				"  - Testcontainers override: set TESTCONTAINERS_HOST_OVERRIDE",
			ErrDockerUnavailable, socketPath,
		)
	}

	if os.IsPermission(err) || isPermissionDenied(err) {
		return fmt.Errorf(
			"%w: permission denied on %s\n\nRemediation:\n"+
				"  - Add your user to the 'docker' group: sudo usermod -aG docker $USER\n"+
				"  - Or run with sudo (not recommended for tests)",
			ErrDockerUnavailable, socketPath,
		)
	}

	return fmt.Errorf(
		"%w: %v\n\nRemediation:\n"+
			"  - Verify Docker is running: docker info\n"+
			"  - macOS: run 'colima start' or start Rancher Desktop\n"+
			"  - Custom socket: set DOCKER_HOST=unix:///path/to/docker.sock",
		ErrDockerUnavailable, err,
	)
}

func isSocketNotFound(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return os.IsNotExist(opErr.Err)
	}
	return false
}

func isPermissionDenied(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return os.IsPermission(opErr.Err)
	}
	return false
}
