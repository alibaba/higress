# Single stage build using pre-built WASM binary
FROM scratch

ARG SERVER_NAME=quark-search

WORKDIR /

# Copy the pre-built WASM binary from local build
COPY ${SERVER_NAME}/main.wasm /plugin.wasm

# Metadata
LABEL org.opencontainers.image.title="${SERVER_NAME}"
LABEL org.opencontainers.image.description="Higress MCP Server - ${SERVER_NAME}"
LABEL org.opencontainers.image.source="https://github.com/alibaba/higress"

# The WASM binary is the only artifact in the image
