FROM rust:1.80 as builder
WORKDIR /workspace
RUN rustup target add wasm32-wasi
ARG PLUGIN_NAME="say-hello"
ARG BUILD_OPTS="--release"
ARG BUILDRC=".buildrc"
COPY . .
WORKDIR /workspace/extensions/$PLUGIN_NAME
RUN if [ -f $BUILDRC ]; then sh $BUILDRC; fi
RUN cargo build --target wasm32-wasi $BUILD_OPTS \
    && cp target/wasm32-wasi/release/*.wasm /main.wasm

FROM scratch
COPY --from=builder /main.wasm plugin.wasm
