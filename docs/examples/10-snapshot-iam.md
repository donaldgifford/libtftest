# Snapshot Testing for IAM Policies

> Runnable counterpart: `Test_Example10_SnapshotIAM` in
> [`examples_integration_test.go`](examples_integration_test.go).

`assert/snapshot` provides JSON snapshot testing — a simple "compare
the JSON I produce against the JSON I committed last time" loop with
an `UPDATE_SNAPSHOTS=1` rewrite protocol. The killer use case is
**locking down IAM policy shapes**: small mistakes in policy
generation are common, hard to spot in diff review, and silently
expand privilege.

The package ships two helpers for getting the JSON to snapshot out of
a `terraform show -json plan.out` payload:

| Helper                                     | What it pulls out                                                          |
| ------------------------------------------ | -------------------------------------------------------------------------- |
| `snapshot.ExtractIAMPolicies(planJSON)`    | Every IAM-policy-bearing resource (role assume policy, inline, managed, standalone) |
| `snapshot.ExtractResourceAttribute(...)`   | Any single attribute by resource address + dot-notation path               |

And two snapshot comparison helpers:

| Helper                                            | When to use                                                          |
| ------------------------------------------------- | -------------------------------------------------------------------- |
| `snapshot.JSONStrict(tb, actual, path)`           | Byte-for-byte. Use when key order is semantically meaningful.        |
| `snapshot.JSONStructural(tb, actual, path)`       | Normalizes keys + whitespace. Use for IAM policies, plan JSON, etc.  |

## Pinning an IAM role's assume role policy

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/assert/snapshot"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestEKSNodeRole_AssumeRolePolicyLocked(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/eks-node-role",
    })

    plan := tc.Plan()

    policies, err := snapshot.ExtractIAMPolicies(plan.JSON)
    if err != nil {
        t.Fatal(err)
    }

    snapshot.JSONStructural(t,
        policies["aws_iam_role.eks_node.assume_role"],
        "testdata/snapshots/eks_node_assume_role.json")
}
```

Run once with `LIBTFTEST_UPDATE_SNAPSHOTS=1` to create the snapshot:

```bash
LIBTFTEST_UPDATE_SNAPSHOTS=1 go test -tags=integration -v -run TestEKSNodeRole_AssumeRolePolicyLocked
```

Then commit `testdata/snapshots/eks_node_assume_role.json`. On every
subsequent run, the test fails if the policy shape changes.

## Pinning a KMS key policy via the generic extractor

```go
func TestKMSKey_PolicyLocked(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/kms-key",
    })

    plan := tc.Plan()

    policy, err := snapshot.ExtractResourceAttribute(
        plan.JSON,
        "aws_kms_key.main",
        "policy",
    )
    if err != nil {
        t.Fatal(err)
    }

    snapshot.JSONStructural(t, policy, "testdata/snapshots/kms_main_policy.json")
}
```

## Managed-policy attachments render as ARNs, not live documents

`ExtractIAMPolicies` deliberately does NOT fetch the live policy
document for `aws_iam_role_policy_attachment` resources — the ARN is
effectively an enum AWS owns (or you own), and fetching the live
document would make the test network-dependent and non-deterministic.

The map key encodes the ARN so the attachment itself is the test:

```
aws_iam_role_policy_attachment.eks_node_worker.managed:arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy
```

The bytes for that key are simply the ARN string. If your module
suddenly attaches a different managed policy, the key changes and the
diff is immediate.

For customer-managed policies whose **documents** also need locking,
those documents live under `aws_iam_policy.<name>.policy` and get
their own snapshot entry — `ExtractIAMPolicies` pulls them at the
same time.

## The update protocol

Setting `LIBTFTEST_UPDATE_SNAPSHOTS=1`:

- Missing snapshot → writes the actual payload, passes the test, logs
  via `tb.Logf`
- Mismatched snapshot → overwrites the file, passes, logs
- Anything else → behaves exactly like a normal snapshot run

The pattern is: **always commit the regenerated snapshots — they ARE
the test**. CI should never run in update mode; it would silently
accept policy changes.
