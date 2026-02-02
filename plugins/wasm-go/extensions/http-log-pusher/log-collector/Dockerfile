# Build Stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
# 禁用 CGO 确保静态链接
RUN CGO_ENABLED=0 GOOS=linux go build -o collector main.go

# Run Stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/collector .
EXPOSE 8080
CMD ["./collector"]