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
    "github.com/donaldgifford/libtftest/assert"
    "github.com/donaldgifford/libtftest/fixtures"
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
    fixtures.SeedSSMParameter(t, tc.AWS(), paramName, "db.example.com", false)

    tc.SetVar("config_prefix", "/"+tc.Prefix()+"/config")
    tc.Apply()

    // Verify the module read the parameter correctly.
    assert.SSM.ParameterHasValue(t, tc.AWS(), paramName, "db.example.com")
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

    fixtures.SeedS3Object(t, tc.AWS(), bucketName, "test/input.json",
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
    fixtures.SeedSecret(t, tc.AWS(), secretName, "sk-test-12345")

    tc.SetVar("api_key_secret_name", secretName)
    tc.Apply()
}
```

## Available Fixtures

Every `Seed*` function has a paired `Seed*Context` variant that accepts
a `context.Context` as the second argument. The non-context variants are
shims that pass `tb.Context()`.

| Function | Context variant | What it seeds | Cleanup |
| --- | --- | --- | --- |
| `SeedS3Object(tb, cfg, bucket, key, body)` | `SeedS3ObjectContext(tb, ctx, ...)` | S3 object | Deletes the object |
| `SeedSSMParameter(tb, cfg, name, value, secure)` | `SeedSSMParameterContext(tb, ctx, ...)` | SSM parameter (String or SecureString) | Deletes the parameter |
| `SeedSecret(tb, cfg, name, value)` | `SeedSecretContext(tb, ctx, ...)` | Secrets Manager secret | Force-deletes the secret |
| `SeedSQSMessage(tb, cfg, queueURL, body)` | `SeedSQSMessageContext(tb, ctx, ...)` | SQS message | None (consumed by test) |

Cleanup callbacks use `context.WithoutCancel(ctx)` so they survive
test-end cancellation.

## With caller-supplied context

```go
ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
defer cancel()

fixtures.SeedS3ObjectContext(t, ctx, tc.AWS(), bucket, "k", []byte("v"))
```
