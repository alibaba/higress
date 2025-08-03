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
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"github.com/stretchr/testify/assert"
	istiomodel "istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	"istio.io/istio/pkg/config/schema/kind"
)

type mockXdsUpdater struct {
	lastPushRequest *istiomodel.PushRequest
}

func (m *mockXdsUpdater) EDSUpdate(shard istiomodel.ShardKey, hostname string, namespace string, entry []*istiomodel.IstioEndpoint) {
	//TODO implement me
	panic("implement me")
}

func (m *mockXdsUpdater) EDSCacheUpdate(shard istiomodel.ShardKey, hostname string, namespace string, entry []*istiomodel.IstioEndpoint) {
	//TODO implement me
	panic("implement me")
}

func (m *mockXdsUpdater) SvcUpdate(shard istiomodel.ShardKey, hostname string, namespace string, event istiomodel.Event) {
	//TODO implement me
	panic("implement me")
}

func (m *mockXdsUpdater) ProxyUpdate(clusterID cluster.ID, ip string) {
	//TODO implement me
	panic("implement me")
}

func (m *mockXdsUpdater) RemoveShard(shardKey istiomodel.ShardKey) {
	//TODO implement me
	panic("implement me")
}

func (m *mockXdsUpdater) ConfigUpdate(req *istiomodel.PushRequest) {
	m.lastPushRequest = req
}

func TestSecretConfigMgr(t *testing.T) {
	updater := &mockXdsUpdater{}
	mgr := NewSecretConfigMgr(updater)

	// Test AddConfig
	t.Run("AddConfig", func(t *testing.T) {
		wasmPlugin := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Name:             "test-plugin",
				Namespace:        "default",
			},
		}

		err := mgr.AddConfig("default/test-secret", wasmPlugin)
		assert.NoError(t, err)
		assert.True(t, mgr.IsSecretWatched("default/test-secret"))

		configs := mgr.GetConfigsForSecret("default/test-secret")
		assert.Len(t, configs, 1)
		assert.Equal(t, kind.WasmPlugin, configs[0].Kind)
		assert.Equal(t, "test-plugin", configs[0].Name)
		assert.Equal(t, "default", configs[0].Namespace)
	})

	// Test DeleteConfig
	t.Run("DeleteConfig", func(t *testing.T) {
		wasmPlugin := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Name:             "test-plugin",
				Namespace:        "default",
			},
		}

		err := mgr.DeleteConfig(wasmPlugin)
		assert.NoError(t, err)
		assert.False(t, mgr.IsSecretWatched("default/test-secret"))
		assert.Empty(t, mgr.GetConfigsForSecret("default/test-secret"))
	})

	// Test HandleSecretChange
	t.Run("HandleSecretChange", func(t *testing.T) {
		// Add a config first
		wasmPlugin := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.WasmPlugin,
				Name:             "test-plugin",
				Namespace:        "default",
			},
		}
		err := mgr.AddConfig("default/test-secret", wasmPlugin)
		assert.NoError(t, err)

		// Test secret change
		secretName := util.ClusterNamespacedName{
			NamespacedName: types.NamespacedName{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		mgr.HandleSecretChange(secretName)
		assert.NotNil(t, updater.lastPushRequest)
		assert.True(t, updater.lastPushRequest.Full)
	})

	// Test full push for secret update
	t.Run("FullPushForSecretUpdate", func(t *testing.T) {
		// Add a secret config
		secretConfig := &config.Config{
			Meta: config.Meta{
				GroupVersionKind: gvk.Secret,
				Name:             "test-secret",
				Namespace:        "default",
			},
		}
		err := mgr.AddConfig("default/test-secret", secretConfig)
		assert.NoError(t, err)

		// Update the secret
		secretName := util.ClusterNamespacedName{
			NamespacedName: types.NamespacedName{
				Name:      "test-secret",
				Namespace: "default",
			},
		}

		mgr.HandleSecretChange(secretName)
		assert.NotNil(t, updater.lastPushRequest)
		assert.True(t, updater.lastPushRequest.Full)
	})
}
