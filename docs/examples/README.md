# Examples

Usage examples for libtftest and sneakystack. Each example is a self-contained
Go test file that demonstrates a specific testing pattern.

## Getting Started

All examples assume you have:

1. A Terraform module under `modules/` in your repo
2. A `test/` directory with `go.mod` importing `github.com/donaldgifford/libtftest`
3. Docker running (for LocalStack containers)
4. Terraform CLI installed

Typical module repo structure:

```
terraform-aws-my-module/
├── main.tf
├── variables.tf
├── outputs.tf
└── test/
    ├── go.mod
    ├── go.sum
    ├── main_test.go          # TestMain with harness.Run
    └── my_module_test.go     # Your tests
```

## Examples

| Example | Description |
| --- | --- |
| [Basic S3 Module Test](01-basic-s3-test.md) | Minimal test for an S3 bucket module |
| [Shared Container with TestMain](02-shared-container.md) | Share one LocalStack container across all tests |
| [Plan-Only Testing](03-plan-testing.md) | Assert on planned changes without applying |
| [Fixtures and Seeding](04-fixtures.md) | Pre-seed data before terraform apply |
| [Custom Image and Pro Edition](05-custom-image.md) | Use Pro, airgapped mirrors, or custom images |
| [sneakystack Sidecar](06-sneakystack.md) | Fill LocalStack gaps with sneakystack |

## Running the Examples

These are documentation examples, not runnable test files. To try them, copy
the code into a `test/` directory alongside your Terraform module and run:

```bash
go test -tags=integration -v ./test/...
```
