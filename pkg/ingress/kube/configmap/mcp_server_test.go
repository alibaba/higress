// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmap

import (
	"errors"
	"testing"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"github.com/stretchr/testify/assert"
)

func Test_validMcpServer(t *testing.T) {
	tests := []struct {
		name    string
		mcp     *McpServer
		wantErr error
	}{
		{
			name: "default",
			mcp: &McpServer{
				Enable:    false,
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			wantErr: nil,
		},
		{
			name:    "nil",
			mcp:     nil,
			wantErr: nil,
		},
		{
			name: "enabled but no redis config",
			mcp: &McpServer{
				Enable:    true,
				Redis:     nil,
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			wantErr: errors.New("redis config cannot be empty when mcp server is enabled"),
		},
		{
			name: "valid config with redis",
			mcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				SsePathSuffix: "/sse",
				MatchList: []*MatchRule{
					{
						MatchRuleDomain: "*",
						MatchRulePath:   "*",
						MatchRuleType:   "exact",
					},
				},
				Servers: []*SSEServer{
					{
						Name: "test-server",
						Path: "/test",
						Type: "test",
						Config: map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validMcpServer(tt.mcp)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_compareMcpServer(t *testing.T) {
	tests := []struct {
		name       string
		old        *McpServer
		new        *McpServer
		wantResult Result
		wantErr    error
	}{
		{
			name:       "compare both nil",
			old:        nil,
			new:        nil,
			wantResult: ResultNothing,
			wantErr:    nil,
		},
		{
			name: "compare result delete",
			old: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "localhost:6379",
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			new:        nil,
			wantResult: ResultDelete,
			wantErr:    nil,
		},
		{
			name: "compare result equal",
			old: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "localhost:6379",
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			new: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "localhost:6379",
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			wantResult: ResultNothing,
			wantErr:    nil,
		},
		{
			name: "compare result replace",
			old: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "localhost:6379",
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			new: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "redis:6379",
				},
				MatchList: []*MatchRule{
					{
						MatchRuleDomain: "*",
						MatchRulePath:   "/test",
						MatchRuleType:   "exact",
					},
				},
				Servers: []*SSEServer{},
			},
			wantResult: ResultReplace,
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compareMcpServer(tt.old, tt.new)
			assert.Equal(t, tt.wantResult, result)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_deepCopyMcpServer(t *testing.T) {
	tests := []struct {
		name    string
		mcp     *McpServer
		wantMcp *McpServer
		wantErr error
	}{
		{
			name: "deep copy with redis only",
			mcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			wantMcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				MatchList: []*MatchRule{},
				Servers:   []*SSEServer{},
			},
			wantErr: nil,
		},
		{
			name: "deep copy with full config",
			mcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				SsePathSuffix: "/sse",
				MatchList: []*MatchRule{
					{
						MatchRuleDomain: "*",
						MatchRulePath:   "*",
						MatchRuleType:   "exact",
					},
				},
				Servers: []*SSEServer{
					{
						Name: "test-server",
						Path: "/test",
						Type: "test",
						Config: map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			wantMcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				SsePathSuffix: "/sse",
				MatchList: []*MatchRule{
					{
						MatchRuleDomain: "*",
						MatchRulePath:   "*",
						MatchRuleType:   "exact",
					},
				},
				Servers: []*SSEServer{
					{
						Name: "test-server",
						Path: "/test",
						Type: "test",
						Config: map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcp, err := deepCopyMcpServer(tt.mcp)
			assert.Equal(t, tt.wantMcp, mcp)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestMcpServerController_AddOrUpdateHigressConfig(t *testing.T) {
	eventPush := "default"
	defaultHandler := func(name string) {
		eventPush = "push"
	}

	defaultName := util.ClusterNamespacedName{}

	tests := []struct {
		name          string
		old           *HigressConfig
		new           *HigressConfig
		wantErr       error
		wantEventPush string
		wantMcp       *McpServer
	}{
		{
			name: "default",
			old: &HigressConfig{
				McpServer: NewDefaultMcpServer(),
			},
			new: &HigressConfig{
				McpServer: NewDefaultMcpServer(),
			},
			wantErr:       nil,
			wantEventPush: "default",
			wantMcp:       NewDefaultMcpServer(),
		},
		{
			name: "replace and push - enable mcp server",
			old: &HigressConfig{
				McpServer: NewDefaultMcpServer(),
			},
			new: &HigressConfig{
				McpServer: &McpServer{
					Enable: true,
					Redis: &RedisConfig{
						Address:  "localhost:6379",
						Username: "default",
						Password: "password",
						DB:       0,
					},
					Servers:   []*SSEServer{},
					MatchList: []*MatchRule{},
				},
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantMcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address:  "localhost:6379",
					Username: "default",
					Password: "password",
					DB:       0,
				},
				Servers:   []*SSEServer{},
				MatchList: []*MatchRule{},
			},
		},
		{
			name: "replace and push - update config",
			old: &HigressConfig{
				McpServer: &McpServer{
					Enable: true,
					Redis: &RedisConfig{
						Address: "localhost:6379",
					},
					Servers:   []*SSEServer{},
					MatchList: []*MatchRule{},
				},
			},
			new: &HigressConfig{
				McpServer: &McpServer{
					Enable: true,
					Redis: &RedisConfig{
						Address: "redis:6379",
					},
					Servers:   []*SSEServer{},
					MatchList: []*MatchRule{},
				},
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantMcp: &McpServer{
				Enable: true,
				Redis: &RedisConfig{
					Address: "redis:6379",
				},
				Servers:   []*SSEServer{},
				MatchList: []*MatchRule{},
			},
		},
		{
			name: "delete and push",
			old: &HigressConfig{
				McpServer: &McpServer{
					Enable: true,
					Redis: &RedisConfig{
						Address: "localhost:6379",
					},
					Servers:   []*SSEServer{},
					MatchList: []*MatchRule{},
				},
			},
			new: &HigressConfig{
				McpServer: nil,
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantMcp:       NewDefaultMcpServer(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMcpServerController("higress-system")
			m.eventHandler = defaultHandler
			eventPush = "default"
			err := m.AddOrUpdateHigressConfig(defaultName, tt.old, tt.new)
			assert.Equal(t, tt.wantEventPush, eventPush)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantMcp, m.GetMcpServer())
		})
	}
}
