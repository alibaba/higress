ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.19-tinygo0.27.0
FROM $BUILDER as builder

ARG PLUGIN_NAME=hello-world

WORKDIR /workspace

COPY . .

WORKDIR /workspace/extensions/$PLUGIN_NAME

RUN go mod tidy
RUN tinygo build -o /main.wasm -scheduler=none -target=wasi ./main.go

FROM scratch as output

COPY --from=builder /main.wasm plugin.wasm
