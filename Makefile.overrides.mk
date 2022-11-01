.DEFAULT_GOAL := default

# This repository has been enabled for BUILD_WITH_CONTAINER=1. Some
# test cases fail within Docker, and Mac + Docker isn't quite perfect.
# For more information see: https://github.com/istio/istio/pull/19322/

BUILD_WITH_CONTAINER ?= 0
CONTAINER_OPTIONS = --mount type=bind,source=/tmp,destination=/tmp --net=host

ifeq ($(BUILD_WITH_CONTAINER),1)
# create phony targets for the top-level items in the repo
PHONYS := $(shell ls | grep -v Makefile)
.PHONY: $(PHONYS)
$(PHONYS):
	@$(MAKE_DOCKER) $@
endif
