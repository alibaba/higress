ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.19-tinygo0.28.1-oras1.0.0
FROM $BUILDER as builder


ARG GOPROXY
ENV GOPROXY=${GOPROXY}

ARG PLUGIN_NAME=hello-world

WORKDIR /workspace

COPY . .

WORKDIR /workspace/extensions/$PLUGIN_NAME

RUN go mod tidy
RUN tinygo build -o /main.wasm -scheduler=none -gc=custom -tags='custommalloc nottinygc_finalizer' -target=wasi ./main.go

FROM scratch as output

COPY --from=builder /main.wasm plugin.wasm
