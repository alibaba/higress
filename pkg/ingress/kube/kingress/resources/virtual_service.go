/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/network"
)

func MakeVirtualServiceRoute(hosts sets.String, http *v1alpha1.HTTPIngressPath) *istiov1alpha3.HTTPRoute {
	matches := []*istiov1alpha3.HTTPMatchRequest{}
	// Deduplicate hosts to avoid excessive matches, which cause a combinatorial expansion in Istio

	for _, host := range hosts.List() {
		matches = append(matches, makeMatch(host, http.Path, http.Headers))
	}

	weights := []*istiov1alpha3.HTTPRouteDestination{}
	for _, split := range http.Splits {
		var h *istiov1alpha3.Headers
		if len(split.AppendHeaders) > 0 {
			h = &istiov1alpha3.Headers{
				Request: &istiov1alpha3.Headers_HeaderOperations{
					Set: split.AppendHeaders,
				},
			}
		}

		weights = append(weights, &istiov1alpha3.HTTPRouteDestination{
			Destination: &istiov1alpha3.Destination{
				Host: network.GetServiceHostname(
					split.ServiceName, split.ServiceNamespace),
				Port: &istiov1alpha3.PortSelector{
					Number: uint32(split.ServicePort.IntValue()),
				},
			},
			Weight:  int32(split.Percent),
			Headers: h,
		})
	}

	var h *istiov1alpha3.Headers
	if len(http.AppendHeaders) > 0 {
		h = &istiov1alpha3.Headers{
			Request: &istiov1alpha3.Headers_HeaderOperations{
				Set: http.AppendHeaders,
			},
		}
	}

	var rewrite *istiov1alpha3.HTTPRewrite
	if http.RewriteHost != "" {
		rewrite = &istiov1alpha3.HTTPRewrite{
			Authority: http.RewriteHost,
		}
	}

	route := &istiov1alpha3.HTTPRoute{
		Retries: &istiov1alpha3.HTTPRetry{}, // Override default istio behaviour of retrying twice.
		Match:   matches,
		Route:   weights,
		Rewrite: rewrite,
		Headers: h,
	}
	return route
}

// getDistinctHostPrefixes deduplicate a set of prefix matches. For example, the set {a, aabb} can be
// reduced to {a}, as a prefix match on {a} accepts all the same inputs as {a, aabb}.
func getDistinctHostPrefixes(hosts sets.String) sets.String {
	// First we sort the list. This ensures that we always process the smallest elements (which match against
	// the most patterns, as they are less specific) first.
	all := hosts.List()
	ns := sets.NewString()
	for _, h := range all {
		prefixExists := false
		h = hostPrefix(h)
		// For each element, check if any existing elements are a prefix. We only insert if none are
		//		// For example, if we already have {a} and we are looking at "ab", we would not add it as it has a prefix of "a"
		for e := range ns {
			if strings.HasPrefix(h, e) {
				prefixExists = true
				break
			}
		}
		if !prefixExists {
			ns.Insert(h)
		}
	}
	return ns
}

func makeMatch(host, path string, headers map[string]v1alpha1.HeaderMatch) *istiov1alpha3.HTTPMatchRequest {
	match := &istiov1alpha3.HTTPMatchRequest{
		Authority: &istiov1alpha3.StringMatch{
			// Do not use Regex as Istio 1.4 or later has 100 bytes limitation.
			MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: host},
		},
	}
	// Empty path is considered match all path. We only need to consider path
	// when it's non-empty.
	if path != "" {
		match.Uri = &istiov1alpha3.StringMatch{
			MatchType: &istiov1alpha3.StringMatch_Prefix{Prefix: path},
		}
	}

	for k, v := range headers {
		match.Headers = map[string]*istiov1alpha3.StringMatch{
			k: {
				MatchType: &istiov1alpha3.StringMatch_Exact{
					Exact: v.Exact,
				},
			},
		}
	}

	return match
}

// hostPrefix returns an host to match either host or host:<any port>.
// For clusterLocalHost, it trims .svc.<local domain> from the host to match short host.
func hostPrefix(host string) string {
	localDomainSuffix := ".svc." + network.GetClusterDomainName()
	if !strings.HasSuffix(host, localDomainSuffix) {
		return host
	}
	return strings.TrimSuffix(host, localDomainSuffix)
}
