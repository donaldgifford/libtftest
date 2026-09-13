# libtftest — docker recipes
#
# Docker image build/push pipeline via docker buildx bake. Reads
# docker-bake.hcl for target definitions. Imported by the main justfile with
# `import? 'docker.just'`.
#
# Only `docker-build`, `docker-ci` and `docker-print` are side-effect free.
# `docker-release` and `docker-push` publish to GHCR — the release target
# tags `:latest` — so they prompt first. CI is the intended caller of the
# release target (see .github/workflows/release.yml); pass `--yes` to skip
# the prompt deliberately.
#
# No `set shell` here: imported modules inherit the root justfile's settings,
# and just rejects a redefinition ("setting `shell` ... is redefined").

# ─── Docker ─────────────────────────────────────────────────────────

# Local native-arch build, loaded into the local image store (ghcr.io/donaldgifford/sneakystack:dev)
[group('docker')]
docker-build:
    @docker buildx bake --load

# CI-shaped build (linux/amd64 only), loaded locally; no push
[group('docker')]
docker-ci:
    @docker buildx bake ci --load

# Print the resolved bake definition (validates docker-bake.hcl)
[group('docker')]
docker-print:
    @docker buildx bake --print

# Multi-arch release build that PUSHES :<TAG> and :latest to GHCR
[group('docker')]
[confirm("This builds linux/amd64+arm64 and PUSHES ghcr.io/donaldgifford/sneakystack:<TAG> and :latest. Continue?")]
docker-release:
    @docker buildx bake release

# Push the default target's tag (ghcr.io/donaldgifford/sneakystack:<TAG>) to GHCR
[group('docker')]
[confirm("This PUSHES ghcr.io/donaldgifford/sneakystack:<TAG>. Continue?")]
docker-push:
    @docker buildx bake --push
