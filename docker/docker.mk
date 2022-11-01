docker.higress: BUILD_ARGS=--build-arg BASE_VERSION=${BASE_VERSION} --build-arg HUB=${HUB}
docker.higress: $(OUT_LINUX)/higress
docker.higress: docker/Dockerfile.higress
	$(HIGRESS_DOCKER_RULE)

# DOCKER_BUILD_VARIANTS ?=debug distroless
# Base images have two different forms:
# * "debug", suffixed as -debug. This is a ubuntu based image with a bunch of debug tools
# * "distroless", suffixed as -distroless. This is distroless image - no shell. proxyv2 uses a custom one with iptables added
# * "default", no suffix. This is currently "debug"
DOCKER_BUILD_VARIANTS ?= default
DOCKER_ALL_VARIANTS ?= debug distroless
# If INCLUDE_UNTAGGED_DEFAULT is set, then building the "DEFAULT_DISTRIBUTION" variant will publish both <tag>-<variant> and <tag>
# This can be done with DOCKER_BUILD_VARIANTS="default debug" as well, but at the expense of building twice vs building once and tagging twice
INCLUDE_UNTAGGED_DEFAULT ?= false
DEFAULT_DISTRIBUTION=debug
HIGRESS_DOCKER_RULE ?= $(foreach VARIANT,$(DOCKER_BUILD_VARIANTS), time (mkdir -p $(HIGRESS_DOCKER_BUILD_TOP)/$@ && TARGET_ARCH=$(TARGET_ARCH) ./docker/docker-copy.sh $^ $(HIGRESS_DOCKER_BUILD_TOP)/$@ && cd $(HIGRESS_DOCKER_BUILD_TOP)/$@ $(BUILD_PRE) && docker build $(BUILD_ARGS) --build-arg BASE_DISTRIBUTION=$(call normalize-tag,$(VARIANT)) -t $(HUB)/$(subst docker.,,$@):$(TAG)$(call variant-tag,$(VARIANT)) -f Dockerfile$(suffix $@) . ); )
