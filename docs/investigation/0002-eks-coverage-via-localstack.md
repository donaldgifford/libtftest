---
id: INV-0002
title: "EKS coverage via LocalStack"
status: Open
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0002: EKS coverage via LocalStack

**Status:** Open
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
- [Recommendation](#recommendation)
  - [RFEs to spin out of this investigation](#rfes-to-spin-out-of-this-investigation)
  - [What this investigation does NOT recommend](#what-this-investigation-does-not-recommend)
  - [Next steps](#next-steps)
- [References](#references)
<!--toc:end-->

## Question

What does it take to make libtftest a viable harness for integration-testing
**Terraform EKS modules** against LocalStack — without libtftest itself
shipping the EKS test suite — and which of the patterns surfaced fall out as
**generic libtftest features** rather than EKS-specific cookbook?

## Hypothesis

LocalStack Pro 4.10+ covers enough of the EKS control-plane surface (clusters,
managed node groups, access entries, pod identity associations, IRSA / OIDC
provider) that a consumer can stand up real integration coverage of their own
EKS module using libtftest + the existing `assert/*`, `fixtures/*`, and
`harness/*` primitives. Three patterns the EKS coverage matrix surfaces are
not EKS-specific and would land cleaner as libtftest features
(idempotency, tag propagation, IAM policy snapshotting).

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
| LocalStack (local Pro/Ultimate) | `2026.5.0.dev121` (CalVer, post-`lstk` rename) |
| LocalStack (CI, OSS) | `4.x` (libtftest's current `localstack/localstack:4.4` default; CalVer applies to Pro tags only) |
| k3d image tag | `EKS_K3S_IMAGE_TAG=v1.32.13-k3s1` (pin to the module's target K8s version) |
| Docker socket mount | required — LocalStack spawns k3d containers from inside its own container |
| `LOCALSTACK_AUTH_TOKEN` | required (EKS is not in Community) — libtftest's `RequirePro(t)` gate already handles auto-skip |

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
   `localstack/localstack:4.4`. EKS Pod Identity / IRSA / the
   `bootstrap_cluster_creator_admin_permissions` flag need 4.10+. Options:
   - **(a)** Bump the global default to 4.10+ (broadest blast radius —
     re-run the full integration matrix, watch for regressions on existing
     S3/DynamoDB/etc. coverage).
   - **(b)** Document `Options.Image` override as the per-EKS-test
     idiom; default stays at 4.4 for non-EKS users.
   - **(c)** Introduce a service-aware default selection in libtftest:
     if `Services` includes `eks`, default to 4.10+.
   - Recommendation: **(b)** for now (no churn), revisit when LocalStack
     4.10+ has been stable for a quarter.

2. **Runtime cost.** A single LocalStack EKS cluster create is 30–90 s
   wall-clock (k3d image pull, k3s boot, EKS API stub bring-up). The
   coverage matrix has ~15 distinct test surfaces. Full sweep is 15–30 min
   even with shared-container mode + parallel sub-tests. Does the consumer
   accept this as nightly-only? Per-PR-only-on-EKS-paths? Decision belongs
   to the consumer, but libtftest should call it out in docs.

3. **EKS-specific helper coverage.** None of `assert/*` currently has an
   EKS namespace. Should there be a stub `assert.EKS.ClusterExistsContext`
   etc., even if the runtime depth is shallow? Or is this purely consumer
   territory?

4. **Pro CI gating.** Existing `integration-tests-pro` job hits 4.4 with
   a `LOCALSTACK_AUTH_TOKEN`. If EKS coverage materializes upstream, that
   job needs to pin to a 4.10+ tag and the `localstack/lstk` CLI may be
   useful (`mise.toml` already has `lstk` available locally). Out of scope
   for this investigation; flagging for the bump-localstack playbook.

## Recommendation

**Answer:** libtftest should not ship the EKS test framework, but should
land three generic features that the EKS coverage matrix surfaces, so
consumers can build the EKS suite (or any other "complex module" suite)
without per-consumer reinvention.

### RFEs to spin out of this investigation

These three patterns came out of the EKS analysis but are not EKS-specific.
They should each land as their own design/impl pair against libtftest core:

1. **`tc.AssertIdempotent(ctx)`** — runs `apply` twice and asserts the
   second plan reports zero changes. Catches `ignore_changes` bugs and
   provider drift. Lives in the `TestCase` API. Likely a thin wrapper
   around the existing `ApplyContext` + `PlanContext` plumbing.

2. **`assert.Tags.*` namespace** — propagation assertions. Given a set
   of resource ARNs and a baseline tag map, verify all resources carry
   the baseline. Service-agnostic; AWS Resource Groups Tagging API is
   the natural backend.

3. **`assert/snapshot/` package** — generic JSON snapshot-testing for
   structured output (IAM policy documents, plan JSON, Terraform output
   values). LocalStack-independent. Two assertion forms: strict (byte
   diff) and structural (normalized key order, comment-skipping). Save
   snapshots to `testdata/snapshots/<test>.json` per convention.

### What this investigation does NOT recommend

- A `cmd/eks-test-scaffold` generator or any EKS-specific scaffolding skill
- An `assert.EKS` namespace (until a consumer use case shows up)
- Bumping the default LocalStack image globally — too much churn for too
  little upside while the matrix is still being explored

### Next steps

- Open three small design docs (`DESIGN-0003/4/5`) or a single
  `DESIGN-0003` covering the three RFEs, then per-feature `IMPL-NNNN` plans
- Land them as small, independent PRs — they're orthogonal
- Leave EKS-specific coverage to whichever consumer needs it; their tests
  using libtftest + the three new features are a natural reference
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
