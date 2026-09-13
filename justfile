# libtftest — task runner
#
# Project automation via just, the single task runner for this repo (it
# replaced the Makefile in IMPL-0005). `just` with no arguments lists every
# recipe. CI installs just via jdx/mise-action (install_args: just) and calls
# the same recipes developers use locally.

set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# Optional recipe modules; the import is skipped silently when the file is
# absent. docker.just carries the buildx bake recipes.
import? 'docker.just'

project_name      := "libtftest"
project_owner     := "donaldgifford"
go_package        := "github.com/" + project_owner + "/" + project_name
build_dir         := "build"
bin_dir           := build_dir + "/bin"
coverage_out      := "coverage.out"
allowed_licenses  := "Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0"
goimports_local   := "github.com/" + project_owner

# Canonical location of the pinned LocalStack single image; bump-localstack
# reads the current version from here.
localstack_pin_file := "localstack/container.go"

# Version info derived from git; falls back to dev when not in a repo or tag-less.
commit_hash := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
version     := `git describe --tags --always --dirty 2>/dev/null || echo dev`

# Default: list recipes
_default:
    @just --list --unsorted

# ─── Build ──────────────────────────────────────────────────────────

# Build everything (core)
[group('build')]
build: build-core

# Build the core CLI binary into build/bin/libtftest
[group('build')]
build-core:
    @mkdir -p {{ bin_dir }}
    @go build -ldflags "-X main.version={{ version }} -X main.commit={{ commit_hash }}" \
        -o {{ bin_dir }}/{{ project_name }} ./cmd/{{ project_name }}
    @echo "✓ Core binaries built"

# Remove build artifacts and the Go build cache
[group('build')]
clean:
    @rm -rf {{ bin_dir }}/
    @rm -f {{ coverage_out }}
    @go clean -cache
    @find . -name "*.test" -delete
    @echo "✓ Cleaned build artifacts"

# ─── Run ────────────────────────────────────────────────────────────

# Build then run the CLI
[group('run')]
run: build
    @{{ bin_dir }}/{{ project_name }}

# ─── Test ───────────────────────────────────────────────────────────

# Run all unit tests with the race detector (no Docker required)
[group('test')]
test:
    @go test -v -race ./...

# Run all tests (core + plugins)
[group('test')]
test-all: test

# Run tests for a single package: just test-pkg ./localstack
[group('test')]
test-pkg pkg:
    @go test -v -race {{ pkg }}

# Run unit tests with a coverage profile written to coverage.out
[group('test')]
test-coverage:
    @go test -v -race -coverprofile={{ coverage_out }} ./...

# Run tests and open the HTML coverage report
[group('test')]
test-report:
    @go test -coverprofile={{ coverage_out }} ./...
    @go tool cover -html={{ coverage_out }}

# Run the LocalStack integration suite (requires Docker + Terraform)
[group('test')]
test-integration:
    @go test -tags=integration -v -race ./...

# Run the Pro integration suite (requires Docker, Terraform and LOCALSTACK_AUTH_TOKEN)
[group('test')]
test-integration-pro:
    @go test -tags='integration localstack_pro' -v -race ./...

# Run the docs/examples runnable tests (requires Docker + Terraform)
[group('test')]
test-examples:
    @go test -tags=integration_examples -v -race ./docs/examples/...

# ─── Lint & format ─────────────────────────────────────────────────

# Run golangci-lint
[group('lint')]
lint:
    @golangci-lint run ./...

# Run golangci-lint with --fix
[group('lint')]
lint-fix:
    @golangci-lint run --fix ./...

# Verify the golangci-lint configuration
[group('lint')]
lint-config:
    @golangci-lint config verify

# Lint GitHub Actions workflows (also runs as the `lint-actions` CI job)
[group('lint')]
lint-actions:
    @actionlint

# Format code with gofmt + goimports
[group('lint')]
fmt:
    @gofmt -s -w .
    @goimports -w -local {{ goimports_local }} .

# ─── Markers & generated docs ──────────────────────────────────────

