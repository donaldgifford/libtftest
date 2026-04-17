variable "REGISTRY" {
  default = "ghcr.io/donaldgifford"
}

variable "TAG" {
  default = "dev"
}

group "default" {
  targets = ["sneakystack"]
}

group "ci" {
  targets = ["sneakystack-ci"]
}

target "sneakystack" {
  dockerfile = "Dockerfile.sneakystack"
  tags       = ["${REGISTRY}/sneakystack:${TAG}", "${REGISTRY}/sneakystack:latest"]
  platforms  = ["linux/amd64", "linux/arm64"]
}

target "sneakystack-ci" {
  dockerfile = "Dockerfile.sneakystack"
  tags       = ["${REGISTRY}/sneakystack:${TAG}"]
  platforms  = ["linux/amd64"]
  cache-from = ["type=gha"]
  cache-to   = ["type=gha,mode=max"]
}
