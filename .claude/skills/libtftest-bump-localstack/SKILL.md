---
name: libtftest:bump-localstack
description: >
  Bump the pinned LocalStack image version across libtftest. Use when
  upgrading to a new LocalStack release: wraps the make bump-localstack
  LS_VERSION=<x> Makefile target with the playbook (release notes,
  CHANGELOG, integration tests, docs).
when_to_use: >
  When the user says "bump LocalStack to X", "upgrade to LocalStack 4.5",
  or "we need to support a new LocalStack version". Activates inside the
  libtftest repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:bump-localstack

The Makefile target `make bump-localstack LS_VERSION=<x>` does the
mechanical sed/grep work. This skill wraps it with the reasoning steps —
release notes, regression checks, CHANGELOG drafting, docs updates —
that humans always end up doing anyway.

## Repo conventions

- Pinned image lives in `localstack/container.go` constants (`defaultImage`,
  `defaultProImage`)
- Same pin is repeated in: integration tests, README, `docs/examples/`,
  `CLAUDE.md`, `_preamble.md`, `Dockerfile.sneakystack`,
  `docker-bake.hcl`. The Makefile target updates all of them.
- LocalStack `:latest` requires a Pro auth token — never use it.

## Procedure

### 1. Confirm the target version

Ask the user:

- **Target version**, e.g., `4.5`. The Makefile target accepts the bare
  version string (no `localstack/localstack:` prefix).
- Whether to bump Pro alongside Community (almost always yes — they
  release together).

### 2. Read the LocalStack release notes

```bash
# Open in the user's browser
open "https://docs.localstack.cloud/references/changelog/"
# Or fetch via curl
curl -s https://api.github.com/repos/localstack/localstack/releases/tags/v$(LS_VERSION) | jq -r '.body' | head -100
```

Walk the release notes for known regression areas (see
`references/release-notes-checklist.md`). Surface anything the user needs
to know before proceeding.

### 3. Run the Makefile target

```bash
make bump-localstack LS_VERSION=<target>
```

This updates all the pin sites in one pass via `find ... -exec sed`.
Review with `git diff` after.

### 4. Verify the build

```bash
make lint
make test
```

If anything fails, capture the error and ask whether to roll back. Common
post-bump failures:

- LocalStack changed the wire format of an endpoint we depend on
- A service that was Community is now Pro (or vice versa)
- The health endpoint response shape changed (`AllServicesReady` parser
  may need updating)

### 5. Run the integration tests

```bash
docker version >/dev/null 2>&1 || { echo "ERROR: Docker not running"; exit 1; }
make test PKG=./libtftest_integration_test.go  # or full integration suite
```

If integration tests fail on the new version, that's the regression area
to investigate before merging.

### 6. Draft a CHANGELOG entry

If `CHANGELOG.md` exists at the repo root, prepend an entry under
`[Unreleased]`:

```markdown
### Changed

- Bumped default LocalStack image to `localstack/localstack:<target>`
  (was `<previous>`). See <release-notes-link> for upstream changes.

### Fixed (if regressions surfaced)

- Updated `<thing>` to handle <regression>.
```

If no CHANGELOG exists, ask the user whether to create one or commit
without.

### 7. Commit with a conventional commit message

```text
chore(localstack): bump pinned image to localstack/localstack:<target>

Updates default image in localstack/container.go and 7 other sites
(README, Dockerfile.sneakystack, docker-bake.hcl, docs/examples/05,
CLAUDE.md, _preamble.md, integration tests).

Verified: make lint && make test && integration suite pass.
Release notes: <link>
```

## Edge cases

- **Pro version released later than Community**: bump Community first,
  pin Pro to the previous version manually, follow up with a Pro-only
  bump. Document in the commit.
- **Pinned version doesn't exist yet** (preview): use the LocalStack
  `<version>-rc.N` tag if available; otherwise skip until the proper
  release.
- **Breaking wire-format change**: roll the bump back, document the
  break in `docs/development/`, and open an upstream issue against
  libtftest pointing at the LocalStack release notes. Don't try to
  hot-fix on the bump branch.

## References

- `references/release-notes-checklist.md` — known regression areas to
  scan for in LocalStack release notes
- `Makefile` — the `bump-localstack` target
- `localstack/container.go` — canonical pin location
