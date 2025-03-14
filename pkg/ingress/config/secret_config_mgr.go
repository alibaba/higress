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

package config

import (
	"fmt"
	"sync"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/kind"
	"istio.io/istio/pkg/util/sets"
)

// toConfigKey converts config.Config to istiomodel.ConfigKey
func toConfigKey(cfg *config.Config) (istiomodel.ConfigKey, error) {
	return istiomodel.ConfigKey{
		Kind:      kind.MustFromGVK(cfg.GroupVersionKind),
		Name:      cfg.Name,
		Namespace: cfg.Namespace,
	}, nil
}

// SecretConfigMgr maintains the mapping between secrets and configs
type SecretConfigMgr struct {
	mutex sync.RWMutex

	// configSet tracks all configs that have been added
	// key format: namespace/name
	configSet sets.Set[string]

	// secretToConfigs maps secret key to dependent configs
	// key format: namespace/name
	secretToConfigs map[string]sets.Set[istiomodel.ConfigKey]

	// watchedSecrets tracks which secrets are being watched
	watchedSecrets sets.Set[string]

	// xdsUpdater is used to push config updates
	xdsUpdater istiomodel.XDSUpdater
}

// NewSecretConfigMgr creates a new SecretConfigMgr
func NewSecretConfigMgr(xdsUpdater istiomodel.XDSUpdater) *SecretConfigMgr {
	return &SecretConfigMgr{
		secretToConfigs: make(map[string]sets.Set[istiomodel.ConfigKey]),
		watchedSecrets:  sets.New[string](),
		configSet:       sets.New[string](),
		xdsUpdater:      xdsUpdater,
	}
}

// AddConfig adds a config and its secret dependencies
func (m *SecretConfigMgr) AddConfig(secretKey string, cfg *config.Config) error {
	IngressLog.Infof("SecretConfigMgr Adding config %s/%s for secret key:%s", cfg.Namespace, cfg.Name, secretKey)
	configKey, _ := toConfigKey(cfg)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	configId := fmt.Sprintf("%s/%s", cfg.Namespace, cfg.Name)
	m.configSet.Insert(configId)

	if configs, exists := m.secretToConfigs[secretKey]; exists {
		configs.Insert(configKey)
	} else {
		m.secretToConfigs[secretKey] = sets.New(configKey)
	}

	// Add to watched secrets
	m.watchedSecrets.Insert(secretKey)
	IngressLog.Infof("watchedSecrets %v", m.watchedSecrets)
	return nil
}

// DeleteConfig removes a config from all secret dependencies
func (m *SecretConfigMgr) DeleteConfig(cfg *config.Config) error {
	IngressLog.Infof("SecretConfigMgr Deleting config %s/%s", cfg.Namespace, cfg.Name)
	configKey, _ := toConfigKey(cfg)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	configId := fmt.Sprintf("%s/%s", cfg.Namespace, cfg.Name)
	if !m.configSet.Contains(configId) {
		return nil
	}

	removeKeys := make([]string, 0)
	// Find and remove the config from all secrets
	for secretKey, configs := range m.secretToConfigs {
		if configs.Contains(configKey) {
			configs.Delete(configKey)
			// If no more configs depend on this secret, remove it
			if configs.Len() == 0 {
				removeKeys = append(removeKeys, secretKey)
			}
		}
	}

	//  Remove the secrets from the secretToConfigs map
	for _, secretKey := range removeKeys {
		IngressLog.Infof("Removing secret %s from secretToConfigs", secretKey)
		delete(m.secretToConfigs, secretKey)
		m.watchedSecrets.Delete(secretKey)
	}
	// Remove the config from the config set
	m.configSet.Delete(configId)
	IngressLog.Infof("watchedSecrets %v", m.watchedSecrets)

	return nil
}

// GetConfigsForSecret returns all configs that depend on the given secret
func (m *SecretConfigMgr) GetConfigsForSecret(secretKey string) []istiomodel.ConfigKey {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if configs, exists := m.secretToConfigs[secretKey]; exists {
		return configs.UnsortedList()
	}
	return nil
}

// IsSecretWatched checks if a secret is being watched
func (m *SecretConfigMgr) IsSecretWatched(secretKey string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.watchedSecrets.Contains(secretKey)
}

// HandleSecretChange handles secret changes and updates affected configs
func (m *SecretConfigMgr) HandleSecretChange(name util.ClusterNamespacedName) {
	secretKey := fmt.Sprintf("%s/%s", name.Namespace, name.Name)
	IngressLog.Infof("SecretConfigMgr Handling Secret %s changed", secretKey)
	// Check if this secret is being watched
	if !m.IsSecretWatched(secretKey) {
		IngressLog.Infof("Secret %s is not being watched", secretKey)
		return
	}

	// Get affected configs
	configKeys := m.GetConfigsForSecret(secretKey)
	if len(configKeys) == 0 {
		IngressLog.Infof("No configs depend on secret %s", secretKey)
		return
	}

	// Create ConfigsUpdated set for push request
	configsUpdated := sets.New[istiomodel.ConfigKey]()

	shouldFullPush := false
	// Update each affected config
	for _, configKey := range configKeys {
		// Create a new config for update
		if configKey.Kind == kind.Secret && configKey.Name == name.Name && configKey.Namespace == name.Namespace {
			// Secret itself is being updated, need full push
			shouldFullPush = true
			break
		}
		configsUpdated.Insert(configKey)
	}
	// Push the updates if any configs were successfully processed
	if shouldFullPush && m.xdsUpdater != nil {
		IngressLog.Infof("Full push triggered for secret %s", secretKey)
		m.xdsUpdater.ConfigUpdate(&istiomodel.PushRequest{
			Full: true,
		})
	} else if configsUpdated.Len() > 0 && m.xdsUpdater != nil {
		IngressLog.Infof("Secret %s changed, updating %d dependent configs", secretKey, len(configKeys))
		m.xdsUpdater.ConfigUpdate(&istiomodel.PushRequest{
			Full:           false,
			ConfigsUpdated: configsUpdated,
		})
	}
	return
}
