PLUGIN_NAME ?= hello-world
REGISTRY ?=
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${BUILD_TIME}-${COMMIT_ID}

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                            -t ${IMG} \
	                            --output extensions/${PLUGIN_NAME} \
	                            .
	@echo ""
	@echo "image:            ${IMG}"
	@echo "output wasm file: extensions/${PLUGIN_NAME}/plugin.wasm"

build-push: build
	docker push ${IMG}
