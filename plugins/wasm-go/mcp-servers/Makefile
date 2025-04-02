# MCP Server Makefile

# Variables
SERVER_NAME ?= quark-search
REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/mcp-server/
GO_VERSION ?= 1.24
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMAGE_TAG = $(if $(strip $(SERVER_VERSION)),${SERVER_VERSION},${BUILD_TIME}-${COMMIT_ID})
IMG ?= ${REGISTRY}${SERVER_NAME}:${IMAGE_TAG}
GOPROXY := $(shell go env GOPROXY)

# Default target
.DEFAULT:
build:
	@echo "Building WASM binary for ${SERVER_NAME}..."
	cd ${SERVER_NAME} && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go
	@echo ""
	@echo "Output WASM file: ${SERVER_NAME}/main.wasm"

# Build Docker image (depends on build target to ensure WASM binary exists)
build-image: build
	@echo "Building Docker image for ${SERVER_NAME}..."
	docker build -t ${IMG} \
		--build-arg SERVER_NAME=${SERVER_NAME} \
		--build-arg GOPROXY=${GOPROXY} \
		-f Dockerfile .
	@echo ""
	@echo "Image: ${IMG}"

# Build and push Docker image
build-push: build-image
	docker push ${IMG}

# Clean build artifacts
clean:
	rm -f ${SERVER_NAME}/main.wasm

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build WASM binary"
	@echo "  build-image  - Build Docker image"
	@echo "  build-push   - Build and push Docker image"
	@echo "  clean        - Remove build artifacts"
	@echo ""
	@echo "Variables:"
	@echo "  SERVER_NAME  - Name of the MCP server (default: ${SERVER_NAME})"
	@echo "  REGISTRY     - Docker registry (default: ${REGISTRY})"
	@echo "  SERVER_VERSION - Version tag for the image (default: timestamp-commit)"
	@echo "  IMG          - Full image name (default: ${IMG})"
