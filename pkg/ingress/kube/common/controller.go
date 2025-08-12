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
	"time"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
	gatewaytool "istio.io/istio/pkg/config/gateway"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/higress/pkg/cert"
	"github.com/alibaba/higress/pkg/common"
	"github.com/alibaba/higress/pkg/ingress/kube/annotations"
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
	ClusterId     cluster.ID
	Host          string
}

func CreateMcpServiceKey(host string, portNumber int32) ServiceKey {
	return ServiceKey{
		Namespace:   "mcp",
		Name:        host,
		ServiceFQDN: host,
		Port:        portNumber,
	}
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
	ClusterId        cluster.ID
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

type ServiceProxyConfig struct {
	ProxyName        string
	UpstreamProtocol common.Protocol
	UpstreamSni      string
}

type ServiceWrapper struct {
	ServiceName            string
	ServiceEntry           *networking.ServiceEntry
	DestinationRuleWrapper *WrapperDestinationRule
	Suffix                 string
	RegistryType           string
	RegistryName           string
	ProxyConfig            *ServiceProxyConfig
	createTime             time.Time
}

func (sew *ServiceWrapper) DeepCopy() *ServiceWrapper {
	res := &ServiceWrapper{}
	*res = *sew
	res.ServiceEntry = sew.ServiceEntry.DeepCopy()

	if sew.DestinationRuleWrapper != nil {
		res.DestinationRuleWrapper = sew.DestinationRuleWrapper
		res.DestinationRuleWrapper.DestinationRule = sew.DestinationRuleWrapper.DestinationRule.DeepCopy()
	}
	return res
}

func (sew *ServiceWrapper) SetCreateTime(createTime time.Time) {
	sew.createTime = createTime
}

func (sew *ServiceWrapper) GetCreateTime() time.Time {
	return sew.createTime
}

type ProxyWrapper struct {
	ProxyName    string
	ListenerPort uint32
	EnvoyFilter  *networking.EnvoyFilter
	createTime   time.Time
}

func (pw *ProxyWrapper) DeepCopy() *ProxyWrapper {
	res := &ProxyWrapper{}
	*res = *pw

	if pw.EnvoyFilter != nil {
		res.EnvoyFilter = pw.EnvoyFilter.DeepCopy()
	}
	return res
}

func (pw *ProxyWrapper) SetCreateTime(createTime time.Time) {
	pw.createTime = createTime
}

func (pw *ProxyWrapper) GetCreateTime() time.Time {
	return pw.createTime
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

type GatewayController interface {
	model.ConfigStoreController

	SetWatchErrorHandler(func(r *cache.Reflector, err error)) error
}
