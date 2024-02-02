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

package configmap

import (
	"testing"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"github.com/stretchr/testify/assert"
)

func Test_validGlobal(t *testing.T) {
	tests := []struct {
		name    string
		global  *Global
		wantErr error
	}{
		{
			name:    "default",
			global:  NewDefaultGlobalOption(),
			wantErr: nil,
		},
		{
			name:    "nil",
			global:  nil,
			wantErr: nil,
		},
		{
			name: "downstream nil",
			global: &Global{
				Downstream:           nil,
				Upstream:             NewDefaultUpStream(),
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validGlobal(tt.global)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_compareGlobal(t *testing.T) {
	tests := []struct {
		name    string
		old     *Global
		new     *Global
		want    Result
		wantErr error
	}{
		{
			name:    "compare both nil",
			old:     nil,
			new:     nil,
			want:    ResultNothing,
			wantErr: nil,
		},
		{
			name:    "compare new nil 1",
			old:     NewDefaultGlobalOption(),
			new:     nil,
			want:    ResultDelete,
			wantErr: nil,
		},
		{
			name:    "compare new nil 2",
			old:     NewDefaultGlobalOption(),
			new:     &Global{},
			want:    ResultDelete,
			wantErr: nil,
		},
		{
			name:    "compare result equal",
			old:     NewDefaultGlobalOption(),
			new:     NewDefaultGlobalOption(),
			want:    ResultNothing,
			wantErr: nil,
		},
		{
			name: "compare result not equal",
			old:  NewDefaultGlobalOption(),
			new: &Global{
				Downstream: &Downstream{
					IdleTimeout: 1,
				},
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
			want:    ResultReplace,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compareGlobal(tt.old, tt.new)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func Test_deepCopyGlobal(t *testing.T) {
	tests := []struct {
		name    string
		global  *Global
		want    *Global
		wantErr error
	}{
		{
			name:    "deep copy 1",
			global:  NewDefaultGlobalOption(),
			want:    NewDefaultGlobalOption(),
			wantErr: nil,
		},
		{
			name: "deep copy 2",
			global: &Global{
				Downstream: &Downstream{
					IdleTimeout:            0,
					MaxRequestHeadersKb:    9600,
					ConnectionBufferLimits: 4096,
					Http2:                  NewDefaultHttp2(),
				},
				Upstream: &Upstream{
					IdleTimeout: 10,
				},
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
			want: &Global{
				Downstream: &Downstream{
					IdleTimeout:            0,
					MaxRequestHeadersKb:    9600,
					ConnectionBufferLimits: 4096,
					Http2:                  NewDefaultHttp2(),
				},
				Upstream: &Upstream{
					IdleTimeout: 10,
				},
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global, err := deepCopyGlobal(tt.global)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, global)
		})
	}
}

func Test_AddOrUpdateHigressConfig(t *testing.T) {
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
		wantGlobal    *Global
	}{
		{
			name:          "default",
			new:           NewDefaultHigressConfig(),
			old:           NewDefaultHigressConfig(),
			wantErr:       nil,
			wantEventPush: "default",
			wantGlobal:    NewDefaultGlobalOption(),
		},
		{
			name: "replace and push",
			old:  NewDefaultHigressConfig(),
			new: &HigressConfig{
				Downstream: &Downstream{
					IdleTimeout:            1,
					MaxRequestHeadersKb:    defaultMaxRequestHeadersKb,
					ConnectionBufferLimits: defaultConnectionBufferLimits,
					Http2:                  NewDefaultHttp2(),
				},
				Upstream: &Upstream{
					IdleTimeout: 10,
				},
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantGlobal: &Global{
				Downstream: &Downstream{
					IdleTimeout:            1,
					MaxRequestHeadersKb:    defaultMaxRequestHeadersKb,
					ConnectionBufferLimits: defaultConnectionBufferLimits,
					Http2:                  NewDefaultHttp2(),
				},
				Upstream: &Upstream{
					IdleTimeout: 10,
				},
				AddXRealIpHeader:     true,
				DisableXEnvoyHeaders: true,
			},
		},
		{
			name: "delete and push",
			old: &HigressConfig{
				Downstream:           NewDefaultDownstream(),
				Upstream:             NewDefaultUpStream(),
				AddXRealIpHeader:     defaultAddXRealIpHeader,
				DisableXEnvoyHeaders: defaultDisableXEnvoyHeaders,
			},
			new:           &HigressConfig{},
			wantErr:       nil,
			wantEventPush: "push",
			wantGlobal: &Global{
				Downstream:           NewDefaultDownstream(),
				Upstream:             NewDefaultUpStream(),
				AddXRealIpHeader:     defaultAddXRealIpHeader,
				DisableXEnvoyHeaders: defaultDisableXEnvoyHeaders,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGlobalOptionController("higress-namespace")
			g.eventHandler = defaultHandler
			eventPush = "default"
			err := g.AddOrUpdateHigressConfig(defaultName, tt.old, tt.new)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantEventPush, eventPush)
			assert.Equal(t, tt.wantGlobal, g.GetGlobal())
		})
	}
}
