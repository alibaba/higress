# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# Modified by Higress

########################
# kubernetes code generators
########################
kubetype_gen = kubetype-gen
deepcopy_gen = deepcopy-gen
client_gen = client-gen
lister_gen = lister-gen
informer_gen = informer-gen

empty:=
space := $(empty) $(empty)
comma := ,

# source packages to scan for kubetype-gen tags
kube_source_packages = $(subst $(space),$(empty), \
	github.com/alibaba/higress/api/networking/v1, \
	github.com/alibaba/higress/api/extensions/v1alpha1 \
	)

# base output package for generated files
kube_base_output_package = github.com/alibaba/higress/client/pkg
# base output package for kubernetes types, register, etc...
kube_api_base_package = $(kube_base_output_package)/apis
# source packages to scan for kubernetes generator tags, e.g. deepcopy-gen, client-gen, etc.
# these should correspond to the output packages from kubetype-gen
kube_api_packages = $(subst $(space),$(empty), \
	$(kube_api_base_package)/networking/v1, \
	$(kube_api_base_package)/extensions/v1alpha1 \
	)
# base output package used by kubernetes client-gen
kube_clientset_package = $(kube_base_output_package)/clientset
# clientset name used by kubernetes client-gen
kube_clientset_name = versioned
# base output package used by kubernetes lister-gen
kube_listers_package = $(kube_base_output_package)/listers
# base output package used by kubernetes informer-gen
kube_informers_package = $(kube_base_output_package)/informers

# file header text
kube_go_header_text = header.go.txt

ifeq ($(IN_BUILD_CONTAINER),1)
	# k8s code generators rely on GOPATH, using $GOPATH/src as the base package
	# directory.  Using --output-base . does not work, as that ends up generating
	# code into ./<package>, e.g. ./istio.io/client-go/pkg/apis/...  To work
	# around this, we'll just let k8s generate the code where it wants and copy
	# back to where it should have been generated.
	move_generated=cp -r $(GOPATH)/src/$(kube_base_output_package)/ . && rm -rf $(GOPATH)/src/$(kube_base_output_package)/
else
	# nothing special for local builds
	move_generated=
endif

rename_generated_files=\
	find $(subst github.com/alibaba/higress/client/, $(empty), $(subst $(comma), $(space), $(kube_api_packages)) $(kube_clientset_package) $(kube_listers_package) $(kube_informers_package)) \
	-name '*.go' -and -not -name 'doc.go' -and -not -name '*.gen.go' -type f -exec sh -c 'mv "$$1" "$${1%.go}".gen.go' - '{}' \;

.PHONY: generate-k8s-client
generate-k8s-client:
	# generate kube api type wrappers for higress types
	@$(kubetype_gen) --input-dirs $(kube_source_packages) --output-package $(kube_api_base_package) -h $(kube_go_header_text)
	@$(move_generated)
	# generate deepcopy for kube api types
	@$(deepcopy_gen) --input-dirs $(kube_api_packages) -O zz_generated.deepcopy  -h $(kube_go_header_text)
	# generate clientsets for kube api types
	@$(client_gen) --clientset-name $(kube_clientset_name) --input-base "" --input  $(kube_api_packages) --output-package $(kube_clientset_package) -h $(kube_go_header_text)
	# generate listers for kube api types
	@$(lister_gen) --input-dirs $(kube_api_packages) --output-package $(kube_listers_package) -h $(kube_go_header_text)
	# generate informers for kube api types
	@$(informer_gen) --input-dirs $(kube_api_packages) --versioned-clientset-package $(kube_clientset_package)/$(kube_clientset_name) --listers-package $(kube_listers_package) --output-package $(kube_informers_package) -h $(kube_go_header_text)
	@$(move_generated)
	@$(rename_generated_files)


.PHONY: clean-k8s-client
clean-k8s-client:
    # remove generated code
	@rm -rf pkg/
