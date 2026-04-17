# Custom Image and Pro Edition

libtftest defaults to the LocalStack Community (OSS) image. You can override
this for Pro, airgapped mirrors, or custom images.

## Using LocalStack Pro

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/assert"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestIAMRole_ProEdition(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionPro,
        ModuleDir: "../../modules/iam-role",
    })
    tc.SetVar("role_name", tc.Prefix()+"-admin")

    tc.Apply()

    // IAM assertions auto-skip on Community edition.
    // On Pro, they verify real IAM enforcement.
    assert.IAM.RoleExists(t, tc.AWS(), tc.Output("role_name"))
}
```

## Environment Variable Override

Set `LIBTFTEST_LOCALSTACK_IMAGE` to override the image for all tests:

```bash
# Use a specific Pro version
LIBTFTEST_LOCALSTACK_IMAGE=localstack/localstack-pro:4.4 \
    go test -tags=integration -v ./test/...

# Use an airgapped mirror
LIBTFTEST_LOCALSTACK_IMAGE=registry.internal/localstack:4.4 \
    go test -tags=integration -v ./test/...
```

## Per-Test Image Override

```go
tc := libtftest.New(t, &libtftest.Options{
    Image:     "localstack/localstack-pro:4.4",
    ModuleDir: "../../modules/my-module",
})
```

## RequirePro Auto-Skip

Pro-only assertions call `RequirePro(t)` internally. On Community edition,
the test is skipped with a clear message instead of failing:

```
--- SKIP: TestIAMRole_ProEdition (0.00s)
    libtftest.go:339: skipping: requires LocalStack Pro (no LOCALSTACK_AUTH_TOKEN set)
```

You can also call it explicitly at the top of a test:

```go
func TestSomethingProOnly(t *testing.T) {
    libtftest.RequirePro(t)  // Skips if not Pro

    // ... Pro-only test logic
}
```

## Image Resolution Order

1. `Options.Image` (explicit per-test override)
2. `LIBTFTEST_LOCALSTACK_IMAGE` environment variable
3. Default based on edition:
   - Community: `localstack/localstack:4.4`
   - Pro: `localstack/localstack-pro:4.4`
