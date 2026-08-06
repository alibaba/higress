FROM golang:latest AS builder
WORKDIR /workspace
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o main .
CMD ./main