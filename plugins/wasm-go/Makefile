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

# builder:
# To build a wasm-go-builder image.
# If you want to use Go/TinyGo with another version, please modify GO_VERSION/TINYGO_VERSION.
# After your wasm-go-builder image is built, you can use --build-arg BUILDER=<your-wasm-go-builder> to build plugin image.
builder:
	docker buildx build --platform linux/amd64,linux/arm64 \
			--build-arg BASE_IMAGE=docker.io/ubuntu \
			--build-arg GO_VERSION=1.19 \
			--build-arg TINYGO_VERSION=0.26.0 \
			-f DockerfileBuilder \
			-t wasm-go-builder:go1.19-tinygo0.26.0 \
			--push \
			.