package localstack

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	dockercontainer "github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/donaldgifford/libtftest/internal/dockerx"
)

const (
	// defaultCommunityImage is the token-free LocalStack image used when no
	// LOCALSTACK_AUTH_TOKEN is configured. LocalStack's unified single image
	// (defaultImage) requires an auth token even for free-tier use, so
	// zero-config community runs stay on the last token-free legacy tag.
	defaultCommunityImage = "localstack/localstack:4.14"
	// defaultImage is the pinned LocalStack single image, used when an auth
	// token is available. LocalStack ships one unified, calendar-versioned
	// image (YYYY.MM.patch); Pro features unlock at runtime via the token,
	// not a separate localstack-pro image. Renovate keeps this pin current
	// (see the customManager in renovate.json5).
	defaultImage   = "localstack/localstack:2026.07.0"
	edgePort       = "4566/tcp"
	imageEnv       = "LIBTFTEST_LOCALSTACK_IMAGE"
	startupTimeout = 90 * time.Second
)

// Container holds a running LocalStack container and its metadata.
type Container struct {
	ID       string
	EdgeURL  string
	Edition  Edition
	Services map[string]string
	ctr      testcontainers.Container
}

// Config configures a LocalStack container.
type Config struct {
	Edition   Edition
	Image     string
	Services  []string
	AuthToken string
	InitHooks []InitHook
}

// ResolveImage returns the container image to use, checking (in order):
// Config.Image, the LIBTFTEST_LOCALSTACK_IMAGE env var, then an
// authentication-dependent default. LocalStack's unified single image
// requires an auth token even for free-tier use, so when no token is
// configured ResolveImage falls back to the last token-free community image
// (defaultCommunityImage); with a token it uses the single image
// (defaultImage), which also unlocks Pro features.
func (c *Config) ResolveImage() string {
	if c.Image != "" {
		return c.Image
	}

	if img := os.Getenv(imageEnv); img != "" {
		return img
	}

	if c.hasAuthToken() {
		return defaultImage
	}

	return defaultCommunityImage
}

// hasAuthToken reports whether a LocalStack auth token is configured, either
// on the Config or via the LOCALSTACK_AUTH_TOKEN environment variable.
func (c *Config) hasAuthToken() bool {
	return c.AuthToken != "" || os.Getenv("LOCALSTACK_AUTH_TOKEN") != ""
}

// Env builds the environment variable map for the container.
func (c *Config) Env() map[string]string {
	env := map[string]string{
		"DEBUG": "1",
	}

	if c.AuthToken != "" {
		env["LOCALSTACK_AUTH_TOKEN"] = c.AuthToken
	} else if token := os.Getenv("LOCALSTACK_AUTH_TOKEN"); token != "" {
		env["LOCALSTACK_AUTH_TOKEN"] = token
	}

	if len(c.Services) > 0 {
		env["SERVICES"] = strings.Join(c.Services, ",")
	}

	return env
}

// Start creates and starts a LocalStack container. It pre-checks that Docker
// is available, then uses testcontainers.Run with functional options.
// The container is considered ready only when AllServicesReady returns true
// on the /_localstack/health endpoint.
func Start(ctx context.Context, cfg *Config) (*Container, error) {
	if err := dockerx.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker pre-check: %w", err)
	}

	edition := DetectEdition(cfg.Edition)
	image := cfg.ResolveImage()

	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithExposedPorts(edgePort),
		testcontainers.WithEnv(cfg.Env()),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/_localstack/health").
				WithPort(edgePort).
				WithStartupTimeout(startupTimeout).
				WithResponseMatcher(AllServicesReady),
		),
	}

	if len(cfg.InitHooks) > 0 {
		hookDir, err := WriteInitHooks(cfg.InitHooks)
		if err != nil {
			return nil, fmt.Errorf("write init hooks: %w", err)
		}

		opts = append(opts, testcontainers.WithHostConfigModifier(
			func(hc *dockercontainer.HostConfig) {
				hc.Binds = append(hc.Binds,
					hookDir+":/etc/localstack/init/ready.d:ro",
				)
			},
		))
	}

	ctr, err := testcontainers.Run(ctx, image, opts...)
	if err != nil {
		return nil, fmt.Errorf("start localstack container: %w", err)
	}

	endpoint, err := ctr.PortEndpoint(ctx, edgePort, "http")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr) //nolint:errcheck // Best-effort cleanup on failure path.
		return nil, fmt.Errorf("get container endpoint: %w", err)
	}

	return &Container{
		ID:       ctr.GetContainerID(),
		EdgeURL:  endpoint,
		Edition:  edition,
		Services: make(map[string]string),
		ctr:      ctr,
	}, nil
}

// Stop terminates the container.
func (c *Container) Stop(_ context.Context) error {
	if c.ctr == nil {
		return nil
	}

	return testcontainers.TerminateContainer(c.ctr)
}

// Endpoint returns the HTTP edge URL for the running container.
func (c *Container) Endpoint() string {
	return c.EdgeURL
}
