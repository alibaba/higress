SHELL := /bin/bash -o pipefail

export BASE_VERSION ?= 2022-10-27T19-02-22

export HUB ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/higress

export CHARTS ?= higress-registry.cn-hangzhou.cr.aliyuncs.com/charts

VERSION_PACKAGE := github.com/alibaba/higress/pkg/cmd/version

GIT_COMMIT:=$(shell git rev-parse HEAD)

GO_LDFLAGS += -X $(VERSION_PACKAGE).higressVersion=$(shell cat VERSION) \
	-X $(VERSION_PACKAGE).gitCommitID=$(GIT_COMMIT)

GO ?= go

export GOPROXY ?= https://proxy.golang.com.cn,direct

GOARCH_LOCAL := $(TARGET_ARCH)
GOOS_LOCAL := $(TARGET_OS)
RELEASE_LDFLAGS='$(GO_LDFLAGS) -extldflags -static -s -w'

export OUT:=$(TARGET_OUT)
export OUT_LINUX:=$(TARGET_OUT_LINUX)

BUILDX_PLATFORM ?=

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

HIGRESS_BINARIES:=./cmd/higress

HGCTL_BINARIES:=./cmd/hgctl

$(OUT):
	@mkdir -p $@

submodule:
	git submodule update --init

prebuild: submodule
	./tools/hack/prebuild.sh

.PHONY: default
default: build

.PHONY: go.test.coverage
go.test.coverage: prebuild
	go test ./cmd/... ./pkg/... -race -coverprofile=coverage.xml -covermode=atomic

.PHONY: build
build: prebuild $(OUT)
	GOPROXY=$(GOPROXY) GOOS=$(GOOS_LOCAL) GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT)/ $(HIGRESS_BINARIES)

.PHONY: build-linux
build-linux: prebuild $(OUT)
ifeq ($(BUILDX_PLATFORM), true)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_amd64/ $(HIGRESS_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_arm64/ $(HIGRESS_BINARIES)
else
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ $(HIGRESS_BINARIES)
endif


.PHONY: build-hgctl
build-hgctl: $(OUT)
	GOPROXY=$(GOPROXY) GOOS=$(GOOS_LOCAL) GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT)/ $(HGCTL_BINARIES)

.PHONY: build-linux-hgctl
build-linux-hgctl: $(OUT)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ $(HGCTL_BINARIES)

.PHONY: build-hgctl-multiarch
build-hgctl-multiarch: $(OUT)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_amd64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/linux_arm64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=amd64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/darwin_amd64/ $(HGCTL_BINARIES)
	GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=arm64 LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh ./out/darwin_arm64/ $(HGCTL_BINARIES)

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
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=$(GOARCH_LOCAL) LDFLAGS=$(RELEASE_LDFLAGS) tools/hack/gobuild.sh $(OUT_LINUX)/ -tags=$(2) $(1)
endif
endef

$(foreach bin,$(HIGRESS_BINARIES),$(eval $(call build-linux,$(bin),"")))

# Create helper targets for each binary, like "pilot-discovery"
# As an optimization, these still build everything
$(foreach bin,$(HIGRESS_BINARIES),$(shell basename $(bin))): build
ifneq ($(OUT_LINUX),$(LOCAL_OUT))
# if we are on linux already, then this rule is handled by build-linux above, which handles BUILD_ALL variable
$(foreach bin,$(HIGRESS_BINARIES),${LOCAL_OUT}/$(shell basename $(bin))): build
endif

.PHONY: push

# for now docker is limited to Linux compiles - why ?
include docker/docker.mk

docker-build: docker.higress ## Build and push docker images to registry defined by $HUB and $TAG

docker-build-base:
	docker buildx build --no-cache --platform linux/amd64,linux/arm64 -t ${HUB}/base:${BASE_VERSION} -f docker/Dockerfile.base . --push

export PARENT_GIT_TAG:=$(shell cat VERSION)
export PARENT_GIT_REVISION:=$(TAG)

export ENVOY_TAR_PATH:=/home/package/envoy.tar.gz

external/package/envoy.tar.gz:
	cd external/proxy; BUILD_WITH_CONTAINER=1  make test_release

build-gateway: prebuild external/package/envoy.tar.gz
	cd external/istio; rm -rf out; GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=amd64 BUILD_WITH_CONTAINER=1 DOCKER_BUILD_VARIANTS=default DOCKER_TARGETS="docker.proxyv2" make docker

build-istio: prebuild
	cd external/istio; rm -rf out; GOOS_LOCAL=linux TARGET_OS=linux TARGET_ARCH=amd64 BUILD_WITH_CONTAINER=1 DOCKER_BUILD_VARIANTS=default DOCKER_TARGETS="docker.pilot" make docker

build-wasmplugins: 
	./tools/hack/build-wasm-plugins.sh

pre-install:
	cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds

define create_ns
   kubectl get namespace | grep $(1) || kubectl create namespace $(1)
endef

install: pre-install
	cd helm/higress; helm dependency build
	helm install higress helm/higress -n higress-system --create-namespace --set 'global.local=true'

