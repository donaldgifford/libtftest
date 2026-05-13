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
| [Cancellation and Deadlines](07-cancellation.md) | Per-call deadlines via the `*Context` API variants |
| [Idempotency Assertions](08-idempotency.md) | `tc.AssertIdempotent` and `tc.AssertIdempotentApply` |
| [Tag Propagation](09-tag-propagation.md) | `tagsassert.PropagatesFromRoot` via the Resource Groups Tagging API |

## Running the Examples

These markdown files are the canonical examples. Their behavior is verified
by [`examples_integration_test.go`](examples_integration_test.go), which
mirrors the snippets one-to-one and runs them end-to-end against
LocalStack.

```bash
# From the libtftest repo root:
make test-examples
# Or directly:
go test -tags=integration_examples -v ./docs/examples/...
```

To try them in a consumer repo, copy the code into a `test/` directory
alongside your Terraform module and run:

```bash
go test -tags=integration -v ./test/...
```

**Keeping markdown + tests in sync.** When you edit a snippet in a
markdown example, the corresponding `Test_ExampleNN_*` function in
`examples_integration_test.go` should match. CI runs the test file on
every PR, so silent drift will fail the build.

## Claude Code Skills

The [`libtftest` plugin](https://github.com/donaldgifford/claude-skills/tree/main/plugins/libtftest)
in `donaldgifford/claude-skills` automates everything in these examples.
Install with:

```bash
claude plugin install donaldgifford/claude-skills:libtftest
```

| Skill                          | What it does                                                              |
| ------------------------------ | ------------------------------------------------------------------------- |
| `tftest`                       | Umbrella — loads libtftest mental model, runs version-detection check     |
| `tftest:scaffold`              | Bootstrap a `test/` directory (go.mod, TestMain, starter test, .gitignore) |
| `tftest:setup-ci`              | Wire the reusable libtftest GitHub Actions workflow                       |
| `tftest:add-test`              | Add a new `Test*` function                                                |
| `tftest:add-fixture`           | Insert a `fixtures/<service>.Seed*` call before `tc.Apply()`              |
| `tftest:add-assertion`         | Insert an `assert/<service>.*` call after `tc.Apply()`                    |
| `tftest:debug`                 | Triage failing/flaky libtftest tests                                      |
| `tftest:enable-pro`            | Switch a suite to LocalStack Pro                                          |
| `tftest:enable-sneakystack`    | Add the sneakystack sidecar for SSO/Orgs/CT services                      |
| `tftest:upgrade`               | Upgrade libtftest with mechanical migrations                              |
| `tftest-reviewer` (agent)      | Review consumer test code for parallel-safety, cleanup, edition gating    |

The plugin's umbrella skill checks the installed libtftest version
against its supported range on activation and warns the model when
they're out of sync.
