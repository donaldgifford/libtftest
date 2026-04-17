# sneakystack Sidecar

sneakystack is a gap-filling HTTP proxy that sits between libtftest and
LocalStack. It handles AWS services that LocalStack doesn't support (or
supports poorly) and forwards everything else through to LocalStack.

## Using sneakystack with the Harness

The easiest way to use sneakystack is as a sidecar in `TestMain`:

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/harness"
    "github.com/donaldgifford/libtftest/localstack"
    "github.com/donaldgifford/libtftest/sneakystack"
)

func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition: localstack.EditionAuto,
        Sidecars: []harness.Sidecar{
            sneakystack.NewSidecar(sneakystack.Config{
                Services: []string{"sso-admin", "organizations"},
            }),
        },
    })
}

func TestPermissionSet(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        ModuleDir: "../../modules/permission-set",
    })
    tc.SetVar("name", tc.Prefix()+"-admin-access")

    tc.Apply()
    // Assertions against SSO Admin resources handled by sneakystack
}
```

## How It Works

1. `harness.Run` starts LocalStack first
2. sneakystack starts as an in-process goroutine, pointing at LocalStack
3. All AWS SDK and Terraform traffic routes through sneakystack
4. Requests for registered services (SSO Admin, Organizations) are handled
   locally by sneakystack against an in-memory store
5. All other requests pass through to LocalStack unmodified

## Standalone Docker Container

sneakystack can also run as a standalone container for non-Go test frameworks
or manual testing:

```bash
# Run sneakystack pointing at a LocalStack container
docker run --rm -p 4567:4567 \
    ghcr.io/donaldgifford/sneakystack:latest \
    --downstream http://host.docker.internal:4566

# Or use the binary directly
sneakystack --downstream http://localhost:4566 --port 4567
```

Then point your tests at sneakystack instead of LocalStack:

```bash
LIBTFTEST_CONTAINER_URL=http://localhost:4567 \
    go test -tags=integration -v ./test/...
```

## sneakystack Architecture

```
Test Code
    |
    v
sneakystack (HTTP proxy)
    |
    ├── SSO Admin requests     -> in-memory Store
    ├── Organizations requests -> in-memory Store
    └── Everything else        -> LocalStack container
```

The `Store` interface abstracts the persistence layer. The default `MapStore`
uses plain Go maps with `sync.RWMutex`. No external database required.

## Store Interface

```go
type Store interface {
    Put(ctx context.Context, kind, id string, obj any) error
    Get(ctx context.Context, kind, id string) (any, error)
    List(ctx context.Context, kind string, filter Filter) ([]any, error)
    Delete(ctx context.Context, kind, id string) error
}
```

Service handlers wrap `Store` with typed accessors for their specific
resource types.
