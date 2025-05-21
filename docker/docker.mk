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
# * "default", no suffix. This is currently "minimal"
DOCKER_BUILD_VARIANTS ?= default
DOCKER_ALL_VARIANTS ?= debug minimal distroless
# If INCLUDE_UNTAGGED_DEFAULT is set, then building the "DEFAULT_DISTRIBUTION" variant will publish both <tag>-<variant> and <tag>
# This can be done with DOCKER_BUILD_VARIANTS="default minimal" as well, but at the expense of building twice vs building once and tagging twice
INCLUDE_UNTAGGED_DEFAULT ?= false
DEFAULT_DISTRIBUTION=minimal
IMG ?= higress
IMG_URL ?= $(HUB)/$(IMG):$(TAG)

# Function to normalize variant name for target use
normalize-tag = $(if $(filter default,$(1)),$(DEFAULT_DISTRIBUTION),$(1))

# Function to add variant suffix to tag
variant-tag = $(if $(filter default,$(1)),$(if $(INCLUDE_UNTAGGED_DEFAULT),,-$(call normalize-tag,$(1))),-$(call normalize-tag,$(1)))

HIGRESS_DOCKER_BUILDX_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker buildx create --name higress --node higress0 --platform linux/amd64,linux/arm64 --use && docker buildx build --no-cache --platform linux/amd64,linux/arm64 $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) --target $(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . --push  ); )

HIGRESS_DOCKER_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) --target $(call normalize-tag,$(VARIANT)) -t $(IMG_URL)$(call variant-tag,$(VARIANT)) -f Dockerfile.higress . ); )