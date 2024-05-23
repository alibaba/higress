ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.19-tinygo0.28.1-oras1.0.0
FROM $BUILDER as builder

ENV GO111MODULE=on \
  GOPROXY=https://goproxy.cn,direct

WORKDIR /workspace

COPY . .

RUN go mod tidy
RUN tinygo build -o /main.wasm -scheduler=none -gc=custom -tags='custommalloc nottinygc_finalizer' -target=wasi ./

FROM scratch as output

COPY --from=builder /main.wasm plugin.wasm

