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
	"errors"
	"fmt"
	"github.com/alibaba/higress/pkg/ingress/kube/util"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validGzip(t *testing.T) {

	tests := []struct {
		name    string
		gzip    *Gzip
		wantErr error
	}{
		{
			name: "default",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: nil,
		},
		{
			name:    "nil",
			gzip:    nil,
			wantErr: nil,
		},

		{
			name: "no content type",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: errors.New("content type can not be empty"),
		},

		{
			name: "MinContentLength less than zero",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    0,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: errors.New("minContentLength can not be less than zero"),
		},

		{
			name: "MemoryLevel less than 1",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: errors.New("memory level need be between 1 and 9"),
		},

		{
			name: "WindowBits less than 9",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          8,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: errors.New("window bits need be between 9 and 15"),
		},

		{
			name: "ChunkSize less than zero",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: errors.New("chunk size need be large than zero"),
		},

		{
			name: "CompressionLevel is not right",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSIONA",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: fmt.Errorf("compressionLevel need be one of %s", compressionLevelValues),
		},

		{
			name: "CompressionStrategy is not right",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGYA",
			},
			wantErr: fmt.Errorf("compressionStrategy need be one of %s", compressionStrategyValues),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validGzip(tt.gzip); err != nil {
				assert.Equal(t, tt.wantErr, err)
			}
		})
	}
}

func Test_compareGzip(t *testing.T) {
	tests := []struct {
		name       string
		old        *Gzip
		new        *Gzip
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
			old: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			new:        nil,
			wantResult: ResultDelete,
			wantErr:    nil,
		},
		{
			name: "compare result equal",
			old: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			new: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantResult: ResultNothing,
			wantErr:    nil,
		},
		{
			name: "compare result replace",
			old: &Gzip{
				Enable:              false,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			new: &Gzip{
				Enable:              true,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantResult: ResultReplace,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compareGzip(tt.old, tt.new)
			assert.Equal(t, tt.wantResult, result)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_deepCopyGzip(t *testing.T) {

	tests := []struct {
		name     string
		gzip     *Gzip
		wantGzip *Gzip
		wantErr  error
	}{
		{
			name: "deep copy case 1",
			gzip: &Gzip{
				Enable:              false,
				MinContentLength:    102,
				ContentType:         []string{"text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: false,
				MemoryLevel:         6,
				WindowBits:          11,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_SPEED",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantGzip: &Gzip{
				Enable:              false,
				MinContentLength:    102,
				ContentType:         []string{"text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: false,
				MemoryLevel:         6,
				WindowBits:          11,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_SPEED",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: nil,
		},

		{
			name: "deep copy case 2",
			gzip: &Gzip{
				Enable:              true,
				MinContentLength:    102,
				ContentType:         []string{"text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         6,
				WindowBits:          11,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_SPEED",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantGzip: &Gzip{
				Enable:              true,
				MinContentLength:    102,
				ContentType:         []string{"text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         6,
				WindowBits:          11,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_SPEED",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gzip, err := deepCopyGzip(tt.gzip)
			assert.Equal(t, tt.wantGzip, gzip)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestGzipController_AddOrUpdateHigressConfig(t *testing.T) {
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
		wantGzip      *Gzip
	}{
		{
			name: "default",
			old: &HigressConfig{
				Gzip: NewDefaultGzip(),
			},
			new: &HigressConfig{
				Gzip: NewDefaultGzip(),
			},
			wantErr:       nil,
			wantEventPush: "default",
			wantGzip:      NewDefaultGzip(),
		},
		{
			name: "replace and push 1",
			old: &HigressConfig{
				Gzip: NewDefaultGzip(),
			},
			new: &HigressConfig{
				Gzip: &Gzip{
					Enable:              true,
					MinContentLength:    1024,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantGzip: &Gzip{
				Enable:              true,
				MinContentLength:    1024,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},

		{
			name: "replace and push 2",
			old: &HigressConfig{
				Gzip: &Gzip{
					Enable:              true,
					MinContentLength:    1024,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			new: &HigressConfig{
				Gzip: &Gzip{
					Enable:              true,
					MinContentLength:    2048,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantGzip: &Gzip{
				Enable:              true,
				MinContentLength:    2048,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},

		{
			name: "replace and push 3",
			old: &HigressConfig{
				Gzip: &Gzip{
					Enable:              true,
					MinContentLength:    1024,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			new: &HigressConfig{
				Gzip: &Gzip{
					Enable:              false,
					MinContentLength:    2048,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantGzip: &Gzip{
				Enable:              false,
				MinContentLength:    2048,
				ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
				DisableOnEtagHeader: true,
				MemoryLevel:         5,
				WindowBits:          12,
				ChunkSize:           4096,
				CompressionLevel:    "BEST_COMPRESSION",
				CompressionStrategy: "DEFAULT_STRATEGY",
			},
		},
		{
			name: "delete and push",
			old: &HigressConfig{
				Gzip: &Gzip{
					Enable:              true,
					MinContentLength:    1024,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			},
			new: &HigressConfig{
				Gzip: nil,
			},
			wantErr:       nil,
			wantEventPush: "push",
			wantGzip:      NewDefaultGzip(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGzipController("higress-system")
			g.eventHandler = defaultHandler
			eventPush = "default"
			err := g.AddOrUpdateHigressConfig(defaultName, tt.old, tt.new)
			assert.Equal(t, tt.wantEventPush, eventPush)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantGzip, g.GetGzip())
		})
	}
}
