# Custom Image and Pro Edition

libtftest defaults to the LocalStack Community (OSS) image. You can override
this for Pro, airgapped mirrors, or custom images.

## Using LocalStack Pro

LocalStack ships a **single image** (calendar-versioned, e.g.
`localstack/localstack:2026.06.1`); there is no separate `-pro` image. Pro
features are unlocked at runtime by setting `LOCALSTACK_AUTH_TOKEN` in the
environment. `Edition: localstack.EditionPro` just declares the intent so
Pro-only assertions run instead of auto-skipping — it does not change the
image.

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    iamassert "github.com/donaldgifford/libtftest/assert/iam"
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
    iamassert.RoleExists(t, tc.AWS(), tc.Output("role_name"))
}
```

## Environment Variable Override

Set `LIBTFTEST_LOCALSTACK_IMAGE` to override the image for all tests:

```bash
# Pin the unified single image (calendar-versioned; needs LOCALSTACK_AUTH_TOKEN)
LIBTFTEST_LOCALSTACK_IMAGE=localstack/localstack:2026.06.1 \
    go test -tags=integration -v ./test/...

# Pin the token-free community image (no account required)
LIBTFTEST_LOCALSTACK_IMAGE=localstack/localstack:4.14 \
    go test -tags=integration -v ./test/...

# Use an airgapped mirror
LIBTFTEST_LOCALSTACK_IMAGE=registry.internal/localstack:2026.06.1 \
    go test -tags=integration -v ./test/...
```

## Per-Test Image Override

```go
tc := libtftest.New(t, &libtftest.Options{
    Image:     "registry.internal/localstack:2026.06.1", // e.g. an airgapped mirror
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
3. Token-aware default:
   - With `LOCALSTACK_AUTH_TOKEN`: `localstack/localstack:2026.06.1` — the
     unified single image (also unlocks Pro).
   - Without a token: `localstack/localstack:4.14` — the last token-free
     community image. (The single image won't boot without a token.)
