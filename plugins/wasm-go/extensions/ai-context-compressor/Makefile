# Makefile for AI Context Compressor Wasm Plugin

PLUGIN_NAME := ai-context-compressor
PLUGIN_VERSION := $(shell cat VERSION)

# Build the wasm binary
build:
	@echo "Building $(PLUGIN_NAME) Wasm plugin..."
	docker run --rm -v $(PWD):/workspace ghcr.io/tetratelabs/wasm-builder:latest /workspace/main.go -o /workspace/main.wasm
	@echo "Build completed: main.wasm"

# Build and package as OCI image
package: build
	@echo "Packaging $(PLUGIN_NAME) as OCI image..."
	wasme build preloaded ./main.wasm --tag $(PLUGIN_NAME):$(PLUGIN_VERSION) --config runtime-config.json
	@echo "Packaging completed: $(PLUGIN_NAME):$(PLUGIN_VERSION)"

# Push to registry
push: package
	@echo "Pushing $(PLUGIN_NAME):$(PLUGIN_VERSION) to registry..."
	wasme push $(PLUGIN_NAME):$(PLUGIN_VERSION)
	@echo "Push completed"

# Clean build artifacts
clean:
	rm -f main.wasm
	@echo "Clean completed"

.PHONY: build package push clean