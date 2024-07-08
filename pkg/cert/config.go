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

package cert

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"istio.io/istio/pkg/config/host"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	ConfigmapCertName      = "higress-https"
	ConfigmapCertConfigKey = "cert"
	DefaultRenewBeforeDays = 30
	RenewMaxDays           = 90
)

type IssuerName string

const (
	IssuerTypeAliyunSSL   IssuerName = "aliyunssl"
	IssuerTypeLetsencrypt IssuerName = "letsencrypt"
)

// Config is the configuration of automatic https.
type Config struct {
	AutomaticHttps           bool              `json:"automaticHttps"`
	FallbackForInvalidSecret bool              `json:"fallbackForInvalidSecret"`
	RenewBeforeDays          int               `json:"renewBeforeDays"`
	CredentialConfig         []CredentialEntry `json:"credentialConfig"`
	ACMEIssuer               []ACMEIssuerEntry `json:"acmeIssuer"`
	Version                  string            `json:"version"`
}

func (c *Config) GetIssuer(issuerName IssuerName) *ACMEIssuerEntry {
	for _, issuer := range c.ACMEIssuer {
		if issuer.Name == issuerName {
			return &issuer
		}
	}
	return nil
}

func (c *Config) MatchSecretNameByDomain(domain string) string {
	for _, credential := range c.CredentialConfig {
		for _, credDomain := range credential.Domains {
			if host.Name(strings.ToLower(domain)).SubsetOf(host.Name(strings.ToLower(credDomain))) {
				return credential.TLSSecret
			}
		}
	}
	return ""
}

func (c *Config) GetSecretNameByDomain(issuerName IssuerName, domain string) string {
	for _, credential := range c.CredentialConfig {
		if credential.TLSIssuer == issuerName {
			for _, credDomain := range credential.Domains {
				if host.Name(strings.ToLower(domain)).SubsetOf(host.Name(strings.ToLower(credDomain))) {
					return credential.TLSSecret
				}
			}
		}
	}
	return ""
}

func ParseTLSSecret(tlsSecret string) (string, string) {
	secrets := strings.Split(tlsSecret, "/")
	switch len(secrets) {
	case 1:
		return "", tlsSecret
	case 2:
		return secrets[0], secrets[1]
	}
	return "", ""
}

func (c *Config) Validate() error {
	// check acmeIssuer
	if len(c.ACMEIssuer) == 0 {
		return fmt.Errorf("acmeIssuer is empty")
	}
	for _, issuer := range c.ACMEIssuer {
		switch issuer.Name {
		case IssuerTypeLetsencrypt:
			if issuer.Email == "" {
				return fmt.Errorf("acmeIssuer %s email is empty", issuer.Name)
			}
			if !ValidateEmail(issuer.Email) {
				return fmt.Errorf("acmeIssuer %s email %s is invalid", issuer.Name, issuer.Email)
			}
		default:
			return fmt.Errorf("acmeIssuer name %s is not supported", issuer.Name)
		}
	}
	// check credentialConfig
	for _, credential := range c.CredentialConfig {
		if len(credential.Domains) == 0 {
			return fmt.Errorf("credentialConfig domains is empty")
		}
		if credential.TLSSecret == "" {
			return fmt.Errorf("credentialConfig tlsSecret is empty")
		} else {
			ns, secret := ParseTLSSecret(credential.TLSSecret)
			if ns == "" && secret == "" {
				return fmt.Errorf("credentialConfig tlsSecret %s is not supported", credential.TLSSecret)
			}
		}

		if credential.TLSIssuer == IssuerTypeLetsencrypt {
			if len(credential.Domains) > 1 {
				return fmt.Errorf("credentialConfig tlsIssuer %s only support one domain", credential.TLSIssuer)
			}
		}
		if credential.TLSIssuer != IssuerTypeLetsencrypt && len(credential.TLSIssuer) > 0 {
			return fmt.Errorf("credential tls issuer %s is not supported", credential.TLSIssuer)
		}
	}

	if c.RenewBeforeDays <= 0 {
		return fmt.Errorf("RenewBeforeDays should be large than zero")
	}

	if c.RenewBeforeDays >= RenewMaxDays {
		return fmt.Errorf("RenewBeforeDays should be less than %d", RenewMaxDays)
	}
	return nil
}

