ARG BUILDER=higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins/wasm-go-builder:go1.24.0-oras1.0.0
FROM $BUILDER AS builder


ARG GOPROXY
ENV GOPROXY=${GOPROXY}

ARG EXTRA_TAGS=""
ENV EXTRA_TAGS=${EXTRA_TAGS}

ARG PLUGIN_NAME=hello-world

WORKDIR /workspace

COPY . .

WORKDIR /workspace/extensions/$PLUGIN_NAME

RUN go mod tidy
RUN \
    GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o /main.wasm ./ 

FROM scratch AS output

COPY --from=builder /main.wasm plugin.wasm
