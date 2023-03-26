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
	. "github.com/alibaba/higress/pkg/ingress/log"
)

const (
	authzTypeAnn                          = "authz-type"
	protoAnn                              = "ext-authz-proto"
	serviceAnn                            = "ext-authz-service"
	servicePortAnn                        = "ext-authz-service-port"
	servicePathPrefixAnn                  = "ext-authz-http-service-path-prefix"
	reqAllowedHeadersExactAnn             = "ext-authz-req-allowed-headers-exact"
	reqAllowedHeadersPrefixAnn            = "ext-authz-req-allowed-headers-prefix"
	reqAllowedHeadersSuffixAnn            = "ext-authz-req-allowed-headers-suffix"
	reqAllowedHeadersContainsAnn          = "ext-authz-req-allowed-headers-contains"
	respAllowedUpstreamHeadersExactAnn    = "ext-authz-req-allowed-upstream-headers-exact"
	respAllowedUpstreamHeadersPrefixAnn   = "ext-authz-req-allowed-upstream-headers-prefix"
	respAllowedUpstreamHeadersSuffixAnn   = "ext-authz-req-allowed-upstream-headers-suffix"
	respAllowedUpstreamHeadersContainsAnn = "ext-authz-req-allowed-upstream-headers-contains"
	respAllowedClientHeadersExactAnn      = "ext-authz-req-allowed-client-headers-exact"
	respAllowedClientHeadersPrefixAnn     = "ext-authz-req-allowed-client-headers-prefix"
	respAllowedClientHeadersSuffixAnn     = "ext-authz-req-allowed-client-headers-suffix"
	respAllowedClientHeadersContainsAnn   = "ext-authz-req-allowed-client-headers-contains"
	rbacPolicyIdAnn                       = "ext-authz-rbac-policy-id"
	reqMaxBytesAnn                        = "ext-authz-req-max-bytes"
	reqAllowPartialAnn                    = "ext-authz-req-allow-partial"
	packAsBytesAnn                        = "ext-authz-pack-as-bytes"
	serviceTimeOutAnn                     = "ext-authz-timeout"

	defaultAuthzType = "ext-authz"
)

type extAuthzProto string

const (
	GRPC extAuthzProto = "grpc"
	HTTP extAuthzProto = "http"
)

var _ Parser = authz{}

type AuthzConfig struct {
	AuthzType string
	ExtAuthz  *ExtAuthzConfig
}

type ExtAuthzConfig struct {
	AuthzProto      extAuthzProto
	AuthzService    *ServiceConfig
	RbacPolicyId    string
	ReqMaxBytes     uint32
	ReqAllowPartial bool
	PackAsBytes     bool
}

type ServiceConfig struct {
	Timeout                            string
	ServiceName                        string
	ServicePort                        int
	ServicePathPrefix                  string
	ReqAllowedHeadersExact             []string
	ReqAllowedHeadersPrefix            []string
	ReqAllowedHeadersSuffix            []string
	ReqAllowedHeadersContains          []string
	RespAllowedUpstreamHeadersExact    []string
	RespAllowedUpstreamHeadersPrefix   []string
	RespAllowedUpstreamHeadersSuffix   []string
	RespAllowedUpstreamHeadersContains []string
	RespAllowedClientHeadersExact      []string
	RespAllowedClientHeadersPrefix     []string
	RespAllowedClientHeadersSuffix     []string
	RespAllowedClientHeadersContains   []string
}

type authz struct{}

