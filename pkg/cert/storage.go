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
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CertificatesPrefix              = "/certificates"
	ConfigmapStoreCertficatesPrefix = "higress-cert-store-certificates-"
	ConfigmapStoreDefaultName       = "higress-cert-store-default"
)

var _ certmagic.Storage = (*ConfigmapStorage)(nil)

type ConfigmapStorage struct {
	namespace string
	client    kubernetes.Interface
	mux       sync.RWMutex
}

type HashValue struct {
	K string `json:"k,omitempty"`
	V []byte `json:"v,omitempty"`
}

func NewConfigmapStorage(namespace string, client kubernetes.Interface) (certmagic.Storage, error) {
	storage := &ConfigmapStorage{
		namespace: namespace,
		client:    client,
	}
	return storage, nil
}

// Exists returns true if key exists in s.
func (s *ConfigmapStorage) Exists(_ context.Context, key string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	cm, err := s.getConfigmapStoreByKey(key)
	if err != nil {
		return false
	}
	if cm.Data == nil {
		return false
	}

	hashKey := fastHash([]byte(key))
	if _, ok := cm.Data[hashKey]; ok {
		return true
	}
	return false
}

// Store saves value at key.
func (s *ConfigmapStorage) Store(_ context.Context, key string, value []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	cm, err := s.getConfigmapStoreByKey(key)
	if err != nil {
		return err
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string, 0)
	}

	hashKey := fastHash([]byte(key))
	hashV := &HashValue{
		K: key,
		V: value,
	}
	bytes, err := json.Marshal(hashV)
	if err != nil {
		return err
	}
	cm.Data[hashKey] = string(bytes)
	return s.updateConfigmap(cm)
}

// Load retrieves the value at key.
func (s *ConfigmapStorage) Load(_ context.Context, key string) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var value []byte
	cm, err := s.getConfigmapStoreByKey(key)
	if err != nil {
		return value, err
	}
	if cm.Data == nil {
		return value, fs.ErrNotExist
	}

	hashKey := fastHash([]byte(key))
	if v, ok := cm.Data[hashKey]; ok {
		hV := &HashValue{}
		err = json.Unmarshal([]byte(v), hV)
		if err != nil {
			return value, err
		}
		return hV.V, nil
	}
	return value, fs.ErrNotExist
}

// Delete deletes the value at key.
func (s *ConfigmapStorage) Delete(_ context.Context, key string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	cm, err := s.getConfigmapStoreByKey(key)
	if err != nil {
		return err
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string, 0)
	}
	hashKey := fastHash([]byte(key))
	delete(cm.Data, hashKey)
	return s.updateConfigmap(cm)
}

// List returns all keys that match the prefix.
// If the prefix is "/certificates", it retrieves all ConfigMaps, otherwise only one.
func (s *ConfigmapStorage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	var keys []string
	var configmapKeys []string
	visitedDirs := make(map[string]struct{})

	// Check if the prefix corresponds to a specific key
	hashPrefix := fastHash([]byte(prefix))
	if strings.HasPrefix(prefix, CertificatesPrefix) {
		// If the prefix is "/certificates", get all ConfigMaps and traverse each one
		// List all ConfigMaps in the namespace with label higress.io/cert-https=true
		configmaps, err := s.client.CoreV1().ConfigMaps(s.namespace).List(ctx, metav1.ListOptions{FieldSelector: "metadata.annotations['higress.io/cert-https'] == 'true'"})
		if err != nil {
			return keys, err
		}

		for _, cm := range configmaps.Items {
			// Check if the ConfigMap name starts with the expected prefix
			if strings.HasPrefix(cm.Name, ConfigmapStoreCertficatesPrefix) {
				// Add the keys from Data field to the list
				for _, v := range cm.Data {
					// Unmarshal the value into hashValue struct
					var hv HashValue
					if err := json.Unmarshal([]byte(v), &hv); err != nil {
						return nil, err
					}
					// Check if the key starts with the specified prefix
					if strings.HasPrefix(hv.K, prefix) {
						// Add the key to the list
						configmapKeys = append(configmapKeys, hv.K)
					}
				}
			}
		}
	} else {
		// If not starting with "/certificates", get the specific ConfigMap
		cm, err := s.getConfigmapStoreByKey(prefix)
		if err != nil {
			return keys, err
		}

		if _, ok := cm.Data[hashPrefix]; ok {
			// The prefix corresponds to a specific key, add it to the list
			configmapKeys = append(configmapKeys, prefix)
		} else {
			// The prefix is considered a directory
			for _, v := range cm.Data {
				// Unmarshal the value into hashValue struct
				var hv HashValue
				if err := json.Unmarshal([]byte(v), &hv); err != nil {
					return nil, err
				}
				// Check if the key starts with the specified prefix
				if strings.HasPrefix(hv.K, prefix) {
					// Add the key to the list
					configmapKeys = append(configmapKeys, hv.K)
				}
			}
		}
	}

	// return all
	if recursive {
		return configmapKeys, nil
	}

	// only return sub dirs
	for _, key := range configmapKeys {
		subPath := strings.TrimPrefix(strings.ReplaceAll(key, prefix, ""), "/")
		paths := strings.Split(subPath, "/")
		if len(paths) > 0 {
			subDir := path.Join(prefix, paths[0])
			if _, ok := visitedDirs[subDir]; !ok {
				keys = append(keys, subDir)
			}
			visitedDirs[subDir] = struct{}{}
		}
	}

	return keys, nil
}

