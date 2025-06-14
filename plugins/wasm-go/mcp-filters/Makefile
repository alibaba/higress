# MCP Filter Makefile

# Variables
FILTER_NAME ?= mcp-router
REGISTRY ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/
BUILD_TIME := $(shell date "+%Y%m%d-%H%M%S")
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null)
IMAGE_TAG = $(if $(strip $(FILTER_VERSION)),${FILTER_VERSION},${BUILD_TIME}-${COMMIT_ID})
IMG ?= ${REGISTRY}${FILTER_NAME}:${IMAGE_TAG}

# Default target
.DEFAULT: build

build:
	@echo "Building WASM binary for filter: ${FILTER_NAME}..."
	@if [ ! -d "${FILTER_NAME}" ]; then \
		echo "Error: Filter directory '${FILTER_NAME}' not found."; \
		exit 1; \
	fi
	cd ${FILTER_NAME} && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go
	@echo ""
	@echo "Output WASM file: ${FILTER_NAME}/main.wasm"

# Build Docker image (depends on build target to ensure WASM binary exists)
build-image: build
	@echo "Building Docker image for ${FILTER_NAME}..."
	docker build -t ${IMG} \
		--build-arg FILTER_NAME=${FILTER_NAME} \
		-f Dockerfile .
	@echo ""
	@echo "Image: ${IMG}"

# Build and push Docker image
build-push: build-image
	docker push ${IMG}

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts for filter: ${FILTER_NAME}..."
	rm -f ${FILTER_NAME}/main.wasm

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build WASM binary for a specific filter"
	@echo "  build-image  - Build Docker image"
	@echo "  build-push   - Build and push Docker image"
	@echo "  clean        - Remove build artifacts for a specific filter"
	@echo ""
	@echo "Variables:"
	@echo "  FILTER_NAME    - Name of the MCP filter to build (default: ${FILTER_NAME})"
	@echo "  REGISTRY       - Docker registry (default: ${REGISTRY})"
	@echo "  FILTER_VERSION - Version tag for the image (default: timestamp-commit)"
	@echo "  IMG            - Full image name (default: ${IMG})"
