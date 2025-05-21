## Copyright 2018 Istio Authors
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##     http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.
docker.higress: BUILD_ARGS=--build-arg BASE_VERSION=${HIGRESS_BASE_VERSION} --build-arg HUB=${HUB} --build-arg PARENT_GIT_TAG=${PARENT_GIT_TAG} --build-arg PARENT_GIT_REVISION=${PARENT_GIT_REVISION}
docker.higress: $(OUT_LINUX)/higress
docker.higress: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_RULE)
docker.higress-buildx: BUILD_ARGS=--build-arg BASE_VERSION=${HIGRESS_BASE_VERSION} --build-arg HUB=${HUB} --build-arg PARENT_GIT_TAG=${PARENT_GIT_TAG} --build-arg PARENT_GIT_REVISION=${PARENT_GIT_REVISION}
docker.higress-buildx: $(AMD64_OUT_LINUX)/higress
docker.higress-buildx: $(ARM64_OUT_LINUX)/higress
docker.higress-buildx: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_BUILDX_RULE)
# Base images have different forms:
# * "debug", suffixed as -debug. This is a ubuntu based image with a bunch of debug tools
# * "minimal", suffixed as -minimal. This is an Alpine-based minimal image with basic tools
# * "distroless", suffixed as -distroless. This is distroless image - no shell
# * "default", no suffix. This is currently "debug"
DOCKER_BUILD_VARIANTS ?= default
DOCKER_ALL_VARIANTS ?= debug minimal distroless
# If INCLUDE_UNTAGGED_DEFAULT is set, then building the "DEFAULT_DISTRIBUTION" variant will publish both <tag>-<variant> and <tag>
# This can be done with DOCKER_BUILD_VARIANTS="default debug" as well, but at the expense of building twice vs building once and tagging twice
INCLUDE_UNTAGGED_DEFAULT ?= false
DEFAULT_DISTRIBUTION=debug
IMG ?= higress
IMG_URL ?= $(HUB)/$(IMG):$(TAG)

# Function to normalize variant name for target use
normalize-variant = $(if $(filter default,$(1)),$(DEFAULT_DISTRIBUTION),$(1))

# Function to add variant suffix to tag
variant-tag = $(if $(filter default,$(1)),$(if $(INCLUDE_UNTAGGED_DEFAULT),,-$(call normalize-variant,$(1))),-$(call normalize-variant,$(1)))

# Create a temporary minimal Dockerfile for Alpine
define create-minimal-dockerfile
	@echo "Creating minimal Dockerfile for Alpine..."
	@mkdir -p $(dir $1)
	@echo "FROM alpine:latest" > $1
	@echo "ARG PARENT_GIT_TAG" >> $1
	@echo "ARG PARENT_GIT_REVISION" >> $1
	@echo "RUN apk --no-cache add ca-certificates && \\" >> $1
	@echo "    adduser -D -u 1337 istio-proxy" >> $1
	@echo "COPY amd64/higress /usr/local/bin/higress" >> $1
	@echo "USER 1337:1337" >> $1
	@echo "ENTRYPOINT [\"/usr/local/bin/higress\"]" >> $1
endef

HIGRESS_DOCKER_BUILDX_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), \
	if [ "$(call normalize-variant,$(VARIANT))" = "minimal" ]; then \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp && \
		$(call create-minimal-dockerfile,$(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp/Dockerfile.higress.minimal) && \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cp $(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp/Dockerfile.higress.minimal $(HIGRESS_DOCKER_BUILD_TOP)/$@/Dockerfile.higress && \
		TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $(AMD64_OUT_LINUX)/higress $(ARM64_OUT_LINUX)/higress $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && \
		docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use && \
		docker buildx build --no-cache --platform linux/amd64,linux/arm64 $(BUILD_ARGS) \
		-t $(IMG_URL)$(call variant-tag,$(VARIANT)) \
		-f Dockerfile.higress . --push; \
	else \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cp docker/Dockerfile.higress $(HIGRESS_DOCKER_BUILD_TOP)/$@/Dockerfile.higress && \
		TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $(AMD64_OUT_LINUX)/higress $(ARM64_OUT_LINUX)/higress $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && \
		docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use && \
		docker buildx build --no-cache --platform linux/amd64,linux/arm64 $(BUILD_ARGS) \
		--build-arg BASE_DISTRIBUTION=$(call normalize-variant,$(VARIANT)) \
		--target $(call normalize-variant,$(VARIANT)) \
		-t $(IMG_URL)$(call variant-tag,$(VARIANT)) \
		-f Dockerfile.higress . --push; \
	fi; )

HIGRESS_DOCKER_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), \
	if [ "$(call normalize-variant,$(VARIANT))" = "minimal" ]; then \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp && \
		$(call create-minimal-dockerfile,$(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp/Dockerfile.higress.minimal) && \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cp $(HIGRESS_DOCKER_BUILD_TOP)/$@/tmp/Dockerfile.higress.minimal $(HIGRESS_DOCKER_BUILD_TOP)/$@/Dockerfile.higress && \
		TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $(OUT_LINUX)/higress $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && \
		docker build $(BUILD_ARGS) \
		-t $(IMG_URL)$(call variant-tag,$(VARIANT)) \
		-f Dockerfile.higress .; \
	else \
		mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cp docker/Dockerfile.higress $(HIGRESS_DOCKER_BUILD_TOP)/$@/Dockerfile.higress && \
		TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $(OUT_LINUX)/higress $(HIGRESS_DOCKER_BUILD_TOP)/$@ && \
		cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && \
		docker build $(BUILD_ARGS) \
		--build-arg BASE_DISTRIBUTION=$(call normalize-variant,$(VARIANT)) \
		--target $(call normalize-variant,$(VARIANT)) \
		-t $(IMG_URL)$(call variant-tag,$(VARIANT)) \
		-f Dockerfile.higress .; \
	fi; )