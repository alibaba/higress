# Build stage
FROM golang:1.24 AS builder

ARG SERVER_NAME=quark-search
ARG GOPROXY

WORKDIR /app

# Copy the server code
COPY ${SERVER_NAME}/ .

# Set GOPROXY if provided
RUN if [ -n "$GOPROXY" ]; then go env -w GOPROXY=${GOPROXY}; fi

# Build the WASM binary
RUN GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm main.go

# Final stage
FROM scratch

ARG SERVER_NAME=quark-search

WORKDIR /

# Copy the WASM binary from the builder stage
COPY --from=builder /app/main.wasm /main.wasm

# Metadata
LABEL org.opencontainers.image.title="${SERVER_NAME}"
LABEL org.opencontainers.image.description="Higress MCP Server - ${SERVER_NAME}"
LABEL org.opencontainers.image.source="https://github.com/alibaba/higress"

# The WASM binary is the only artifact in the image
