---
name: libtftest:release
description: >
  Tag and push a libtftest release. Use when shipping vX.Y.Z: verifies
  clean main + green CI + unique version, runs make release-check,
  drafts CHANGELOG, tags and pushes with explicit confirmation. Refuses
  to push to anything other than the resolved upstream remote.
when_to_use: >
  When the user says "release v0.1.0", "tag a release", "ship libtftest
  vX.Y.Z", or runs /libtftest:release. Activates inside the libtftest
  repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:release

Drives the release process from "main is green and ready" through
`git push origin <tag>`. This skill is **destructive** at the final step
(tag + push); it requires explicit user confirmation before that step
and refuses to push to anything other than the resolved upstream remote.

## Repo conventions

- Versions follow SemVer: `v0.X.Y` for pre-1.0, `vX.Y.Z` for 1.0+
- Pre-1.0 may include breaking changes between minor versions
- Releases are tagged on `main` only — never on a feature branch
- `make release` exists but does only the tag+push; this skill does the
  full pre-flight + CHANGELOG draft

## Procedure

### 1. Confirm the version

Ask the user for the target version (e.g., `v0.1.0`). Validate:

```bash
# Format check
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]] || echo "ERROR: bad format"

# Doesn't already exist
git tag -l | grep -qx "$VERSION" && echo "ERROR: $VERSION already exists"

# Higher than the current latest
git describe --tags --abbrev=0 2>/dev/null
```

### 2. Verify the working copy

```bash
# On main
test "$(git branch --show-current)" = "main" || echo "ERROR: not on main"

# Clean
test -z "$(git status --porcelain)" || echo "ERROR: uncommitted changes"

# Up to date with origin
git fetch origin
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" || echo "ERROR: not synced with origin/main"
```

If any check fails, halt and surface the issue.

### 3. Verify CI is green

```bash
# Most recent run on main
gh run list --branch main --limit 1 --json conclusion,status,headSha
```

The most recent run must be `conclusion: "success"` and target the
current HEAD SHA. If not, halt:

> "CI on the current main HEAD isn't green. The latest run on main is
> `<conclusion>`. Refusing to release until CI is green."

### 4. Run release-check

```bash
make release-check
```

This validates the goreleaser config. If it fails, surface the error.

### 5. Resolve the upstream remote

```bash
gh repo view --json url --jq .url
# Should be https://github.com/donaldgifford/libtftest
```

The skill will only push tags to the remote that matches `git remote get-url
origin` AND is owned by `donaldgifford`. Refuse to push to forks or
unrelated remotes.

### 6. Draft the CHANGELOG entry

Read the previous tag's CHANGELOG entry for shape. Generate a new
`[<VERSION>] - <YYYY-MM-DD>` section by walking the commits since the
previous tag:

```bash
PREV_TAG=$(git describe --tags --abbrev=0)
git log "$PREV_TAG..HEAD" --pretty=format:'%s' | sort | uniq
```

Group by conventional-commit type:

- `feat:` → "Added"
- `fix:` → "Fixed"
- `chore:`, `refactor:` → "Changed"
- `docs:` → "Documentation"
- `test:` → "Testing" (or omit)

Show the user the draft and ask for adjustments before committing.

### 7. Commit the CHANGELOG

```bash
git add CHANGELOG.md
git commit -m "chore(release): prepare $VERSION"
git push origin main
```

Wait for CI to go green on the new commit (re-check step 3).

### 8. Tag and push (DESTRUCTIVE)

Before this step, **explicitly confirm** with the user:

> "Ready to tag and push `$VERSION` to `origin`. This is destructive —
> tags can be force-replaced but it's noisy. Type `yes` to proceed."

If the user types anything other than `yes`, abort.

```bash
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"
```

### 9. Verify the release

```bash
# goreleaser fires off via GHA on tag push
gh run list --workflow release.yml --limit 1 --json conclusion,status

# Once goreleaser completes, the release page should exist
gh release view "$VERSION"
```

Tell the user where to find the release notes for editing if they want
to flesh them out.

## Edge cases

- **First release** (no previous tag): the CHANGELOG draft can't walk
  commits-since-tag. Walk all commits on main instead.
- **Revert needed after push**: tags can be deleted with
  `git push origin :refs/tags/<tag>`, but the goreleaser GHA may have
  already published binaries. Don't try to revert from this skill —
  hand off to the user with instructions.
- **Pre-release tag** (e.g., `v0.2.0-rc.1`): allow these but flag in
  the CHANGELOG draft as `[Pre-release]` rather than a regular section.
- **No CHANGELOG.md exists**: ask whether to create one or skip the
  drafting step. Recommend creating one.

## Refusal conditions

This skill **refuses** to:

- Push to any remote other than the one matching `git remote get-url
  origin` AND owned by `donaldgifford`
- Tag while the working copy is dirty
- Tag while CI is failing on the current HEAD
- Tag a version lower than or equal to the current latest tag
- Skip the explicit confirmation step before pushing

If the user wants to bypass any of these, they should run the underlying
git commands manually.
