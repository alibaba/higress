// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package annotations

import (
	"reflect"
	"testing"

	networking "istio.io/api/networking/v1alpha3"
)

func TestApplyWeight(t *testing.T) {
	route := &networking.HTTPRoute{
		Headers: &networking.Headers{
			Request: &networking.Headers_HeaderOperations{
				Add: map[string]string{
					"normal": "true",
				},
			},
		},
		Route: []*networking.HTTPRouteDestination{
			{
				Destination: &networking.Destination{
					Host: "normal",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
			},
		},
	}

	canary1 := &networking.HTTPRoute{
		Headers: &networking.Headers{
			Request: &networking.Headers_HeaderOperations{
				Add: map[string]string{
					"canary1": "true",
				},
			},
		},
		Route: []*networking.HTTPRouteDestination{
			{
				Destination: &networking.Destination{
					Host: "canary1",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
			},
		},
	}

	canary2 := &networking.HTTPRoute{
		Headers: &networking.Headers{
			Request: &networking.Headers_HeaderOperations{
				Add: map[string]string{
					"canary2": "true",
				},
			},
		},
		Route: []*networking.HTTPRouteDestination{
			{
				Destination: &networking.Destination{
					Host: "canary2",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
			},
		},
	}

	ApplyByWeight(canary1, route, &Ingress{
		Canary: &CanaryConfig{
			Weight: 30,
		},
	})

	ApplyByWeight(canary2, route, &Ingress{
		Canary: &CanaryConfig{
			Weight: 20,
		},
	})

	expect := &networking.HTTPRoute{
		Route: []*networking.HTTPRouteDestination{
			{
				Destination: &networking.Destination{
					Host: "normal",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"normal": "true",
						},
					},
				},
			},
			{
				Destination: &networking.Destination{
					Host: "canary1",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"canary1": "true",
						},
					},
				},
				Weight: 30,
				FallbackClusters: []*networking.Destination{
					{
						Host: "normal",
						Port: &networking.PortSelector{
							Number: 80,
						},
					},
				},
			},
			{
				Destination: &networking.Destination{
					Host: "canary2",
					Port: &networking.PortSelector{
						Number: 80,
					},
				},
				Headers: &networking.Headers{
					Request: &networking.Headers_HeaderOperations{
						Add: map[string]string{
							"canary2": "true",
						},
					},
				},
				Weight: 20,
				FallbackClusters: []*networking.Destination{
					{
						Host: "normal",
						Port: &networking.PortSelector{
							Number: 80,
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(route, expect) {
		t.Fatal("Should be equal")
	}
}
