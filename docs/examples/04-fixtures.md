# Fixtures and Seeding

Some modules expect resources to already exist before `terraform apply` runs.
The `fixtures` package seeds data into LocalStack and automatically cleans up
via `t.Cleanup`.

## Seed an SSM Parameter Before Apply

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    ssmassert "github.com/donaldgifford/libtftest/assert/ssm"
    s3fix "github.com/donaldgifford/libtftest/fixtures/s3"
    secretsfix "github.com/donaldgifford/libtftest/fixtures/secretsmanager"
    ssmfix "github.com/donaldgifford/libtftest/fixtures/ssm"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestModule_ReadsSSMParameter(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/app-config",
        Services:  []string{"s3", "ssm"},
    })

    // Seed an SSM parameter that the module reads during apply.
    paramName := "/" + tc.Prefix() + "/config/db-host"
    ssmfix.SeedParameter(t, tc.AWS(), paramName, "db.example.com", false)

    tc.SetVar("config_prefix", "/"+tc.Prefix()+"/config")
    tc.Apply()

    // Verify the module read the parameter correctly.
    ssmassert.ParameterHasValue(t, tc.AWS(), paramName, "db.example.com")
}
```

## Seed an S3 Object

```go
func TestModule_ProcessesUploadedFile(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/file-processor",
        Services:  []string{"s3", "lambda"},
    })

    bucketName := tc.Prefix() + "-input"
    // Assume the bucket is created by the module; seed an object after apply.
    tc.SetVar("input_bucket", bucketName)
    tc.Apply()

    s3fix.SeedObject(t, tc.AWS(), bucketName, "test/input.json",
        []byte(`{"key": "value"}`))

    // ... assert on processing results
}
```

## Seed a Secret

```go
func TestModule_UsesSecret(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/app",
        Services:  []string{"secretsmanager"},
    })

    secretName := tc.Prefix() + "/api-key"
    secretsfix.SeedSecret(t, tc.AWS(), secretName, "sk-test-12345")

    tc.SetVar("api_key_secret_name", secretName)
    tc.Apply()
}
```

## Available Fixtures

Every `Seed*` function has a paired `Seed*Context` variant that accepts
a `context.Context` as the second argument. The non-context variants are
shims that pass `tb.Context()`.

| Package | Function | Context variant | What it seeds | Cleanup |
| --- | --- | --- | --- | --- |
| `fixtures/s3` (alias `s3fix`) | `SeedObject(tb, cfg, bucket, key, body)` | `SeedObjectContext(tb, ctx, ...)` | S3 object | Deletes the object |
| `fixtures/ssm` (alias `ssmfix`) | `SeedParameter(tb, cfg, name, value, secure)` | `SeedParameterContext(tb, ctx, ...)` | SSM parameter (String or SecureString) | Deletes the parameter |
| `fixtures/secretsmanager` (alias `secretsfix`) | `SeedSecret(tb, cfg, name, value)` | `SeedSecretContext(tb, ctx, ...)` | Secrets Manager secret | Force-deletes the secret |
| `fixtures/sqs` (alias `sqsfix`) | `SeedMessage(tb, cfg, queueURL, body)` | `SeedMessageContext(tb, ctx, ...)` | SQS message | None (consumed by test) |

Cleanup callbacks use `context.WithoutCancel(ctx)` so they survive
test-end cancellation.

## With caller-supplied context

```go
ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
defer cancel()

s3fix.SeedObjectContext(t, ctx, tc.AWS(), bucket, "k", []byte("v"))
```
