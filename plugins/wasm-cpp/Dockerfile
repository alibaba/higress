ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools-proxy:release-1.12-2021-12-09T23-01-43
FROM $BUILDER as builder

ARG PLUGIN_NAME

WORKDIR /workspace

COPY . .

RUN bazel build //extensions/$PLUGIN_NAME:$PLUGIN_NAME.wasm

FROM scratch as output

ARG PLUGIN_NAME

COPY --from=builder /workspace/bazel-bin/extensions/$PLUGIN_NAME/$PLUGIN_NAME.wasm plugin.wasm