# Fail if any libtftest.RequirePro caller lacks a `libtftest:requires` marker
[group('docs')]
check-markers:
    @go run ./tools/docgen check -root .
    @echo "✓ All RequirePro callers carry a marker"

# Regenerate docs/feature-matrix.md from `libtftest:requires` markers
[group('docs')]
docs-matrix:
    @go run ./tools/docgen render -root . -out docs/feature-matrix.md
    @echo "✓ Regenerated docs/feature-matrix.md"

# ─── License compliance ─────────────────────────────────────────────

# Check dependency licenses against the allow list
[group('license')]
license-check:
    @go-licenses check ./... --allowed_licenses={{ allowed_licenses }}

# Generate CSV report of all dependency licenses
[group('license')]
license-report:
    @go-licenses report ./... --template=.github/licenses-csv.tpl

# ─── LocalStack ─────────────────────────────────────────────────────

# Start a shared LocalStack via lstk (then: export LIBTFTEST_CONTAINER_URL=http://localhost:4566)
[group('localstack')]
localstack-up:
    lstk start

# Stop the lstk-managed LocalStack container
[group('localstack')]
localstack-down:
    lstk stop

# Show lstk-managed LocalStack status and deployed resources
[group('localstack')]
localstack-status:
    lstk status

# Tail lstk-managed LocalStack logs
[group('localstack')]
localstack-logs:
    lstk logs

# Renovate keeps the LocalStack pin current automatically (customManager in
# renovate.json5); bump-localstack is the manual/offline escape hatch. It bumps
# the calendar-versioned single image (defaultImage) in every *.go, *.tf,
# Dockerfile*, docker-bake.hcl and *.md (incl. CLAUDE.md). The token-free
# community fallback tag (defaultCommunityImage, legacy semver) is a
# deliberate manual pin and is left untouched. (just uses the comment line
# directly above a recipe as its --list description, hence the gap below.)

# Bump the pinned LocalStack single-image calver everywhere: just bump-localstack 2026.08.1
[group('localstack')]
bump-localstack ls_version:
    #!/usr/bin/env bash
    set -euo pipefail
    old="$(grep -oE 'localstack/localstack:[0-9]{4}\.[0-9]{2}\.[0-9]+' {{ localstack_pin_file }} | head -1 | cut -d: -f2)"
    if [ -z "$old" ]; then
        echo "Error: could not detect current pinned version in {{ localstack_pin_file }}" >&2
        exit 1
    fi
    echo "Bumping LocalStack pin..."
    echo "  current: $old"
    echo "  target:  {{ ls_version }}"
    find . -type f \( -name '*.go' -o -name '*.tf' -o -name 'Dockerfile*' -o -name 'docker-bake.hcl' -o -name '*.md' \) \
        -not -path './.git/*' -not -path './vendor/*' -not -path './build/*' \
        -exec sed -i.bak "s|localstack/localstack:$old|localstack/localstack:{{ ls_version }}|g" {} +
    find . -name '*.bak' -not -path './.git/*' -delete
    echo "✓ Pin updated. Review with 'git diff', then run 'just test' to verify."

# ─── Release ────────────────────────────────────────────────────────

# Validate the goreleaser config
[group('release')]
release-check:
    @goreleaser check

# Snapshot release locally (no publish, no sign); needs syft for the SBOMs
[group('release')]
release-local:
    @goreleaser release --snapshot --clean --skip=publish --skip=sign

# Tag and push a new release: just release v0.1.0 (normally release.yml tags on merge instead)
[group('release')]
release tag:
    @git tag -a {{ tag }} -m "Release {{ tag }}"
    @git push origin {{ tag }}

# ─── Composite gates ────────────────────────────────────────────────

# Pre-commit gate: lint + test
[group('gate')]
check: lint test
    @echo "✓ Pre-commit checks passed"

# Full CI gate: lint + test + build + license-check + check-markers
[group('gate')]
ci: lint test build license-check check-markers
    @echo "✓ CI pipeline complete"
