SHELL := /bin/bash -o pipefail

export BASE_VERSION ?= 2022-10-27T19-02-22

export HUB ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/higress

export CHARTS ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/charts

GO ?= go

GOARCH_LOCAL := $(TARGET_ARCH)
GOOS_LOCAL := $(TARGET_OS)
RELEASE_LDFLAGS='-extldflags -static -s -w'

export OUT:=$(TARGET_OUT)
export OUT_LINUX:=$(TARGET_OUT_LINUX)

# If tag not explicitly set in users' .istiorc.mk or command line, default to the git sha.
TAG ?= $(shell git rev-parse --verify HEAD)
ifeq ($(TAG),)
  $(error "TAG cannot be empty")
endif

VARIANT :=
ifeq ($(VARIANT),)
  TAG_VARIANT:=${TAG}
else
  TAG_VARIANT:=${TAG}-${VARIANT}
endif

HIGRESS_DOCKER_BUILD_TOP:=${OUT_LINUX}/docker_build

BINARIES:=./cmd/higress

$(OUT):
	@mkdir -p $@

submodule:
	git submodule update --init

prebuild: submodule
	./script/prebuild.sh

.PHONY: build
build: prebuild $(OUT)
	GOOS=$(GOOS_LOCAL) GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) script/gobuild.sh $(OUT)/ $(BINARIES)

.PHONY: build-linux
build-linux: prebuild $(OUT)
	GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) script/gobuild.sh $(OUT_LINUX)/ $(BINARIES)

# Create targets for OUT_LINUX/binary
# There are two use cases here:
# * Building all docker images (generally in CI). In this case we want to build everything at once, so they share work
# * Building a single docker image (generally during dev). In this case we just want to build the single binary alone
BUILD_ALL ?= true
define build-linux
.PHONY: $(OUT_LINUX)/$(shell basename $(1))
ifeq ($(BUILD_ALL),true)
$(OUT_LINUX)/$(shell basename $(1)): build-linux
else
$(OUT_LINUX)/$(shell basename $(1)): $(OUT_LINUX)
	GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) script/gobuild.sh $(OUT_LINUX)/ -tags=$(2) $(1)
endif
endef

$(foreach bin,$(BINARIES),$(eval $(call build-linux,$(bin),"")))

# Create helper targets for each binary, like "pilot-discovery"
# As an optimization, these still build everything
$(foreach bin,$(BINARIES),$(shell basename $(bin))): build
ifneq ($(OUT_LINUX),$(LOCAL_OUT))
# if we are on linux already, then this rule is handled by build-linux above, which handles BUILD_ALL variable
$(foreach bin,$(BINARIES),${LOCAL_OUT}/$(shell basename $(bin))): build
endif

.PHONY: push

# for now docker is limited to Linux compiles - why ?
include docker/docker.mk

docker-build: docker.higress ## Build and push docker images to registry defined by $HUB and $TAG

export PARENT_GIT_TAG:=$(shell git describe --tags)
export PARENT_GIT_REVISION:=$(TAG)

export ENVOY_TAR_PATH:=/home/package/envoy.tar.gz

build-istio: prebuild
	cd external/istio; GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=amd64 BUILD_WITH_CONTAINER=1 DOCKER_BUILD_VARIANTS=default DOCKER_TARGETS="docker.pilot" make docker

external/package/envoy.tar.gz:
	cd external/proxy; BUILD_WITH_CONTAINER=1  make test_release

build-gateway: prebuild external/package/envoy.tar.gz
	cd external/istio; GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=amd64 BUILD_WITH_CONTAINER=1 DOCKER_BUILD_VARIANTS=default DOCKER_TARGETS="docker.proxyv2" make docker

helm-push:
	cd helm; tar -zcf higress.tgz higress; helm push higress.tgz "oci://$(CHARTS)"

helm-push-istio:
	cd helm/istio; helm dependency update
	cd helm; tar -zcf istio.tgz istio; helm push istio.tgz "oci://$(CHARTS)"

helm-push-kind:
	cd helm/kind/higress; helm dependency update
	cd helm/kind; tar -zcf higress.tgz higress; helm push higress.tgz "oci://$(CHARTS)"
	cd helm/kind/istio; helm dependency update
	cd helm/kind; tar -zcf istio.tgz istio; helm push istio.tgz "oci://$(CHARTS)"


DIRS_TO_CLEAN := $(OUT)
DIRS_TO_CLEAN += $(OUT_LINUX)

clean-higress: ## Cleans all the intermediate files and folders previously generated.
	rm -rf $(DIRS_TO_CLEAN)

clean-istio:
	rm -rf external/istio

clean-gateway: clean-istio
	rm -rf external/envoy
	rm -rf external/proxy
	rm external/package/envoy.tar.gz

clean: clean-higress clean-gateway
