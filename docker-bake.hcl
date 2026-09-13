// docker-bake.hcl — multi-arch build pipeline for sneakystack, the LocalStack
// gap-filling proxy that ships from the libtftest repo (cmd/sneakystack,
// Dockerfile.sneakystack).
//
// Targets:
//   - default: local single-arch build (used by `docker buildx bake`)
//   - ci:      linux/amd64 build + push of `:dev-ci` for PR validation
//   - release: multi-arch build + push to GHCR (CI only, gated on tag)
//
// CI consumes this via docker/bake-action@v7 with the `targets` input. The
// release workflow merges in tag-derived image refs from
// docker/metadata-action's bake-file outputs and overrides VERSION/COMMIT
// with `--set *.args.VERSION=... *.args.COMMIT=...`. Locally, `docker.just`
// wraps the safe targets and prompts before the publishing ones.

variable "REGISTRY" {
  default = "ghcr.io/donaldgifford/sneakystack"
}

variable "TAG" {
  default = "dev"
}

// Build metadata forwarded to Dockerfile.sneakystack ARGs and from there into
// -ldflags (the `version`/`commit`/`date` vars in cmd/sneakystack/main.go).
variable "VERSION" {
  default = "0.0.0-dev"
}

variable "COMMIT" {
  default = "none"
}

variable "DATE" {
  default = "unknown"
}

group "default" {
  targets = ["sneakystack"]
}

group "ci" {
  targets = ["sneakystack-ci"]
}

group "release" {
  targets = ["sneakystack-release"]
}

target "_common" {
  context    = "."
  dockerfile = "Dockerfile.sneakystack"
  args = {
    VERSION = "${VERSION}"
    COMMIT  = "${COMMIT}"
    DATE    = "${DATE}"
  }
  labels = {
    // `source` must name the repo that owns the package: GHCR uses it to
    // link the image to its source, README and visibility settings.
    "org.opencontainers.image.source"      = "https://github.com/donaldgifford/libtftest"
    "org.opencontainers.image.licenses"    = "Apache-2.0"
    "org.opencontainers.image.description" = "sneakystack: LocalStack gap-filling proxy (IAM Identity Center, Organizations, Control Tower) from libtftest"
  }
}

// Stub providing default `tags` for local `docker buildx bake`. CI runs
// override this target via docker/metadata-action's bake-file-tags output so
// the bake pushes the same semver-derived image refs the metadata-action
// emits — which is what cosign then signs in the next step.
// `sneakystack-release` inherits from this and does NOT declare tags itself,
// so the override actually takes effect (with HCL inheritance, a child's tags
// list replaces the parent's, not extends it).
target "docker-metadata-action" {
  tags = [
    "${REGISTRY}:${TAG}",
    "${REGISTRY}:latest",
  ]
}

// Local build. `platforms` is deliberately unset so buildx uses the daemon's
// native platform (linux/arm64 on Apple Silicon Docker Desktop, linux/amd64
// elsewhere) with no QEMU emulation. Don't reintroduce
// "linux/${BAKE_LOCAL_PLATFORM}": that variable already contains the OS
// (e.g. "darwin/arm64/v8") and the result is an invalid platform specifier.
target "sneakystack" {
  inherits = ["_common"]
  tags     = ["${REGISTRY}:${TAG}"]
}

// CI builds are linux/amd64 only — emulated arm64 builds via QEMU on
// GitHub's ubuntu-latest runners take ~25 min and dominate PR feedback
// time. Multi-arch coverage is restored in `sneakystack-release`, which
// runs only on tag pushes.
target "sneakystack-ci" {
  inherits  = ["_common"]
  tags      = ["${REGISTRY}:${TAG}-ci"]
  platforms = ["linux/amd64"]
}

// Publishes. Tags intentionally omitted — they come from
// docker-metadata-action (defaults for local bake; CI overrides via
// metadata-action). `output = type=registry` means this target PUSHES,
// including `:latest` when run locally with the default tags above.
target "sneakystack-release" {
  inherits = ["_common", "docker-metadata-action"]
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
  output = ["type=registry"]
}