func (a authz) Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error {
	IngressLog.Infof("Parse authz annotations")
	if !needAuthzConfig(annotations) {
		return nil
	}

	authzConfig := &AuthzConfig{
		AuthzType: defaultAuthzType,
	}

	authzType, err := annotations.ParseStringForHigress(authzTypeAnn)
	if err != nil {
		IngressLog.Errorf("Parse authz type error %v within ingress %/%s", err, config.Namespace, config.Name)
		return nil
	}
	if authzType != defaultAuthzType {
		IngressLog.Errorf("Auth type %s within ingress %/%s is not supported yet.", authzType, config.Namespace, config.Name)
		return nil
	}
	proto := GRPC
	if rawProto, err := annotations.ParseStringForHigress(protoAnn); err == nil {
		resultProto := extAuthzProto(rawProto)
		if resultProto == GRPC || resultProto == HTTP {
			proto = resultProto
		}
	}
	extAuthzConfig := &ExtAuthzConfig{
		AuthzProto: proto,
	}

	serviceConfig := &ServiceConfig{}
	if timeout, err := annotations.ParseStringForHigress(serviceTimeOutAnn); err == nil {
		serviceConfig.Timeout = timeout
	}
	if service, err := annotations.ParseStringForHigress(serviceAnn); err == nil && service != "" {
		serviceConfig.ServiceName = service
	} else {
		IngressLog.Errorf("Authz service name within ingress %s/%s is not configure", config.Namespace, config.Name)
		return nil
	}
	if servicePort, err := annotations.ParseIntForHigress(servicePortAnn); err == nil {
		if servicePort <= 0 || servicePort > 65535 {
			IngressLog.Errorf("Authz service port within ingress %s/%s is invalid", config.Namespace, config.Name)
			return nil
		}
		serviceConfig.ServicePort = servicePort
	} else {
		serviceConfig.ServicePort = 80
	}
	if servicePathPrefix, err := annotations.ParseStringForHigress(servicePathPrefixAnn); err == nil {
		serviceConfig.ServicePathPrefix = servicePathPrefix
	}
	if reqAllowedHeadersExact, err := annotations.ParseStringForHigress(reqAllowedHeadersExactAnn); err == nil {
		serviceConfig.ReqAllowedHeadersExact = splitStringWithSpaceTrim(reqAllowedHeadersExact)
	}
	if reqAllowedHeadersPrefix, err := annotations.ParseStringForHigress(reqAllowedHeadersPrefixAnn); err == nil {
		serviceConfig.ReqAllowedHeadersPrefix = splitStringWithSpaceTrim(reqAllowedHeadersPrefix)
	}
	if reqAllowedHeadersSuffix, err := annotations.ParseStringForHigress(reqAllowedHeadersSuffixAnn); err == nil {
		serviceConfig.ReqAllowedHeadersSuffix = splitStringWithSpaceTrim(reqAllowedHeadersSuffix)
	}
	if reqAllowedHeadersContains, err := annotations.ParseStringForHigress(reqAllowedHeadersContainsAnn); err == nil {
		serviceConfig.ReqAllowedHeadersContains = splitStringWithSpaceTrim(reqAllowedHeadersContains)
	}
	if respAllowedUpstreamHeadersExact, err := annotations.ParseStringForHigress(respAllowedUpstreamHeadersExactAnn); err == nil {
		serviceConfig.RespAllowedUpstreamHeadersExact = splitStringWithSpaceTrim(respAllowedUpstreamHeadersExact)
	}
	if respAllowedUpstreamHeadersPrefix, err := annotations.ParseStringForHigress(respAllowedUpstreamHeadersPrefixAnn); err == nil {
		serviceConfig.RespAllowedUpstreamHeadersPrefix = splitStringWithSpaceTrim(respAllowedUpstreamHeadersPrefix)
	}
	if respAllowedUpstreamHeadersSuffix, err := annotations.ParseStringForHigress(respAllowedUpstreamHeadersSuffixAnn); err == nil {
		serviceConfig.RespAllowedUpstreamHeadersSuffix = splitStringWithSpaceTrim(respAllowedUpstreamHeadersSuffix)
	}
	if respAllowedUpstreamHeadersContains, err := annotations.ParseStringForHigress(respAllowedUpstreamHeadersContainsAnn); err == nil {
		serviceConfig.RespAllowedUpstreamHeadersContains = splitStringWithSpaceTrim(respAllowedUpstreamHeadersContains)
	}
	if respAllowedClientHeadersExact, err := annotations.ParseStringForHigress(respAllowedClientHeadersExactAnn); err == nil {
		serviceConfig.RespAllowedClientHeadersExact = splitStringWithSpaceTrim(respAllowedClientHeadersExact)
	}
	if respAllowedClientHeadersPrefix, err := annotations.ParseStringForHigress(respAllowedClientHeadersPrefixAnn); err == nil {
		serviceConfig.RespAllowedClientHeadersPrefix = splitStringWithSpaceTrim(respAllowedClientHeadersPrefix)
	}
	if respAllowedClientHeadersSuffix, err := annotations.ParseStringForHigress(respAllowedClientHeadersSuffixAnn); err == nil {
		serviceConfig.RespAllowedClientHeadersSuffix = splitStringWithSpaceTrim(respAllowedClientHeadersSuffix)
	}
	if respAllowedClientHeadersContains, err := annotations.ParseStringForHigress(respAllowedClientHeadersContainsAnn); err == nil {
		serviceConfig.RespAllowedClientHeadersContains = splitStringWithSpaceTrim(respAllowedClientHeadersContains)
	}
	if rbacPolicyId, err := annotations.ParseStringForHigress(rbacPolicyIdAnn); err == nil {
		extAuthzConfig.RbacPolicyId = rbacPolicyId
	} else {
		rbacPolicyId = config.Namespace + "-" + config.Name + "-ext-authz-policy"
		extAuthzConfig.RbacPolicyId = rbacPolicyId
	}
	if reqMaxBytes, err := annotations.ParseUint32ForHigress(reqMaxBytesAnn); err == nil {

		extAuthzConfig.ReqMaxBytes = reqMaxBytes
	}
	if reqAllowPartial, err := annotations.ParseBoolForHigress(reqAllowPartialAnn); err == nil {
		extAuthzConfig.ReqAllowPartial = reqAllowPartial
	}
	if packAsBytes, err := annotations.ParseBoolForHigress(packAsBytesAnn); err == nil {
		extAuthzConfig.PackAsBytes = packAsBytes
	}
	extAuthzConfig.AuthzService = serviceConfig
	authzConfig.ExtAuthz = extAuthzConfig
	config.Authz = authzConfig
	return nil
}

func needAuthzConfig(annotations Annotations) bool {
	return annotations.HasASAP(authzTypeAnn) &&
		annotations.HasASAP(serviceAnn)
}