type CredentialEntry struct {
	Domains      []string   `json:"domains"`
	TLSIssuer    IssuerName `json:"tlsIssuer,omitempty"`
	TLSSecret    string     `json:"tlsSecret,omitempty"`
	CACertSecret string     `json:"cacertSecret,omitempty"`
}

type ACMEIssuerEntry struct {
	Name  IssuerName `json:"name"`
	Email string     `json:"email"`
	AK    string     `json:"ak"` // Only applicable for certain issuers like 'aliyunssl'
	SK    string     `json:"sk"` // Only applicable for certain issuers like 'aliyunssl'
}
type ConfigMgr struct {
	client    kubernetes.Interface
	config    atomic.Value
	namespace string
}

func (c *ConfigMgr) SetConfig(config *Config) {
	c.config.Store(config)
}

func (c *ConfigMgr) GetConfig() *Config {
	value := c.config.Load()
	if value != nil {
		if config, ok := value.(*Config); ok {
			return config
		}
	}
	return nil
}

func (c *ConfigMgr) InitConfig(email string) (*Config, error) {
	var defaultConfig *Config
	cm, err := c.GetConfigmap()
	if err != nil {
		if errors.IsNotFound(err) {
			if len(strings.TrimSpace(email)) == 0 {
				email = getRandEmail()
			}
			defaultConfig = newDefaultConfig(email)
			err2 := c.ApplyConfigmap(defaultConfig)
			if err2 != nil {
				return nil, err2
			}
		}
		return nil, err
	} else {
		defaultConfig, err = c.ParseConfigFromConfigmap(cm)
		if err != nil {
			return nil, err
		}
	}
	return defaultConfig, nil
}

func (c *ConfigMgr) ParseConfigFromConfigmap(configmap *v1.ConfigMap) (*Config, error) {
	if _, ok := configmap.Data[ConfigmapCertConfigKey]; !ok {
		return nil, fmt.Errorf("no cert key %s in configmap %s", ConfigmapCertConfigKey, configmap.Name)
	}

	config := newDefaultConfig("")
	if err := yaml.Unmarshal([]byte(configmap.Data[ConfigmapCertConfigKey]), config); err != nil {
		return nil, fmt.Errorf("data:%s,  convert to higress config error, error: %+v", configmap.Data[ConfigmapCertConfigKey], err)
	}
	// validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *ConfigMgr) GetConfigFromConfigmap() (*Config, error) {
	var config *Config
	cm, err := c.GetConfigmap()
	if err != nil {
		return nil, err
	} else {
		config, err = c.ParseConfigFromConfigmap(cm)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func (c *ConfigMgr) GetConfigmap() (configmap *v1.ConfigMap, err error) {
	configmapName := ConfigmapCertName
	cm, err := c.client.CoreV1().ConfigMaps(c.namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
	return cm, err
}

func (c *ConfigMgr) ApplyConfigmap(config *Config) error {
	configmapName := ConfigmapCertName
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.namespace,
			Name:      configmapName,
		},
	}
	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	cm.Data = make(map[string]string, 0)
	cm.Data[ConfigmapCertConfigKey] = string(bytes)

	_, err = c.client.CoreV1().ConfigMaps(c.namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err = c.client.CoreV1().ConfigMaps(c.namespace).Create(context.Background(), cm, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if _, err = c.client.CoreV1().ConfigMaps(c.namespace).Update(context.Background(), cm, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func NewConfigMgr(namespace string, client kubernetes.Interface) (*ConfigMgr, error) {
	configMgr := &ConfigMgr{
		client:    client,
		namespace: namespace,
	}
	return configMgr, nil
}

func newDefaultConfig(email string) *Config {

	defaultIssuer := []ACMEIssuerEntry{
		{
			Name:  IssuerTypeLetsencrypt,
			Email: email,
		},
	}
	defaultCredentialConfig := make([]CredentialEntry, 0)
	config := &Config{
		AutomaticHttps:           true,
		FallbackForInvalidSecret: false,
		RenewBeforeDays:          DefaultRenewBeforeDays,
		ACMEIssuer:               defaultIssuer,
		CredentialConfig:         defaultCredentialConfig,
		Version:                  time.Now().Format("20060102030405"),
	}
	return config
}

func getRandEmail() string {
	num1 := rangeRandom(100, 100000)
	num2 := rangeRandom(100, 100000)
	return fmt.Sprintf("your%d@yours%d.com", num1, num2)
}
