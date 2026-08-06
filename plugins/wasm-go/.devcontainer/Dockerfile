FROM higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/gateway:1.3.1

FROM ubuntu:20.04

RUN apt -y update \
    && apt install -y --no-install-recommends python3-pip net-tools vim wget make curl git 2>&1 \
    && apt install -y --reinstall ca-certificates \
    && apt-get autoremove -y && apt-get clean \
    && rm -rf /tmp/* /var/tmp/* \
    && rm -rf /var/lib/apt/lists/*

ENV PATH=/opt/tinygo/bin:/opt/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN wget --no-check-certificate https://github.com/tinygo-org/tinygo/releases/download/v0.29.0/tinygo0.29.0.linux-amd64.tar.gz \
    && tar -zvxf tinygo0.29.0.linux-amd64.tar.gz -C /opt \
    && rm tinygo0.29.0.linux-amd64.tar.gz

RUN wget --no-check-certificate https://go.dev/dl/go1.19.linux-amd64.tar.gz \
    && tar -zvxf go1.19.linux-amd64.tar.gz -C /opt \
    && rm go1.19.linux-amd64.tar.gz \
    && go install -v golang.org/x/tools/gopls@latest

COPY --from=0 /usr/local/bin/envoy /usr/local/bin/envoy