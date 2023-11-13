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

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"

	networking "istio.io/api/networking/v1alpha3"
)

func TestRetryParse(t *testing.T) {
	retry := retry{}
	inputCases := []struct {
		input  map[string]string
		expect *RetryConfig
	}{
		{},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryCount): "1",
			},
			expect: &RetryConfig{
				retryCount:      1,
				retryOn:         "5xx",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(perRetryTimeout): "10",
			},
			expect: &RetryConfig{
				retryCount: 3,
				perRetryTimeout: &types.Duration{
					Seconds: 10,
				},
				retryOn: "5xx",
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryCount): "2",
				buildNginxAnnotationKey(retryOn):    "off",
			},
			expect: &RetryConfig{
				retryCount:      0,
				retryOn:         "5xx",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryCount): "2",
				buildNginxAnnotationKey(retryOn):    "error,timeout",
			},
			expect: &RetryConfig{
				retryCount:      2,
				retryOn:         "5xx",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryCount): "2",
				buildNginxAnnotationKey(retryOn):    "error  timeout",
			},
			expect: &RetryConfig{
				retryCount:      2,
				retryOn:         "5xx",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryOn): "timeout,non_idempotent",
			},
			expect: &RetryConfig{
				retryCount:      3,
				retryOn:         "5xx,non_idempotent",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryOn): "timeout non_idempotent",
			},
			expect: &RetryConfig{
				retryCount:      3,
				retryOn:         "5xx,non_idempotent",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryOn): "timeout,http_503,http_502,http_404",
			},
			expect: &RetryConfig{
				retryCount:      3,
				retryOn:         "5xx,retriable-status-codes,503,502,404",
				perRetryTimeout: &types.Duration{},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(retryOn): "timeout http_503  http_502 http_404",
			},
			expect: &RetryConfig{
				retryCount:      3,
				retryOn:         "5xx,retriable-status-codes,503,502,404",
				perRetryTimeout: &types.Duration{},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{}
			_ = retry.Parse(inputCase.input, config, nil)
			assert.Equal(t, inputCase.expect, config.Retry)
		})
	}
}

func TestRetryApplyRoute(t *testing.T) {
	retry := retry{}
	inputCases := []struct {
		config *Ingress
		input  *networking.HTTPRoute
		expect *networking.HTTPRoute
	}{
		{
			config: &Ingress{},
			input:  &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{},
		},
		{
			config: &Ingress{
				Retry: &RetryConfig{
					retryCount: 3,
					retryOn:    "test",
				},
			},
			input: &networking.HTTPRoute{},
			expect: &networking.HTTPRoute{
				Retries: &networking.HTTPRetry{
					Attempts: 3,
					RetryOn:  "test",
				},
			},
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			retry.ApplyRoute(inputCase.input, inputCase.config)
			if !reflect.DeepEqual(inputCase.input, inputCase.expect) {
				t.Fatalf("Should be equal")
			}
		})
	}
}
