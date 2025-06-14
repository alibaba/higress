# Use a minimal base image as we only need to store the wasm file.
FROM scratch

# Add build argument for the filter name. This will be passed by the Makefile.
ARG FILTER_NAME

# Copy the compiled WASM binary into the image's root directory.
# The wasm file will be named after the filter.
COPY ${FILTER_NAME}/main.wasm /plugin.wasm

# Metadata
LABEL org.opencontainers.image.title="${FILTER_NAME}"
LABEL org.opencontainers.image.description="Higress MCP filter - ${FILTER_NAME}"
LABEL org.opencontainers.image.source="https://github.com/alibaba/higress"