PLUGIN_NAME ?= hello-world
REGISTRY ?=
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${BUILD_TIME}-${COMMIT_ID}

build:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                               --build-arg THIS_ARCH=${THIS_ARCH} \
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