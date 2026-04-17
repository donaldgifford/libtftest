package dockerx

import (
	"errors"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestClassifyError_SocketNotFound(t *testing.T) {
	t.Parallel()

	// Simulate a "no such file" error wrapped in a net.OpError.
	inner := &os.PathError{Op: "dial", Path: "/nonexistent.sock", Err: os.ErrNotExist}
	opErr := &net.OpError{Op: "dial", Net: "unix", Addr: nil, Err: inner}

	got := classifyError(opErr, "/nonexistent.sock")

	if !errors.Is(got, ErrDockerUnavailable) {
		t.Errorf("classifyError() = %v, want ErrDockerUnavailable", got)
	}
	if !strings.Contains(got.Error(), "socket not found") {
		t.Errorf("classifyError() = %q, want 'socket not found' in message", got)
	}
	if !strings.Contains(got.Error(), "colima start") {
		t.Errorf("classifyError() = %q, want 'colima start' remediation", got)
	}
}

func TestClassifyError_PermissionDenied(t *testing.T) {
	t.Parallel()

	inner := &os.PathError{Op: "dial", Path: "/var/run/docker.sock", Err: os.ErrPermission}
	opErr := &net.OpError{Op: "dial", Net: "unix", Addr: nil, Err: inner}

	got := classifyError(opErr, "/var/run/docker.sock")

	if !errors.Is(got, ErrDockerUnavailable) {
		t.Errorf("classifyError() = %v, want ErrDockerUnavailable", got)
	}
	if !strings.Contains(got.Error(), "permission denied") {
		t.Errorf("classifyError() = %q, want 'permission denied' in message", got)
	}
	if !strings.Contains(got.Error(), "docker") {
		t.Errorf("classifyError() = %q, want 'docker' group remediation", got)
	}
}

func TestClassifyError_GenericError(t *testing.T) {
	t.Parallel()

	got := classifyError(errors.New("connection refused"), "/var/run/docker.sock")

	if !errors.Is(got, ErrDockerUnavailable) {
		t.Errorf("classifyError() = %v, want ErrDockerUnavailable", got)
	}
	if !strings.Contains(got.Error(), "connection refused") {
		t.Errorf("classifyError() = %q, want original error in message", got)
	}
}

func TestDockerSocket_RespectsDockerHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "unix:///custom/docker.sock")

	got := dockerSocket()
	want := "/custom/docker.sock"

	if got != want {
		t.Errorf("dockerSocket() = %q, want %q", got, want)
	}
}

func TestDockerSocket_DefaultPath(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")

	got := dockerSocket()

	if runtime.GOOS == "linux" {
		if got != "/var/run/docker.sock" {
			t.Errorf("dockerSocket() = %q, want /var/run/docker.sock on linux", got)
		}
	} else {
		// On macOS, it could be any of the candidates or the fallback.
		if got == "" {
			t.Error("dockerSocket() returned empty string")
		}
	}
}
