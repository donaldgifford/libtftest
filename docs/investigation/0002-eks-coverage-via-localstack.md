---
id: INV-0002
title: "EKS coverage via LocalStack"
status: Concluded
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0002: EKS coverage via LocalStack

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Environment](#environment)
- [Architecture](#architecture)
  - [LocalStack uses k3d, not raw k3s](#localstack-uses-k3d-not-raw-k3s)
  - [Do not combine the testcontainers-go k3s module with LocalStack EKS](#do-not-combine-the-testcontainers-go-k3s-module-with-localstack-eks)
  - [Requirements](#requirements)
- [Coverage Matrix](#coverage-matrix)
  - [Core resources](#core-resources)
  - [Auth and access depth](#auth-and-access-depth)
  - [Security posture](#security-posture)
  - [Module hygiene](#module-hygiene)
- [Explicitly out of scope for LocalStack](#explicitly-out-of-scope-for-localstack)
- [Snapshot-test IAM policies (LocalStack-independent)](#snapshot-test-iam-policies-localstack-independent)
- [Recommended LocalStack config](#recommended-localstack-config)
- [Open questions](#open-questions)
- [Package layout decision](#package-layout-decision)
  - [Options considered](#options-considered)
  - [Decision: Option A2](#decision-option-a2)
  - [Why not also refactor awsx/?](#why-not-also-refactor-awsx)
  - [Migration shape](#migration-shape)
- [Recommendation](#recommendation)
  - [Sequencing](#sequencing)
  - [1. Package layout: assert/{service}/, fixtures/{service}/](#1-package-layout-assertservice-fixturesservice)
  - [2. TestCase.AssertIdempotent(ctx)](#2-testcaseassertidempotentctx)
  - [3. assert/tags package](#3-asserttags-package)
  - [4. assert/snapshot package](#4-assertsnapshot-package)
  - [What this investigation does NOT recommend](#what-this-investigation-does-not-recommend)
  - [Next steps](#next-steps)
- [References](#references)
<!--toc:end-->

## Question

What does it take to make libtftest a viable harness for integration-testing
**Terraform EKS modules** against LocalStack — without libtftest itself
shipping the EKS test suite — which of the patterns surfaced fall out as
**generic libtftest features** rather than EKS-specific cookbook, and **does
the current flat `assert/` / `fixtures/` package layout still scale** when
we commit to 15+ AWS services?

## Hypothesis

LocalStack Pro CalVer (`2026.5.0.dev121` locally) covers enough of the EKS
control-plane surface (clusters, managed node groups, access entries, pod
identity associations, IRSA / OIDC provider) that a consumer can stand up
real integration coverage of their own EKS module using libtftest +
the existing primitives.

Three patterns the EKS coverage matrix surfaces are not EKS-specific and
would land cleaner as libtftest features (idempotency, tag propagation,
IAM policy snapshotting).

Separately: the current flat package layout (`assert/s3.go` +
`assert.S3.BucketExists` zero-size struct namespacing) does not scale
past ~10 services without becoming unwieldy. A move to per-service
sub-packages (Option A2 in the analysis below) is the right shape
before the next wave of services lands.

## Context

**Triggered by:** Loose notes in `docs/eks-notes.md` (deleted in this branch)
that sketched out test coverage for an in-house EKS module. The notes mixed
genuinely valuable architecture observations (k3d vs. k3s, pod identity trust
policy gotchas) with content that didn't have a clear home in the docz
structure. This investigation formalizes the analysis, captures the
LocalStack-version question, and isolates the three libtftest RFE candidates
that came out of the coverage matrix.

libtftest will **not** ship the EKS test framework itself. A consumer can
build the matrix below on top of libtftest's existing primitives once the
RFE candidates land (or sooner, with local helpers).

## Environment

| Component | Version / Value |
|-----------|-----------------|
| LocalStack Pro/Ultimate (local) | `localstack/localstack-pro:2026.5.0.dev121` (latest dev CalVer tag) |
| LocalStack OSS (CI default) | `localstack/localstack:2026.04.0` (CalVer applies to both Pro and OSS — earlier 4.x SemVer is retired) |
| k3d image tag | `EKS_K3S_IMAGE_TAG=v1.32.13-k3s1` (pin to the module's target K8s version) |
| Docker socket mount | required — LocalStack spawns k3d containers from inside its own container |
| `LOCALSTACK_AUTH_TOKEN` | required (EKS is not in Community) — libtftest's `RequirePro(t)` gate already handles auto-skip |
| libtftest gating | both `Edition` (Community vs. Pro) and the underlying image tag need to match — see "Edition gating" under Open Questions |

## Architecture

### LocalStack uses k3d, not raw k3s

LocalStack's EKS provider spins up an embedded **k3d** cluster (which itself
wraps k3s). You'll see two containers per cluster:

- `rancher/k3d-proxy:<ver>` — serverlb / API proxy
- `rancher/k3s:<ver>` — k3s server

Managed node groups are added via `k3d node create` (or `k3s agent`) using
the shared cluster token (`EKS_K3D_CLUSTER_TOKEN`). Mocked EC2 instance
records are created in parallel.

### Do not combine the testcontainers-go k3s module with LocalStack EKS

Two reasons:

1. LocalStack already manages its own k3d lifecycle. Adding a separate
   `testcontainers-go/modules/k3s` testcontainer is redundant.
2. The BYO-cluster mode (`EKS_K8S_PROVIDER=local` + mounted kubeconfig) does
   **not** support managed node group provisioning — that flow depends on
   k3d primitives (`k3d node create`, cluster token join). A raw k3s
   testcontainer breaks node group tests, which is exactly what you want
   to cover.

Use the k3s/k3d testcontainer only if you want a *separate* lightweight
cluster to deploy workloads into after the IaC test converges — and even
then, treat the two as independent test scopes.

### Requirements

- LocalStack **Pro/Ultimate** license (EKS is not in Community).
- Docker socket mounted into the LocalStack container so it can spawn
  k3d containers.
- `EKS_K3S_IMAGE_TAG` pinned to the Kubernetes version the module targets.
- Terraform AWS provider `endpoints` block pointing at LocalStack, plus
  `skip_credentials_validation`, `skip_metadata_api_check`,
  `skip_requesting_account_id`. (libtftest's `_libtftest_override.tf.json`
  injection already handles this — no consumer change required.)

## Coverage Matrix

### Core resources

- Cluster creation
- Managed node groups
- Access entries
- Pod identity associations
- IAM roles for pod identities

### Auth and access depth

- **`authentication_mode` matrix** (`API`, `API_AND_CONFIG_MAP`, `CONFIG_MAP`).
  Drives whether access entries are even honored. A wrong default silently
  breaks them.
- **`bootstrap_cluster_creator_admin_permissions` flag.** LocalStack 4.10+
  respects it and auto-creates the EKS service role access entry — assertable.
- **EKS Pod Identity Agent addon** installation. Pod identity associations
  are inert without it in real EKS. Test the IaC wiring; the runtime
  exchange won't fully work in LocalStack regardless.
- **Pod identity trust policy correctness.** Assert both:
  - `Principal.Service: pods.eks.amazonaws.com`
  - `Action: ["sts:AssumeRole", "sts:TagSession"]` — `TagSession` is the
    common omission and silently breaks ABAC.
- **IRSA / OIDC provider** (LocalStack 4.10+). Worth covering even alongside
  pod identity:
  - OIDC provider resource exists and points at the cluster issuer
  - Role trust policy `sub` claim regex matches
    `system:serviceaccount:<ns>:<sa>`

### Security posture

- **KMS envelope encryption** (`encryption_config`). Key exists, key policy
  allows the EKS service principal.
- **Control plane logging** (`enabled_cluster_log_types`): api, audit,
  authenticator, controllerManager, scheduler. Easy assertion, common
  audit finding.
- **Endpoint access config**: `endpoint_public_access`,
  `endpoint_private_access`, `public_access_cidrs`. Most "EKS exposed to
  internet" findings come from this.
- **Cluster security group** + any additional SGs the module attaches.
  Node-to-cluster ingress path.
- **IMDS hop limit / IMDSv2 enforcement** on the node group launch template.
  Pod identity only isolates from IMDS if pod access to IMDS is also
  restricted.

### Module hygiene

- **Idempotency** — `terraform apply` twice; the second plan must be empty.
  Catches bad `ignore_changes` and provider drift.
- **Clean destroy** — node groups + access entries + pod identity
  associations have ordering deps. Destroy is where most modules break.
- **Tag propagation** — root-level tags reach cluster, node groups, ASG /
  launch template.
- **Input validation** — invalid combos (e.g., access entries on
  `CONFIG_MAP`-only auth) should fail at plan, not apply.

## Explicitly out of scope for LocalStack

These either don't work or give false confidence. Test elsewhere (real EKS
sandbox, unit tests on rendered output, or skip entirely):

- Pod identity agent runtime credential injection / STS round-trip
- IRSA token exchange end-to-end
- VPC CNI / ENI-per-pod, security groups for pods
- Karpenter / cluster autoscaler actually provisioning capacity
- Node AMI / userdata behavior at runtime
- Cluster version upgrade paths end-to-end
- Fargate compute

## Snapshot-test IAM policies (LocalStack-independent)

Independent of LocalStack — render IAM trust + permission policies as JSON
via `terraform plan -out` → `terraform show -json`, extract the policy
documents, and snapshot-assert them. Catches:

- Missing `sts:TagSession` on pod identity trust policies
- Wrong OIDC `sub` regex on IRSA roles
- Overscoped `Resource: "*"`
- Missing condition keys (`aws:RequestedRegion`, `aws:PrincipalArn`)

Faster than spinning LocalStack and catches the things that actually cause
incidents.

## Recommended LocalStack config

```bash
# docker run / compose env
DEBUG=1
PERSISTENCE=0
EKS_K3S_IMAGE_TAG=v1.32.13-k3s1     # pin to module's target K8s version
EKS_K3D_CLUSTER_TOKEN=libtftest     # deterministic for test debugging
LAMBDA_EXECUTOR=docker-reuse        # if module touches Lambda too
LOCALSTACK_AUTH_TOKEN=<pro-token>

# required mounts
/var/run/docker.sock:/var/run/docker.sock
```

libtftest consumers can pass these via `Options.Env` (existing API) or set
them on the per-test container via `Options.Image` + `Options.Services`.

## Open questions

1. **LocalStack default image bump.** libtftest's current default is
   `localstack/localstack:4.4`. The OSS line is now on CalVer
   (`2026.04.0`), and Pro EKS features need at least the equivalent of
   the old 4.10+. Two distinct decisions:
   - **(a)** Bump the OSS default from `4.4` to `2026.04.0` for the
     non-Pro integration job (`integration-tests` in CI). Low risk —
     CalVer renaming aside, the upstream behavior is largely the same.
   - **(b)** Use `localstack/localstack-pro:2026.5.0.dev121` (or the
     stable Pro CalVer once dev is rotated) for Pro-gated tests in
     `integration-tests-pro`. Required for EKS coverage to mean anything.
   - Recommendation: do both bumps as a coordinated change — see the
     `libtftest:bump-localstack` skill playbook.

2. **Edition gating must also track image variant.** Today `RequirePro(t)`
   detects edition via the LocalStack health endpoint and skips
   community-mode tests. With CalVer, the *image name* now diverges
   too: `localstack/localstack` (OSS) vs.
   `localstack/localstack-pro` (Pro). Two implications:
   - Options should accept both — `Options.Image` covers this already.
   - libtftest's auto-pull logic in `localstack/` should not assume the
     suffix; verify `pull` works for both names.
   - `harness.Run`'s edition-aware default selection (if any) needs
     to pick the right base image. Today there's a single `DefaultImage`
     constant; consider a tuple `{Community: ..., Pro: ...}` or derive
     from `Edition`.

3. **Pro CI gating.** Existing `integration-tests-pro` job hits the OSS
   default with a `LOCALSTACK_AUTH_TOKEN`. If EKS coverage materializes
   upstream, that job needs to pin to `localstack/localstack-pro:<calver>`
   and the `localstack/lstk` CLI may be useful (`mise.toml` already has
   `lstk` available locally). Belongs to the `bump-localstack` playbook
   work, flagged here for context.

## Package layout decision

While working through the EKS coverage matrix it became clear that
libtftest's current flat layout doesn't scale past ~10 services. With
realistic forward-looking coverage spanning EKS, ECS, SNS, SQS, KMS,
Secrets Manager, EventBridge, Step Functions, API Gateway, CloudWatch,
Route53, RDS, and Cognito on top of the existing S3 / DynamoDB / IAM /
SSM / Lambda set, libtftest is committing to 15+ services. The current
zero-size struct namespacing trick (`assert.S3.BucketExists`) becomes
a 200-line file with one struct per service.

### Options considered

| Layout | Verdict |
|---|---|
| **Status quo** — `assert/s3.go` + zero-size struct namespacing | Doesn't scale past ~10 services; awkward godoc per service; growing single file per package |
| **Option A2** — per-service sub-packages: `assert/{service}/`, `fixtures/{service}/` | **Selected** |
| **Option B with interfaces** — `service/{name}/{assert,fixtures}.go` + interfaces in top-level | Rejected — no substitution surface justifies the interface layer; ceremony without payoff |
| **Option B without interfaces** — `service/{name}/{assert,fixtures}.go` co-located | Rejected — Go's package-as-namespace forces `eks.AssertCluster` / `eks.SeedCluster` disambiguation, losing the co-location win |

### Decision: Option A2

Adopt per-service sub-packages under the existing top-level package names:

```text
libtftest/
├── assert/
│   ├── s3/        — package s3: BucketExists, BucketExistsContext, ...
│   ├── dynamodb/  — package dynamodb: TableExists, ...
│   ├── iam/       — package iam: RoleExists, ...
│   ├── ssm/
│   ├── lambda/
│   └── eks/       — (future) package eks: ClusterExists, ...
└── fixtures/
    ├── s3/        — package s3: SeedObject, SeedObjectContext, ...
    ├── ssm/
    ├── secretsmanager/
    └── sqs/
```

Each service file becomes its own `package <service>` (matching the
AWS SDK v2 `service/<name>` convention so consumers' import muscle
memory carries over). Collisions with the AWS SDK package names are
resolved by import alias at the call site:

```go
import (
    s3sdk    "github.com/aws/aws-sdk-go-v2/service/s3"
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
    s3fix    "github.com/donaldgifford/libtftest/fixtures/s3"
)

s3assert.BucketExists(t, tc.AWS(), bucket)
s3fix.SeedObject(t, tc.AWS(), bucket, key, body)
```

### Why not also refactor `awsx/`?

`awsx/` is already idiomatic Go — one file per service, ~10-line
constructor per service, single flat package. There's no namespacing
pain there because every export is `NewXxx(cfg)`. Leave it alone.

### Migration shape

Pre-1.0, do this as a single coordinated PR — no shim or re-export
layer. The consumer-side breakage is one find-and-replace per call
site (`assert.S3.BucketExists` → `s3assert.BucketExists`, etc.).

Action items:

- Move each `assert/<service>.go` into `assert/<service>/<service>.go`,
  rewrite the zero-size struct methods as package-level functions,
  delete the package-level `var S3 = s3Asserts{}` exports
- Same for `fixtures/`
- Keep the paired-method pattern (`Foo` + `FooContext`) — that's
  unrelated to layout
- Update `tftest:add-assertion` and `tftest:add-fixture` skill templates
  in claude-skills to emit the new shape
- Update `.claude/skills/libtftest-add-assertion` and
  `libtftest-add-fixture` to emit the new shape
- Update `docs/examples/` to use the new import shape
- Update `README.md` API surface section
- Bump minor — pre-1.0 SemVer permits breaking changes on minor
- Plugin in claude-skills: bump pin range lower bound to match the new
  libtftest minor

## Recommendation

**Answer:** libtftest should not ship the EKS test framework, but should
land **four** changes (one layout refactor + three generic features)
that the EKS coverage matrix surfaces, so consumers can build the EKS
suite (or any other "complex module" suite) without per-consumer
reinvention.

### Sequencing

The package-layout refactor needs to land **first**, so the three RFE
features ship straight into the new layout instead of being migrated
twice.

```text
PR 1 — Layout refactor (Option A2)
       ↓
PR 2 — tc.AssertIdempotent
PR 3 — assert/tags + assert/snapshot   (independent, can land in parallel)
       ↓
Optional: consumer publishes EKS test suite + a reference impl link
          back into libtftest's docs/examples/
```

### 1. Package layout: `assert/{service}/`, `fixtures/{service}/`

See [Package layout decision](#package-layout-decision) above. This is
the prerequisite — every other RFE below assumes the new layout.

### 2. `TestCase.AssertIdempotent(ctx)`

Runs `apply` twice and asserts the second plan reports zero changes.
Catches `ignore_changes` bugs and provider drift. Lives in the
`TestCase` API. Thin wrapper around the existing `ApplyContext` +
`PlanContext` plumbing.

### 3. `assert/tags` package

Root-tag propagation assertions. Given a set of resource ARNs and a
baseline tag map, verify all resources carry the baseline. Service-agnostic
backed by the AWS Resource Groups Tagging API.

```go
import tagsassert "github.com/donaldgifford/libtftest/assert/tags"

tagsassert.PropagatesFromRoot(t, tc.AWS(), expectedTags, arns...)
```

### 4. `assert/snapshot` package

Generic JSON snapshot-testing for structured output (IAM policy
documents, plan JSON, Terraform output values). LocalStack-independent.
Two assertion forms: strict (byte diff) and structural (normalized
key order, comment-skipping). Save snapshots to
`testdata/snapshots/<test>.json` per convention.

```go
import snapassert "github.com/donaldgifford/libtftest/assert/snapshot"

snapassert.JSONStructural(t, planJSON, "testdata/snapshots/iam-trust-policy.json")
```

### What this investigation does NOT recommend

- A `cmd/eks-test-scaffold` generator or any EKS-specific scaffolding skill
- An `assert/eks` package (until a consumer use case shows up — then it
  goes in the new layout naturally)
- Refactoring `awsx/` — already idiomatic
- Interface-based design for `assert/*` / `fixtures/*` — no substitution
  surface justifies it

### Next steps

- One `DESIGN-0003` covering the four PRs above, then per-PR
  `IMPL-NNNN` plans
- Land PR 1 (layout) first; it's mostly mechanical
- Land PRs 2–4 in any order — they're orthogonal
- Leave EKS-specific coverage to whichever consumer needs it; their
  tests using libtftest + the new features are a natural reference
  implementation worth linking from `docs/examples/`

## References

- LocalStack EKS docs: <https://docs.localstack.cloud/aws/services/eks/>
- LocalStack 4.10 release (Pod Identity + IRSA):
  <https://blog.localstack.cloud/localstack-for-aws-release-v-4-10-0/>
- testcontainers-go LocalStack module:
  <https://golang.testcontainers.org/modules/localstack/>
- `lstk` CLI (LocalStack helper): <https://github.com/localstack/lstk>
- libtftest `Options.Image` (per-test image override):
  `libtftest.go` / `Options.Image` field
- libtftest `RequirePro(t)` gate: `localstack/edition.go`
