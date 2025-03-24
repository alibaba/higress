FROM golang:1.23-bullseye AS golang-base

ARG GOPROXY
ARG GO_FILTER_NAME

ENV GOFLAGS=-buildvcs=false
ENV GOPROXY=${GOPROXY}

WORKDIR /workspace

COPY . .

WORKDIR /workspace/$GO_FILTER_NAME

RUN go mod tidy
RUN go build -o /$GO_FILTER_NAME.so -buildmode=c-shared .

FROM scratch AS output
ARG GO_FILTER_NAME
COPY --from=golang-base /$GO_FILTER_NAME.so $GO_FILTER_NAME.so