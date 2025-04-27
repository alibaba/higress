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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
)

func TestIngressDomainCache(t *testing.T) {
	cache := NewIngressDomainCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.Valid)
	assert.Empty(t, cache.Invalid)

	cache.Valid["example.com"] = &IngressDomainBuilder{
		Host:      "example.com",
		Protocol:  HTTP,
		ClusterId: "cluster-1",
		Ingress: &config.Config{
			Meta: config.Meta{
				Name:      "test-ingress",
				Namespace: "default",
			},
		},
	}

	cache.Invalid = append(cache.Invalid, model.IngressDomain{
		Host:  "invalid.com",
		Error: "invalid domain",
	})

	result := cache.Extract()
	assert.Equal(t, 1, len(result.Valid))
	assert.Equal(t, "example.com", result.Valid[0].Host)
	assert.Equal(t, string(HTTP), result.Valid[0].Protocol)

	assert.Equal(t, 1, len(result.Invalid))
	assert.Equal(t, "invalid.com", result.Invalid[0].Host)
}

func TestIngressDomainBuilder(t *testing.T) {
	builder := &IngressDomainBuilder{
		Host:      "example.com",
		Protocol:  HTTP,
		ClusterId: "cluster-1",
		Ingress: &config.Config{
			Meta: config.Meta{
				Name:      "test-ingress",
				Namespace: "default",
			},
		},
	}

	domain := builder.Build()
	assert.Equal(t, "example.com", domain.Host)
	assert.Equal(t, string(HTTP), domain.Protocol)

	builder.Event = MissingSecret
	eventDomain := builder.Build()
	assert.Contains(t, eventDomain.Error, "misses secret")

	builder.Event = DuplicatedTls
	builder.PreIngress = &config.Config{
		Meta: config.Meta{
			Name:      "pre-ingress",
			Namespace: "default",
		},
	}
	builder.PreIngress.Meta.Annotations = map[string]string{
		ClusterIdAnnotation: "pre-cluster",
	}
	dupDomain := builder.Build()
	assert.Contains(t, dupDomain.Error, "conflicted with ingress")

	builder.Protocol = HTTPS
	builder.SecretName = "test-secret"
	builder.Event = ""
	httpsDomain := builder.Build()
	assert.Equal(t, string(HTTPS), httpsDomain.Protocol)
	assert.Equal(t, "test-secret", httpsDomain.SecretName)
}
