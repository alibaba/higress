FROM ubuntu as builder

RUN apt-get update \
  && apt-get install -y wget \
  && rm -rf /var/lib/apt/lists/*

RUN wget https://golang.google.cn/dl/go1.19.3.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.3.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

RUN wget https://github.com/tinygo-org/tinygo/releases/download/v0.25.0/tinygo_0.25.0_amd64.deb
RUN dpkg -i tinygo_0.25.0_amd64.deb
ENV export PATH=$PATH:/usr/local/bin

WORKDIR /workspace

COPY . .

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod tidy
RUN tinygo build -o main.wasm -scheduler=none -target=wasi ./main.go

FROM scratch

COPY --from=builder /workspace/main.wasm plugin.wasm
