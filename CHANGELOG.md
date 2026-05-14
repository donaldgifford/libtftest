# Changelog

All notable changes to libtftest are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/).
While the library is pre-1.0, minor-version bumps may contain
breaking changes; the API freeze begins at v1.0.

This file is regenerated from conventional commits by `git-cliff`.
Manual edits will be overwritten by the `Changelog Sync` job in the
`Release` workflow after each tagged release — author release notes
via conventional commit subjects and bodies instead.

## [unreleased]

### Features

- *(internal/testfake)* Add shared FakeTB for per-service test packages
- *(libtftest)* Add AssertIdempotent + AssertIdempotentApply
- *(assert/tags)* Add service-agnostic tag-propagation assertion
- *(assert/snapshot)* Add JSON snapshot testing + IAM policy extractor

### Tooling

- *(tools)* Add docgen marker scanner + feature-matrix renderer
- *(tools)* Wire docs-matrix + check-markers into Make and CI

### Bug Fixes

- *(testfake)* Implement missing testing.TB methods to unblock CI
- *(examples)* Drop t.Parallel from Test_Example10_SnapshotIAM

### Refactor

- *(assert/s3)* Migrate S3 assertions to per-service package
- *(assert/dynamodb)* Migrate DynamoDB assertions to per-service package
- *(assert/iam)* Migrate IAM assertions to per-service package
- *(assert/ssm)* Migrate SSM Parameter Store assertions to per-service package
- *(assert/lambda)* Migrate Lambda assertions to per-service package
- *(assert)* Delete old flat-layout files; update example surface guard
- *(fixtures)* Migrate fixtures to per-service packages

### Documentation

- *(inv)* Add INV-0002 for EKS coverage via LocalStack
- *(inv)* INV-0002 — concrete image tags, edition gating, layout decision
- *(design)* Add DESIGN-0003 for layout refactor + 3 hygiene primitives
- *(design)* Resolve DESIGN-0003 open questions
- *(impl)* Add IMPL-0004 for layout refactor + hygiene primitives
- *(impl)* Resolve IMPL-0004 open questions + future-work INVs
- Conclude INV-0003, refine INV-0004, fold convention into IMPL-0004
- Conclude INV-0004 and fold tools/docgen into IMPL-0004
- *(design)* Add Parts 5 + 6 to DESIGN-0003 for doc.go and tools/docgen
- Roll out doc.go convention to all packages
- *(examples)* Migrate example surface refs to per-service shape
- Update README + CLAUDE.md for per-service layout
- *(skills)* Update add-assertion + add-fixture for per-service shape
- *(impl-0004)* Mark Phase 3 quality gates green
- Add idempotency example + README/CLAUDE updates
- *(assert/tags)* Add example 09 + integration surface + README
- *(assert/snapshot)* Add example 10 + README/index updates
- *(impl-0004)* Close Testing Plan checkboxes and note Phase 8/9 scope
- *(impl-0004)* Mark Phase 8 done (claude-skills plugin v0.3.0 shipped)
- *(impl-0004)* Check off memory pointer (Phase 9 task)
- *(impl-0004)* Check off PR CI green (Phase 9)
- *(impl-0004)* Check off dependabot orphan verification (Phase 9)
- Flip IMPL-0004/DESIGN-0003/INV-0002 statuses post-implementation

### Miscellaneous Tasks

- *(tooling)* Add lstk + just to mise

## [0.1.2] - 2026-05-13

### Bug Fixes

- *(release)* Wire up docker bake release target + cosign signing ([#11](https://github.com/donaldgifford/libtftest/issues/11))

## [0.1.1] - 2026-05-12

### Miscellaneous Tasks

- Consolidate changelog regen into release workflow ([#10](https://github.com/donaldgifford/libtftest/issues/10))

## [0.1.0] - 2026-05-12

### Features

- Terratest 1.0 *Context paired-method API (IMPL-0003, v0.1.0) ([#9](https://github.com/donaldgifford/libtftest/issues/9))

## [0.0.2] - 2026-05-11

### Features

- *(skills)* Add libtftest:add-awsx-client local skill
- *(skills)* Add libtftest:add-assertion local skill
- *(skills)* Add libtftest:add-fixture local skill
- *(agents)* Add libtftest-reviewer review agent
- *(skills)* Add Phase 5 libtftest operational skills

### Documentation

- *(design,impl)* Add 0002 claude skills design and implementation plan
- *(impl)* Mark Phase 0 complete in IMPL-0002
- *(impl)* Mark Phase 1 local skills complete in IMPL-0002
- *(impl)* Mark Phase 2 consumer-scaffolding skills complete in IMPL-0002
- *(impl)* Mark Phase 3 day-2 consumer skills complete in IMPL-0002
- *(impl)* Mark Phase 4 review agents complete in IMPL-0002
- *(impl)* Mark Phase 5 operational skills complete in IMPL-0002
- *(skills)* Wire up Phase 6 discovery and CI
- *(impl)* Mark Phase 6 discovery+CI complete in IMPL-0002
- Update CLAUDE.md status for IMPL-0002 skills work

### Miscellaneous Tasks

- *(skills)* Scaffold .claude/ for local skill development

## [0.0.1] - 2026-04-17

### Features

- Initialize Go module github.com/donaldgifford/libtftest
- Scaffold package directory structure
- *(naming)* Implement parallel-safe prefix generation
- *(dockerx)* Implement Docker daemon detection with error classification
- *(logx)* Implement structured logging and artifact dumping
- *(localstack)* Add testcontainers-go dep and edition detection
- *(localstack)* Implement health check parsing and edition detection
- *(localstack)* Implement container lifecycle and init hooks
- Add testdata/mod-s3 fixture Terraform module
- *(localstack)* Add integration tests and pin default image to 4.4
- *(tf)* Implement workspace copy, override injection, and options
- Implement core TestCase API with Plan, Apply, and cleanup
- Implement awsx clients, fixtures, and assertion helpers
- *(harness)* Implement shared-container TestMain and Sidecar interface
- *(sneakystack)* Implement Store, proxy, sidecar, and Docker packaging
- Finalize CI pipeline, reusable workflow, and README

### Bug Fixes

- Use PortEndpoint for edge port and upgrade vulnerable xz dep
- *(ci)* Add Terraform setup to integration test jobs

### Documentation

- Add IMPL-0001 and update DESIGN-0001 with current API patterns
- Add development guide, usage examples, and expand README

### Miscellaneous Tasks

- Verify build, lint, and golangci config for Phase 1
- Update testing plan checklist in IMPL-0001