// Stat returns information about key. only support for no certificates path
func (s *ConfigmapStorage) Stat(_ context.Context, key string) (certmagic.KeyInfo, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	// Create a new KeyInfo struct
	info := certmagic.KeyInfo{}

	// Get the ConfigMap containing the keys
	cm, err := s.getConfigmapStoreByKey(key)
	if err != nil {
		return info, err
	}

	// Check if the key exists in the ConfigMap
	hashKey := fastHash([]byte(key))
	if data, ok := cm.Data[hashKey]; ok {
		// The key exists, populate the KeyInfo struct
		info.Key = key
		info.Modified = time.Now() // Since we're not tracking modification time in ConfigMap
		info.Size = int64(len(data))
		info.IsTerminal = true
	} else {
		// Check if there are other keys with the same prefix
		prefixKeys := make([]string, 0)
		for _, v := range cm.Data {
			var hv HashValue
			if err := json.Unmarshal([]byte(v), &hv); err != nil {
				return info, err
			}
			// Check if the key starts with the specified prefix
			if strings.HasPrefix(hv.K, key) {
				// Add the key to the list
				prefixKeys = append(prefixKeys, hv.K)
			}
		}
		// If there are multiple keys with the same prefix, then it's not a terminal node
		if len(prefixKeys) > 0 {
			info.Key = key
			info.IsTerminal = false
		} else {
			return info, fmt.Errorf("prefix '%s' is not existed", key)
		}
	}
	return info, nil
}

// Lock obtains a lock named by the given name. It blocks
// until the lock can be obtained or an error is returned.
func (s *ConfigmapStorage) Lock(ctx context.Context, name string) error {
	return nil
}

// Unlock releases the lock for name.
func (s *ConfigmapStorage) Unlock(_ context.Context, name string) error {
	return nil
}

func (s *ConfigmapStorage) String() string {
	return "ConfigmapStorage"
}

func (s *ConfigmapStorage) getConfigmapStoreNameByKey(key string) string {
	parts := strings.SplitN(key, "/", 10)
	if len(parts) >= 4 && parts[1] == "certificates" {
		domain := strings.TrimSuffix(parts[3], ".crt")
		domain = strings.TrimSuffix(domain, ".key")
		domain = strings.TrimSuffix(domain, ".json")
		issuerKey := parts[2]
		return ConfigmapStoreCertficatesPrefix + fastHash([]byte(issuerKey+domain))
	}
	return ConfigmapStoreDefaultName
}

func (s *ConfigmapStorage) getConfigmapStoreByKey(key string) (*v1.ConfigMap, error) {
	configmapName := s.getConfigmapStoreNameByKey(key)
	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(context.Background(), configmapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Save default ConfigMap
			cm = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   s.namespace,
					Name:        configmapName,
					Annotations: map[string]string{"higress.io/cert-https": "true"},
				},
			}
			_, err = s.client.CoreV1().ConfigMaps(s.namespace).Create(context.Background(), cm, metav1.CreateOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return cm, nil
}

// updateConfigmap adds or updates the annotation higress.io/cert-https to true.
func (s *ConfigmapStorage) updateConfigmap(configmap *v1.ConfigMap) error {
	if configmap.ObjectMeta.Annotations == nil {
		configmap.ObjectMeta.Annotations = make(map[string]string)
	}
	configmap.ObjectMeta.Annotations["higress.io/cert-https"] = "true"

	_, err := s.client.CoreV1().ConfigMaps(configmap.Namespace).Update(context.Background(), configmap, metav1.UpdateOptions{})
	return err
}
