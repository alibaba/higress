FROM rust:1.80 as builder
WORKDIR /workspace
RUN rustup target add wasm32-wasip1
ARG PLUGIN_NAME="say-hello"
ARG BUILD_OPTS="--release"
ARG PREBUILD=".prebuild"
ENV HTTPS_PROXY=http://127.0.0.1:7890
ENV HTTP_PROXY=http://127.0.0.1:7890
COPY . .
WORKDIR /workspace/extensions/$PLUGIN_NAME
RUN if [ -f $PREBUILD ]; then sh $PREBUILD; fi
RUN cargo build --target wasm32-wasip1 $BUILD_OPTS \
    && cp target/wasm32-wasip1/release/*.wasm /main.wasm \
	&& cargo clean

FROM scratch
COPY --from=builder /main.wasm plugin.wasm
