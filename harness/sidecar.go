package harness

import "context"

// Sidecar is implemented by packages that provide auxiliary services
// (in-process or containerized) that sit between libtftest and LocalStack.
type Sidecar interface {
	// Start launches the sidecar with the given LocalStack edge URL as
	// its downstream target. Returns the URL callers should use instead
	// of the raw LocalStack URL.
	Start(ctx context.Context, localstackURL string) (edgeURL string, err error)

	// Stop shuts down the sidecar.
	Stop(ctx context.Context) error

	// Healthy returns true when the sidecar is ready to accept traffic.
	Healthy(ctx context.Context) bool
}
