ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/build-tools-proxy:release-1.19-04ab00931b61c082300832a7dd51634e5e3634ad
FROM $BUILDER as builder

ARG PLUGIN_NAME

WORKDIR /workspace

COPY . .

RUN bazel build //extensions/$PLUGIN_NAME:$PLUGIN_NAME.wasm

FROM scratch as output

ARG PLUGIN_NAME

COPY --from=builder /workspace/bazel-bin/extensions/$PLUGIN_NAME/$PLUGIN_NAME.wasm plugin.wasm