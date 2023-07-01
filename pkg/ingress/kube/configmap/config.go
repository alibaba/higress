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

package configmap

import (
	"errors"
	"reflect"
)

type Result int32

const (
	ResultNothing Result = iota
	ResultReplace
	ResultDelete

	HigressConfigMapName          = "higress-config"
	HigressConfigMapKey           = "higress"
	HigressTracingEnvoyFilterName = "higress-config-tracing-envoyfilter"

	defaultTimeout  = 500
	defaultSampling = 100.0
)

type HigressConfig struct {
	Tracing Tracing `json:"tracing,omitempty"`
}

func NewDefaultHigressConfig() *HigressConfig {
	higressConfig := &HigressConfig{
		Tracing: Tracing{
			Enable:   false,
			Timeout:  defaultTimeout,
			Sampling: defaultSampling,
		},
	}
	return higressConfig
}

type Tracing struct {
	// Flag to control trace
	Enable bool `json:"enable,omitempty"`
	// The percentage of requests (0.0 - 100.0) that will be randomly selected for trace generation,
	// if not requested by the client or not forced. Default is 100.0.
	Sampling float64 `json:"sampling,omitempty"`
	// The timeout for the gRPC request. Default is 500ms
	Timeout int32 `json:"timeout,omitempty"`
	// The tracer implementation to be used by Envoy.
	//
	// Types that are assignable to Tracer:
	Zipkin        *Zipkin        `json:"zipkin,omitempty"`
	Skywalking    *Skywalking    `json:"skywalking,omitempty"`
	OpenTelemetry *OpenTelemetry `json:"opentelemetry,omitempty"`
}

// Zipkin defines configuration for a Zipkin tracer.
type Zipkin struct {
	// Address of the Zipkin service (e.g. _zipkin:9411_).
	Service string `json:"service,omitempty"`
	Port    string `json:"port,omitempty"`
}

// Defines configuration for a Skywalking tracer.
type Skywalking struct {
	// Address of the Skywalking tracer.
	Service string `json:"service,omitempty"`
	Port    string `json:"port,omitempty"`
	// The access token
	AccessToken string `json:"access_token,omitempty"`
}

type OpenTelemetry struct {
	// Address of OpenTelemetry tracer.
	Service string `json:"service,omitempty"`
	Port    string `json:"port,omitempty"`
}

func validServiceAndPort(service string, port string) bool {
	if len(service) == 0 || len(port) == 0 {
		return false
	}
	return true
}

func ValidTracing(config *HigressConfig) error {
	t := config.Tracing

	if t.Timeout <= 0 {
		return errors.New("timeout can not be less than zero")
	}

	if t.Sampling < 0 || t.Sampling > 100 {
		return errors.New("sampling must be in (0.0 - 100.0)")
	}

	tracerNum := 0
	if t.Zipkin != nil {
		if validServiceAndPort(t.Zipkin.Service, t.Zipkin.Port) {
			tracerNum++
		} else {
			return errors.New("zipkin service and port can not be empty")
		}
	}

	if t.Skywalking != nil {
		if validServiceAndPort(t.Skywalking.Service, t.Skywalking.Port) {
			tracerNum++
		} else {
			return errors.New("skywalking service and port can not be empty")
		}
	}

	if t.OpenTelemetry != nil {
		if validServiceAndPort(t.OpenTelemetry.Service, t.OpenTelemetry.Port) {
			tracerNum++
		} else {
			return errors.New("opentelemetry service and port can not be empty")
		}
	}

	if tracerNum != 1 {
		return errors.New("only one of skywalking，zipkin and opentelemetry configuration can be set")
	}
	return nil
}

func CompareTracing(old *HigressConfig, new *HigressConfig) (Result, error) {
	if old == nil || new == nil {
		return ResultNothing, nil
	}

	if !reflect.DeepEqual(old, new) {
		return ResultReplace, nil
	}

	return ResultNothing, nil
}
