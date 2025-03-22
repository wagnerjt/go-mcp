variable "TAG" {
    default = "latest"
}

variable "REGISTRY" {
    default = "ghcr.io/wagnerjt/go-mcp"
}

group "default" {
    targets = ["server"]
}

group "all" {
    targets = ["server", "litellm-bridge"]
}

target "server" {
    dockerfile = "Dockerfile"
    tags = ["${REGISTRY}/server:${TAG}"]
    context = "./server"
}

target "litellm-bridge" {
    dockerfile = "Dockerfile"
    tags = ["${REGISTRY}/bridge:${TAG}"]
    context = "./bridge"
}

