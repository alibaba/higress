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
	"strings"

	"github.com/alibaba/higress/pkg/cert"
	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	gatewaytool "istio.io/istio/pkg/config/gateway"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ServiceKey struct {
	Namespace   string
	Name        string
	ServiceFQDN string
	Port        int32
}

type WrapperConfig struct {
	Config            *config.Config
	AnnotationsConfig *annotations.Ingress
}

type WrapperConfigWithRuleKey struct {
	Config  *config.Config
	RuleKey string
}

type WrapperGateway struct {
	Gateway       *networking.Gateway
	WrapperConfig *WrapperConfig
	ClusterId     string
	Host          string
}

func (w *WrapperGateway) IsHTTPS() bool {
	if w.Gateway == nil || len(w.Gateway.Servers) == 0 {
		return false
	}

	for _, server := range w.Gateway.Servers {
		if gatewaytool.IsTLSServer(server) {
			return true
		}
	}

	return false
}

type WrapperHTTPRoute struct {
	HTTPRoute        *networking.HTTPRoute
	WrapperConfig    *WrapperConfig
	RawClusterId     string
	ClusterId        string
	ClusterName      string
	Host             string
	OriginPath       string
	OriginPathType   PathType
	WeightTotal      int32
	IsDefaultBackend bool
	RuleKey          string
}

func (w *WrapperHTTPRoute) Meta() string {
	return strings.Join([]string{w.WrapperConfig.Config.Namespace, w.WrapperConfig.Config.Name}, "/")
}

func (w *WrapperHTTPRoute) BasePathFormat() string {
	return strings.Join([]string{w.Host, w.OriginPath}, "-")
}

func (w *WrapperHTTPRoute) PathFormat() string {
	return strings.Join([]string{w.Host, string(w.OriginPathType), w.OriginPath}, "-")
}

type WrapperVirtualService struct {
	VirtualService           *networking.VirtualService
	WrapperConfig            *WrapperConfig
	ConfiguredDefaultBackend bool
	AppRoot                  string
}

type WrapperTrafficPolicy struct {
	TrafficPolicy     *networking.TrafficPolicy
	PortTrafficPolicy *networking.TrafficPolicy_PortTrafficPolicy
	WrapperConfig     *WrapperConfig
}

type WrapperDestinationRule struct {
	DestinationRule *networking.DestinationRule
	WrapperConfig   *WrapperConfig
	ServiceKey      ServiceKey
}

type IngressController interface {
	// RegisterEventHandler adds a handler to receive config update events for a
	// configuration type
	RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler)

	List() []config.Config

	ServiceLister() listerv1.ServiceLister

	SecretLister() listerv1.SecretLister

	ConvertGateway(convertOptions *ConvertOptions, wrapper *WrapperConfig, httpsCredentialConfig *cert.Config) error

	ConvertHTTPRoute(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	ApplyDefaultBackend(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	ApplyCanaryIngress(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	ConvertTrafficPolicy(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	// Run until a signal is received
	Run(stop <-chan struct{})

	SetWatchErrorHandler(func(r *cache.Reflector, err error)) error

	// HasSynced returns true after initial cache synchronization is complete
	HasSynced() bool
}

type KIngressController interface {
	// RegisterEventHandler adds a handler to receive config update events for a
	// configuration type
	RegisterEventHandler(kind config.GroupVersionKind, handler model.EventHandler)

	List() []config.Config

	ServiceLister() listerv1.ServiceLister

	SecretLister() listerv1.SecretLister

	ConvertGateway(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	ConvertHTTPRoute(convertOptions *ConvertOptions, wrapper *WrapperConfig) error

	// Run until a signal is received
	Run(stop <-chan struct{})

	SetWatchErrorHandler(func(r *cache.Reflector, err error)) error

	// HasSynced returns true after initial cache synchronization is complete
	HasSynced() bool
}
