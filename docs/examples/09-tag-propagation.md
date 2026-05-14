# Tag Propagation Assertions

> Runnable counterpart: `Test_Example09_TagPropagation` in
> [`examples_integration_test.go`](examples_integration_test.go).

Many Terraform modules carry a `tags` variable that's merged onto every
resource the module creates. Verifying that propagation works — without
writing a per-service assertion for every resource type — is what
`assert/tags` exists for. It calls the AWS Resource Groups Tagging API
once and checks a baseline tag map against all of the listed ARNs in
parallel.

The subset semantics are deliberate: the baseline is "every tag the
module *must* set"; extra tags on the resource (e.g. AWS-managed
`CreatedBy`) don't fail the check.

## Basic usage

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    tagsassert "github.com/donaldgifford/libtftest/assert/tags"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Module_PropagatesTags(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-tagged")
    tc.SetVar("tags", map[string]string{
        "Owner": "platform",
        "Env":   "test",
    })

    tc.Apply()

    bucketARN := tc.Output("bucket_arn")

    tagsassert.PropagatesFromRoot(t, tc.AWS(), map[string]string{
        "Owner": "platform",
        "Env":   "test",
    }, bucketARN)
}
```

## Aggregated failure messages

When the assertion fails, every missing or mismatched tag is reported
in a single `t.Errorf` call. Example output:

```
PropagatesFromRoot: 3 tag problem(s):
  - arn:aws:s3:::my-bucket: missing tag "Env" (want "prod")
  - arn:aws:s3:::my-bucket: tag "Owner" = "team-y", want "team-x"
  - arn:aws:s3:::other-bucket: not returned by GetResources
```

Aggregation is intentional — one CI run surfaces every propagation
defect, not just the first.

## With a deadline

The `*Context` variant accepts a caller-supplied `context.Context`:

```go
import (
    "context"
    "time"
)

func TestS3Module_PropagatesTagsWithDeadline(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-deadline-tags")
    tc.SetVar("tags", map[string]string{"Owner": "platform"})
    tc.Apply()

    arn := tc.Output("bucket_arn")

    ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
    defer cancel()

    tagsassert.PropagatesFromRootContext(t, ctx, tc.AWS(),
        map[string]string{"Owner": "platform"}, arn)
}
```

## When the Resource Groups Tagging API isn't enough

`assert/tags` is service-agnostic but the underlying AWS API has
limits:

- A handful of resource types don't show up in `GetResources` even
  when they carry tags (rare; check the AWS docs for the resource
  type in question).
- LocalStack OSS implements `GetResources`; coverage of less-common
  resource types is improving but may lag the official AWS Tagging
  API in some niche cases.

For those edge cases, fall back to a per-service assertion (e.g.
`s3assert.BucketHasTag(t, cfg, bucket, "Owner", "platform")`) for the
affected resource. The bulk-of-the-module case is what `assert/tags`
is for.
