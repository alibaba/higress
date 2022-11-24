FROM registry.cn-hangzhou.aliyuncs.com/dspo/golang:1.19 as builder

WORKDIR /workspace

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

RUN go build -a -o upstream_example main.go


FROM registry.cn-hangzhou.aliyuncs.com/dspo/centos:7

MAINTAINER zhongrun.czr@alibaba-inc.com

WORKDIR /

RUN yum install -y net-tools.x86_64

COPY --from=builder /workspace/upstream_example /upstream_example
RUN chmod 777 /upstream_example

USER 65532:65532

ENTRYPOINT ["/upstream_example"]
