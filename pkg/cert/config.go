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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const (
	ConfigmapCertName         = "higress-https"
	ConfigmapCertConfigKey    = "cert"
	DefaultRenewalWindowRatio = 0.5
)

type Config struct {
	Email              string   `json:"email,omitempty"`
	Domains            []string `json:"domains,omitempty"`
	RenewalWindowRatio float64  `json:"renewalWindowRatio,omitempty"`
	AutomaticHttps     bool     `json:"automaticHttps,omitempty"`
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
			defaultConfig = &Config{
				Email:              strings.TrimSpace(email),
				RenewalWindowRatio: DefaultRenewalWindowRatio,
				AutomaticHttps:     true,
				Domains:            make([]string, 0),
			}
			err2 := c.ApplyConfigmap(defaultConfig)
			if err2 != nil {
				return nil, err2
			}
		}
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

	config := newDefaultConfig()
	if err := yaml.Unmarshal([]byte(configmap.Data[ConfigmapCertConfigKey]), config); err != nil {
		return nil, fmt.Errorf("data:%s,  convert to higress config error, error: %+v", configmap.Data[ConfigmapCertConfigKey], err)
	}

	if !ValidateEmail(config.Email) {
		return nil, fmt.Errorf("%s is not valid email address", config.Email)
	}

	if config.RenewalWindowRatio <= 0 || config.RenewalWindowRatio >= 1 {
		return nil, fmt.Errorf("RenewalWindowRatio should be between 0 and 1")
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

func NewConfigMgr(namespace string, client kubernetes.Interface) *ConfigMgr {
	configMgr := &ConfigMgr{
		client:    client,
		namespace: namespace,
	}
	return configMgr
}

func newDefaultConfig() *Config {
	config := &Config{
		Email:              "", // blank email address represents init status
		AutomaticHttps:     true,
		RenewalWindowRatio: DefaultRenewalWindowRatio,
		Domains:            make([]string, 0),
	}

	return config
}

func getRandEmail() string {
	num1 := rangeRandom(100, 100000)
	num2 := rangeRandom(100, 100000)
	return fmt.Sprintf("your%d@yours%d.com", num1, num2)
}
