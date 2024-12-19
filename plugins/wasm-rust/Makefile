PLUGIN_NAME ?= say-hello
BUILDER_REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
RUST_VERSION ?= 1.82
ORAS_VERSION ?= 1.0.0
HIGRESS_VERSION ?= 1.0.0-rc
BUILDER ?= ${BUILDER_REGISTRY}wasm-rust-builder:rust${RUST_VERSION}-oras${ORAS_VERSION}
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMAGE_TAG = $(if $(strip $(PLUGIN_VERSION)),${PLUGIN_VERSION},${BUILD_TIME}-${COMMIT_ID})
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${IMAGE_TAG}

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build \
		--build-arg PLUGIN_NAME=${PLUGIN_NAME} \
		-t ${IMG} \
	    --output extensions/${PLUGIN_NAME} \
	    .
	@echo ""
	@echo "output wasm file: extensions/${PLUGIN_NAME}/plugin.wasm"

lint-base:
	cargo fmt --all --check
	cargo clippy --workspace --all-features --all-targets
lint:
	cargo fmt --all --check --manifest-path extensions/${PLUGIN_NAME}/Cargo.toml
	cargo clippy --workspace --all-features --all-targets --manifest-path extensions/${PLUGIN_NAME}/Cargo.toml

test-base:
	cargo test --lib

test:
	cargo test --manifest-path extensions/${PLUGIN_NAME}/Cargo.toml

builder:
	DOCKER_BUILDKIT=1 docker build \
			--build-arg RUST_VERSION=$(RUST_VERSION) \
			--build-arg ORAS_VERSION=$(ORAS_VERSION) \
			--build-arg HIGRESS_VERSION=$(HIGRESS_VERSION) \
			-f DockerfileBuilder \
			-t ${BUILDER} \
			.
	@echo ""
	@echo "image: ${BUILDER}"
