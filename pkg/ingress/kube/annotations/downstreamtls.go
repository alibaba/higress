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
	"strings"

	networking "istio.io/api/networking/v1alpha3"
	gatewaytool "istio.io/istio/pkg/config/gateway"
	"istio.io/istio/pkg/config/security"
	"k8s.io/apimachinery/pkg/types"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
)

const (
	authTLSSecret      = "auth-tls-secret"
	sslCipher          = "ssl-cipher"
	gatewaySdsCaSuffix = "-cacert"
	annotationMinTLSVersion = "tls-min-version" 
	annotationMaxTLSVersion = "tls-max-version"
	RuleMinVersion map[string]string
	RuleMaxVersion map[string]string
)

var (
	_ Parser         = &downstreamTLS{}
	_ GatewayHandler = &downstreamTLS{}
)

type DownstreamTLSConfig struct {
	CipherSuites []string
	Mode         networking.ServerTLSSettings_TLSmode
	CASecretName types.NamespacedName
	MinVersion string
	MaxVersion string
}

type downstreamTLS struct{}

func (d downstreamTLS) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needDownstreamTLS(annotations) {
		return nil
	}

	downstreamTLSConfig := &DownstreamTLSConfig{
		Mode: networking.ServerTLSSettings_SIMPLE,
		RuleMinVersion: make(map[string]string),
		RuleMaxVersion: make(map[string]string),
	}
	defer func() {
		config.DownstreamTLS = downstreamTLSConfig
	}()

	if secretName, err := annotations.ParseStringASAP(authTLSSecret); err == nil {
		namespacedName := util.SplitNamespacedName(secretName)
		if namespacedName.Name == "" {
			IngressLog.Errorf("CA secret name %s format is invalid.", secretName)
		} else {
			if namespacedName.Namespace == "" {
				namespacedName.Namespace = config.Namespace
			}
			downstreamTLSConfig.CASecretName = namespacedName
			downstreamTLSConfig.Mode = networking.ServerTLSSettings_MUTUAL
		}
	}

	if rawTlsCipherSuite, err := annotations.ParseStringASAP(sslCipher); err == nil {
		var validCipherSuite []string
		cipherList := strings.Split(rawTlsCipherSuite, ":")
		for _, cipher := range cipherList {
			if security.IsValidCipherSuite(cipher) {
				validCipherSuite = append(validCipherSuite, cipher)
			}
		}

		downstreamTLSConfig.CipherSuites = validCipherSuite
	}
	
	if minVersion, err := annotations.ParseStringASAP(annotationMinTLSVersion); err == nil {
		downstreamTLSConfig.MinVersion = minVersion
	}

	if maxVersion, err := annotations.ParseStringASAP(annotationMaxTLSVersion); err == nil {
		downstreamTLSConfig.MaxVersion = maxVersion
	}
	
	for key, value := range annotations {
			if strings.HasPrefix(key, annotationMinTLSVersion+".") {
					ruleName := strings.TrimPrefix(key, annotationMinTLSVersion+".")
					downstreamTLSConfig.RuleMinVersion[ruleName] = value
			}
			if strings.HasPrefix(key, annotationMaxTLSVersion+".") {
					ruleName := strings.TrimPrefix(key, annotationMaxTLSVersion+".")
					downstreamTLSConfig.RuleMaxVersion[ruleName] = value
			}
	}


	return nil
}

func (d downstreamTLS) ApplyGateway(gateway *networking.Gateway, config *Ingress) {
	if config.DownstreamTLS == nil {
		return
	}

	downstreamTLSConfig := config.DownstreamTLS
	for _, server := range gateway.Servers {
		if gatewaytool.IsTLSServer(server) {
			if downstreamTLSConfig.CASecretName.Name != "" {
				serverCert := extraSecret(server.Tls.CredentialName)
				if downstreamTLSConfig.CASecretName.Namespace != serverCert.Namespace ||
					(downstreamTLSConfig.CASecretName.Name != serverCert.Name &&
						downstreamTLSConfig.CASecretName.Name != serverCert.Name+gatewaySdsCaSuffix) {
					IngressLog.Errorf("CA secret %s is invalid", downstreamTLSConfig.CASecretName.String())
				} else {
					server.Tls.Mode = downstreamTLSConfig.Mode
				}
			}

			if len(downstreamTLSConfig.CipherSuites) != 0 {
				server.Tls.CipherSuites = downstreamTLSConfig.CipherSuites
			}

			ruleName := getRuleName(server)

			// 优先使用规则级别的TLS版本设置
			if minVersion, exists := downstreamTLSConfig.RuleMinVersion[ruleName]; exists {
					server.Tls.MinProtocolVersion = convertTLSVersion(minVersion)
			} else if downstreamTLSConfig.MinVersion != "" {
					// 回退到全局设置
					server.Tls.MinProtocolVersion = convertTLSVersion(downstreamTLSConfig.MinVersion)
			}

			if maxVersion, exists := downstreamTLSConfig.RuleMaxVersion[ruleName]; exists {
					server.Tls.MaxProtocolVersion = convertTLSVersion(maxVersion)
			} else if downstreamTLSConfig.MaxVersion != "" {
					// 回退到全局设置
					server.Tls.MaxProtocolVersion = convertTLSVersion(downstreamTLSConfig.MaxVersion)
			}
		}
	}
	
}

func needDownstreamTLS(annotations Annotations) bool {
	return annotations.HasASAP(sslCipher) ||
		annotations.HasASAP(authTLSSecret)||
		annotations.HasASAP(annotationMinTLSVersion) ||
		annotations.HasASAP(annotationMaxTLSVersion)
}

func convertTLSVersion(version string) networking.ServerTLSSettings_TLSProtocol {
		switch version {
		case "TLSv1_0":
				return networking.ServerTLSSettings_TLSV1_0
		case "TLSv1_1":
				return networking.ServerTLSSettings_TLSV1_1
		case "TLSv1_2":
				return networking.ServerTLSSettings_TLSV1_2
		case "TLSv1_3":
				return networking.ServerTLSSettings_TLSV1_3
		default:
				return networking.ServerTLSSettings_TLS_AUTO
		}
}

func getRuleName(server *networking.Server) string {
		// 从server配置中提取规则名称
		// 可以使用server.Name或其他标识
		return server.Name
}