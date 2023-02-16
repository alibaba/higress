PLUGIN_NAME ?= hello-world
REGISTRY ?=
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${BUILD_TIME}-${COMMIT_ID}
BUILDER_IMG ?= ${REGISTRY}wasm-go-builder:go${GO_VERSION}-tinygo${TINYGO_VERSION}

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                               --build-arg GO_VERSION=${GO_VERSION} \
	                               --build-arg TINYGO_VERSION=${TINYGO_VERSION} \
	                               -t ${IMG} \
	                               -f DockerfileBuilder \
	                               --output extensions/${PLUGIN_NAME} .
	@echo ""
	@echo "image:            ${IMG}"
	@echo "output wasm file: extensions/${PLUGIN_NAME}/plugin.wasm"

build-push: build
	docker push ${IMG}

builder:
	DOCKER_BUILDKIT=1 docker build --target wasm-builder \
	                               --build-arg GO_VERSION=${GO_VERSION} \
	                               --build-arg TINYGO_VERSION=${TINYGO_VERSION} \
	                               -t ${BUILDER_IMG} \
	                               -f DockerfileBuilder \
	                               .
	@echo "You can use the following command to build the final image in the next stage"
	@echo "WASM_BUILDER=${BUILDER_IMG} PLUGIN_NAME=${PLUGIN_NAME} make bb"

bb: build-on-builder
build-on-builder:
	DOCKER_BUILDKIT=1 docker build --target building \
	                               --target output \
	                               --build-arg WASM_BUILDER=${WASM_BUILDER} \
	                               -t ${IMG} \
	                               -f DockerfileBuilder \
	                               --output extensions/${PLUGIN_NAME} .
	@echo ""
	@echo "image:            ${IMG}"
	@echo "output wasm file: extensions/${PLUGIN_NAME}/plugin.wasm"
