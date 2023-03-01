PLUGIN_NAME ?= hello-world
REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
GO_VERSION ?= 1.19
TINYGO_VERSION ?= 0.27.0
BUILDER ?= ${REGISTRY}wasm-go-builder:go${GO_VERSION}-tinygo${TINYGO_VERSION}
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMG ?= ${REGISTRY}${PLUGIN_NAME}:${BUILD_TIME}-${COMMIT_ID}

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build --build-arg PLUGIN_NAME=${PLUGIN_NAME} \
	                            --build-arg BUILDER=${BUILDER}  \
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
# e.g.
#   REGISTRY=<your_docker_registry> make builder
# If you want to use Go/TinyGo with another version, please modify GO_VERSION/TINYGO_VERSION.
# After your wasm-go-builder image is built, you can use it to build plugin image.
# e.g.
#   PLUGIN_NAME=request-block BUILDER=<your-wasm-go-builder> make
builder:
	BUILDER=$(REGISTRY)wasm-go-builder:go$(GO_VERSION)-tinygo$(TINYGO_VERSION)
	docker buildx build --no-cache \
			--platform linux/amd64,linux/arm64 \
			--build-arg BASE_IMAGE=docker.io/ubuntu \
			--build-arg GO_VERSION=$(GO_VERSION) \
			--build-arg TINYGO_VERSION=$(TINYGO_VERSION) \
			-f DockerfileBuilder \
			-t ${BUILDER} \
			--push \
			.
	@echo ""
	@echo "image: ${BUILDER}"
