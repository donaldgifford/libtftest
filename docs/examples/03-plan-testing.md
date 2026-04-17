# Plan-Only Testing

Use `tc.Plan()` to assert on planned changes without running `terraform apply`.
This is useful for:

- Modules whose apply takes too long even on LocalStack
- Golden-file testing (diff plan output against checked-in baselines)
- Catching unexpected destroys before they happen

## Plan and Assert on Changes

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Module_PlanCreatesResources(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-plan-test")

    result := tc.Plan()

    // Verify the plan creates resources (bucket + versioning).
    if result.Changes.Add < 2 {
        t.Errorf("Plan.Changes.Add = %d, want >= 2", result.Changes.Add)
    }

    // Verify nothing is being destroyed.
    if result.Changes.Destroy > 0 {
        t.Errorf("Plan.Changes.Destroy = %d, want 0", result.Changes.Destroy)
    }

    t.Logf("Plan: +%d ~%d -%d",
        result.Changes.Add, result.Changes.Change, result.Changes.Destroy)
}
```

## Error-Returning Variant

Use `PlanE` for negative tests where you expect the plan to fail:

```go
func TestInvalidModule_PlanFails(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/broken-module",
    })

    _, err := tc.PlanE()
    if err == nil {
        t.Error("PlanE() succeeded, want error for broken module")
    }
}
```

## PlanResult Fields

| Field | Type | Description |
| --- | --- | --- |
| `JSON` | `[]byte` | Raw `terraform show -json` output |
| `FilePath` | `string` | Path to the binary plan file |
| `Changes.Add` | `int` | Resources to create |
| `Changes.Change` | `int` | Resources to update in-place |
| `Changes.Destroy` | `int` | Resources to destroy |
