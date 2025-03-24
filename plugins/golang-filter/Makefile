GO_FILTER_NAME ?= mcp-server
GOPROXY := $(shell go env GOPROXY)

.DEFAULT:
build:
	DOCKER_BUILDKIT=1 docker build --build-arg GOPROXY=$(GOPROXY) \
								    --build-arg GO_FILTER_NAME=${GO_FILTER_NAME} \
									-t ${GO_FILTER_NAME} \
									--output ./${GO_FILTER_NAME} \
									.