ENVOY_LATEST_IMAGE_TAG ?= 1.1.0
ISTIO_LATEST_IMAGE_TAG ?= 1.1.0

install-dev: pre-install
	helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=$(TAG)' --set 'gateway.replicas=1' --set 'gateway.tag=$(ENVOY_LATEST_IMAGE_TAG)' --set 'global.local=true'

install-dev-wasmplugin: build-wasmplugins pre-install
	helm install higress helm/core -n higress-system --create-namespace --set 'controller.tag=$(TAG)' --set 'gateway.replicas=1' --set 'gateway.tag=$(ENVOY_LATEST_IMAGE_TAG)' --set 'global.local=true'  --set 'global.volumeWasmPlugins=true'

uninstall:
	helm uninstall higress -n higress-system

upgrade: pre-install
	cd helm/higress; helm dependency build
	helm upgrade higress helm/higress -n higress-system --set 'global.local=true'

helm-push:
	cp api/kubernetes/customresourcedefinitions.gen.yaml helm/core/crds
	cd helm; tar -zcf higress.tgz higress; helm push higress.tgz "oci://$(CHARTS)"

cue = cue-gen -paths=./external/api/common-protos

gen-api: prebuild
	cd api;./gen.sh

gen-client: gen-api
	cd client; make generate-k8s-client

DIRS_TO_CLEAN := $(OUT)
DIRS_TO_CLEAN += $(OUT_LINUX)

clean-higress: ## Cleans all the intermediate files and folders previously generated.
	rm -rf $(DIRS_TO_CLEAN)

clean-istio:
	rm -rf external/istio

clean-gateway: clean-istio
	rm -rf external/envoy
	rm -rf external/proxy
	rm -rf external/package/envoy.tar.gz

clean-env:
	rm -rf out/

clean-tool:
	rm -rf tools/bin

clean: clean-higress clean-gateway clean-istio clean-env clean-tool

include tools/tools.mk
include tools/lint.mk

# gateway-conformance-test runs gateway api conformance tests.
.PHONY: gateway-conformance-test
gateway-conformance-test:

# higress-conformance-test runs ingress api conformance tests.
.PHONY: higress-conformance-test
higress-conformance-test: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev run-higress-e2e-test delete-cluster

# higress-wasmplugin-test runs ingress wasmplugin tests.
.PHONY: higress-wasmplugin-test
higress-wasmplugin-test: $(tools/kind) delete-cluster create-cluster docker-build kube-load-image install-dev-wasmplugin run-higress-e2e-test-wasmplugin delete-cluster

# create-cluster creates a kube cluster with kind.
.PHONY: create-cluster
create-cluster: $(tools/kind)
	tools/hack/create-cluster.sh

# delete-cluster deletes a kube cluster.
.PHONY: delete-cluster
delete-cluster: $(tools/kind) ## Delete kind cluster.
	$(tools/kind) delete cluster --name higress

# kube-load-image loads a local built docker image into kube cluster.
# dubbo-provider-demo和nacos-standlone-rc3的镜像已经上传到阿里云镜像库，第一次需要先拉到本地
# docker pull registry.cn-hangzhou.aliyuncs.com/hinsteny/dubbo-provider-demo:0.0.1
# docker pull registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3:1.0.0-RC3
.PHONY: kube-load-image
kube-load-image: $(tools/kind) ## Install the Higress image to a kind cluster using the provided $IMAGE and $TAG.
	tools/hack/kind-load-image.sh higress-registry.cn-hangzhou.cr.aliyuncs.com/higress/higress $(TAG)
	tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/dubbo-provider-demo 0.0.1
	tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
	tools/hack/docker-pull-image.sh docker.io/hashicorp/consul 1.16.0
	tools/hack/docker-pull-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
	tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/dubbo-provider-demo 0.0.1
	tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/hinsteny/nacos-standlone-rc3 1.0.0-RC3
	tools/hack/kind-load-image.sh docker.io/hashicorp/consul 1.16.0
	tools/hack/kind-load-image.sh registry.cn-hangzhou.aliyuncs.com/2456868764/httpbin 1.0.2
# run-higress-e2e-test starts to run ingress e2e tests.
.PHONY: run-higress-e2e-test
run-higress-e2e-test:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go --ingress-class=higress --debug=true

# run-higress-e2e-test starts to run ingress e2e tests.
.PHONY: run-higress-e2e-test-wasmplugin
run-higress-e2e-test-wasmplugin:
	@echo -e "\n\033[36mRunning higress conformance tests...\033[0m"
	@echo -e "\n\033[36mWaiting higress-controller to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-controller --for=condition=Available
	@echo -e "\n\033[36mWaiting higress-gateway to be ready...\033[0m\n"
	kubectl wait --timeout=10m -n higress-system deployment/higress-gateway --for=condition=Available
	go test -v -tags conformance ./test/e2e/e2e_test.go -isWasmPluginTest=true -wasmPluginType=$(PLUGIN_TYPE) -wasmPluginName=$(PLUGIN_NAME) --ingress-class=higress --debug=true
